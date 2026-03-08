package main

import (
	"strings"
	"time"

	"dxcluster/spot"
)

type outputDeliveryPlan struct {
	archivePeerAllowMed bool
	allowFast           bool
	allowMed            bool
	allowSlow           bool
	telnetDeliverNow    bool
	familySnapshot      spot.ResolverSnapshot
	familySnapshotOK    bool
}

func (p *outputPipeline) startStabilizerReleaseLoop() {
	go func() {
		releaseCh := p.telnetStabilizer.ReleaseChan()
		for envelope := range releaseCh {
			p.handleStabilizerRelease(envelope)
		}
	}()
}

func (p *outputPipeline) handleStabilizerRelease(envelope *telnetStabilizerEnvelope) {
	if envelope == nil || envelope.spot == nil {
		return
	}
	delayed := envelope.spot
	checksCompleted := envelope.checksCompleted + 1
	ctyDB := p.ctyLookup()
	var knownCallset *spot.KnownCallsigns
	if p.knownCalls != nil {
		knownCallset = p.knownCalls.Load()
	}
	now := time.Now().UTC()
	resolverEvidence := spot.ResolverEvidence{Key: envelope.resolverKey}
	hasResolverEvidence := envelope.hasResolverKey
	if !hasResolverEvidence {
		resolverEvidence, hasResolverEvidence = buildResolverEvidenceSnapshot(delayed, p.correctionCfg, p.adaptiveMinReports, now)
	}
	resolverEvidenceEnqueued := envelope.evidenceEnqueued
	if maybeApplyResolverCorrection(
		delayed,
		p.signalResolver,
		resolverEvidence,
		hasResolverEvidence,
		p.correctionCfg,
		ctyDB,
		p.metaCache,
		p.tracker,
		p.dash,
		p.recentBandStore,
		knownCallset,
		p.adaptiveMinReports,
		p.spotterReliability,
		p.spotterReliabilityCW,
		p.spotterReliabilityRTTY,
		p.confusionModel,
	) {
		return
	}
	applyKnownCallFloor(delayed, p.knownCalls, p.recentBandStore, p.customSCPStore, p.correctionCfg)
	delayed.RefreshBeaconFlag()
	delayed.EnsureNormalized()
	if applyLicenseGate(delayed, ctyDB, p.metaCache, p.unlicensedReporter) {
		return
	}
	delaySnapshot, delaySnapshotOK := spot.ResolverSnapshot{}, false
	if hasResolverEvidence && p.signalResolver != nil {
		delaySnapshot, delaySnapshotOK = p.signalResolver.Lookup(resolverEvidence.Key)
	}
	delayDecision := evaluateTelnetStabilizerDelay(delayed, p.recentBandStore, p.correctionCfg, time.Now().UTC(), delaySnapshot, delaySnapshotOK)
	if shouldRetryTelnetByStabilizer(delayDecision, checksCompleted) {
		if p.telnetStabilizer.EnqueueWithContext(
			delayed,
			checksCompleted,
			delayDecision.Reason.String(),
			resolverEvidence.Key,
			hasResolverEvidence,
			resolverEvidenceEnqueued,
		) {
			if p.tracker != nil {
				p.tracker.IncrementStabilizerHeld()
				p.tracker.IncrementStabilizerHeldReason(delayDecision.Reason.String())
			}
			return
		}
		if p.tracker != nil {
			p.tracker.IncrementStabilizerOverflowRelease()
		}
		delayDecision.ShouldDelay = false
	}
	if !shouldRecordRecentBandAfterStabilizerDelay(p.correctionCfg.StabilizerTimeoutAction, delayDecision.ShouldDelay) {
		if p.tracker != nil {
			p.tracker.IncrementStabilizerSuppressedTimeout()
			p.tracker.IncrementStabilizerSuppressedTimeoutReason(delayDecision.Reason.String())
			p.tracker.ObserveStabilizerGlyphTurns(delayed.Confidence, checksCompleted)
		}
		return
	}
	recordRecentBandObservation(delayed, p.recentBandStore, p.customSCPStore, p.correctionCfg)
	allowFast, allowMed, allowSlow := p.computeTelnetAllows(delayed)
	if !allowFast && !allowMed && !allowSlow {
		p.telnet.DeliverSelfSpot(delayed)
		return
	}
	if p.familySuppressor != nil && p.familySuppressor.ShouldSuppressWithResolver(delayed, p.correctionCfg, time.Now().UTC(), delaySnapshot, delaySnapshotOK) {
		p.telnet.DeliverSelfSpot(delayed)
		return
	}
	if p.tracker != nil {
		p.tracker.IncrementStabilizerReleasedDelayed()
		p.tracker.IncrementStabilizerReleasedDelayedReason(stabilizerReleaseReason(delayDecision, envelope.delayReason))
		p.tracker.ObserveStabilizerGlyphTurns(delayed.Confidence, checksCompleted)
	}
	if p.lastOutput != nil {
		p.lastOutput.Store(time.Now().UTC().UnixNano())
	}
	p.telnet.BroadcastSpot(delayed, allowFast, allowMed, allowSlow)
}

func (p *outputPipeline) computeTelnetAllows(s *spot.Spot) (bool, bool, bool) {
	allowFast := true
	if p.secondaryFast != nil {
		allowFast = p.secondaryFast.ShouldForward(s)
	}
	allowMed := true
	if p.secondaryMed != nil {
		allowMed = p.secondaryMed.ShouldForward(s)
	}
	allowSlow := true
	if p.secondarySlow != nil {
		allowSlow = p.secondarySlow.ShouldForward(s)
	}
	fallbackAllowed := allowFast
	if p.secondaryFast == nil {
		if p.secondaryMed != nil {
			fallbackAllowed = allowMed
		} else if p.secondarySlow != nil {
			fallbackAllowed = allowSlow
		}
	}
	if p.secondaryFast == nil {
		allowFast = fallbackAllowed
	}
	if p.secondaryMed == nil {
		allowMed = fallbackAllowed
	}
	if p.secondarySlow == nil {
		allowSlow = fallbackAllowed
	}
	return allowFast, allowMed, allowSlow
}

func (p *outputPipeline) deliverSpot(ctx *outputSpotContext) {
	if p.secondaryStage != nil {
		p.secondaryStage.Add(1)
	}
	plan, ok := p.resolveDeliveryPlan(ctx)
	if !ok {
		return
	}
	p.updateGridCache(ctx.spot)
	p.emitSpot(ctx.spot, plan)
}

func (p *outputPipeline) resolveDeliveryPlan(ctx *outputSpotContext) (outputDeliveryPlan, bool) {
	s := ctx.spot
	plan := outputDeliveryPlan{
		archivePeerAllowMed: true,
		allowFast:           true,
		allowMed:            true,
		allowSlow:           true,
		telnetDeliverNow:    p.telnet != nil,
	}
	if p.archivePeerSecondaryMed != nil {
		plan.archivePeerAllowMed = p.archivePeerSecondaryMed.ShouldForward(s)
	}
	if !p.stabilizerEnabled {
		plan.allowFast, plan.allowMed, plan.allowSlow = p.computeTelnetAllows(s)
		plan.archivePeerAllowMed = plan.allowMed
		recordRecentBandObservation(s, p.recentBandStore, p.customSCPStore, p.correctionCfg)
		if ctx.hasStabilizerResolverKey && p.signalResolver != nil {
			plan.familySnapshot, plan.familySnapshotOK = p.signalResolver.Lookup(ctx.stabilizerResolverKey)
		}
		if !plan.allowFast && !plan.allowMed && !plan.allowSlow {
			p.telnet.DeliverSelfSpot(s)
			return plan, false
		}
		if p.familySuppressor != nil && p.familySuppressor.ShouldSuppressWithResolver(s, p.correctionCfg, time.Now().UTC(), plan.familySnapshot, plan.familySnapshotOK) {
			p.telnet.DeliverSelfSpot(s)
			return plan, false
		}
		return plan, true
	}

	plan.telnetDeliverNow = false
	delaySnapshot, delaySnapshotOK := spot.ResolverSnapshot{}, false
	if ctx.hasStabilizerResolverKey && p.signalResolver != nil {
		delaySnapshot, delaySnapshotOK = p.signalResolver.Lookup(ctx.stabilizerResolverKey)
	}
	plan.familySnapshot, plan.familySnapshotOK = delaySnapshot, delaySnapshotOK
	delayDecision := evaluateTelnetStabilizerDelay(s, p.recentBandStore, p.correctionCfg, time.Now().UTC(), delaySnapshot, delaySnapshotOK)
	if delayDecision.ShouldDelay {
		delayed := cloneSpotForTelnetStabilizer(s)
		if delayed != nil && p.telnetStabilizer.EnqueueWithContext(
			delayed,
			0,
			delayDecision.Reason.String(),
			ctx.stabilizerResolverKey,
			ctx.hasStabilizerResolverKey,
			ctx.stabilizerEvidenceEnqueued,
		) {
			if p.tracker != nil {
				p.tracker.IncrementStabilizerHeld()
				p.tracker.IncrementStabilizerHeldReason(delayDecision.Reason.String())
			}
		} else {
			plan.allowFast, plan.allowMed, plan.allowSlow = p.computeTelnetAllows(s)
			plan.telnetDeliverNow = true
			if p.tracker != nil {
				p.tracker.IncrementStabilizerOverflowRelease()
				if stabilizerImmediateCountEligible(s) {
					p.tracker.IncrementStabilizerReleasedImmediate()
					p.tracker.IncrementStabilizerReleasedImmediateReason(delayDecision.Reason.String())
				}
			}
		}
	} else {
		plan.allowFast, plan.allowMed, plan.allowSlow = p.computeTelnetAllows(s)
		plan.telnetDeliverNow = true
		if p.tracker != nil && stabilizerImmediateCountEligible(s) {
			p.tracker.IncrementStabilizerReleasedImmediate()
			p.tracker.IncrementStabilizerReleasedImmediateReason(stabilizerDelayReasonNone.String())
		}
	}
	if shouldRecordRecentBandInMainLoop(p.stabilizerEnabled, !plan.telnetDeliverNow) {
		recordRecentBandObservation(s, p.recentBandStore, p.customSCPStore, p.correctionCfg)
	}
	if plan.telnetDeliverNow && !plan.allowFast && !plan.allowMed && !plan.allowSlow {
		p.telnet.DeliverSelfSpot(s)
		plan.telnetDeliverNow = false
	}
	if plan.telnetDeliverNow && p.familySuppressor != nil && p.familySuppressor.ShouldSuppressWithResolver(s, p.correctionCfg, time.Now().UTC(), plan.familySnapshot, plan.familySnapshotOK) {
		p.telnet.DeliverSelfSpot(s)
		plan.telnetDeliverNow = false
	}
	return plan, true
}

func (p *outputPipeline) updateGridCache(s *spot.Spot) {
	if p.gridUpdate == nil {
		return
	}
	if dxGrid := strings.TrimSpace(s.DXMetadata.Grid); dxGrid != "" && !s.DXMetadata.GridDerived {
		dxCall := s.DXCallNorm
		if dxCall == "" {
			dxCall = s.DXCall
		}
		p.gridUpdate(dxCall, dxGrid)
	}
	if deGrid := strings.TrimSpace(s.DEMetadata.Grid); deGrid != "" && !s.DEMetadata.GridDerived {
		deCall := s.DECallNorm
		if deCall == "" {
			deCall = s.DECall
		}
		p.gridUpdate(deCall, deGrid)
	}
}

func (p *outputPipeline) emitSpot(s *spot.Spot, plan outputDeliveryPlan) {
	emittedNow := false
	if p.archiveWriter != nil && plan.archivePeerAllowMed && shouldArchiveSpot(s) {
		p.archiveWriter.Enqueue(s)
		emittedNow = true
	}
	if plan.telnetDeliverNow {
		p.telnet.BroadcastSpot(s, plan.allowFast, plan.allowMed, plan.allowSlow)
		emittedNow = true
	}
	if p.peerManager != nil && plan.archivePeerAllowMed {
		peerSpot := cloneSpotForPeerPublish(s)
		if p.peerManager.PublishDX(peerSpot) {
			emittedNow = true
		}
	}
	if emittedNow && p.lastOutput != nil {
		p.lastOutput.Store(time.Now().UTC().UnixNano())
	}
}
