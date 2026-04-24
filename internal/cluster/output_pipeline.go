package cluster

import (
	"log"
	"runtime/debug"
	"sync/atomic"
	"time"

	"dxcluster/archive"
	"dxcluster/buffer"
	"dxcluster/config"
	"dxcluster/cty"
	"dxcluster/dedup"
	"dxcluster/pathreliability"
	"dxcluster/peer"
	"dxcluster/spot"
	"dxcluster/stats"
	"dxcluster/telnet"
	"dxcluster/ui"
)

// outputPipeline owns the single consumer of deduplicated spots plus the
// temporal and FT corroboration timer state. The only extra goroutine is the
// pre-existing telnet stabilizer release loop.
type outputPipeline struct {
	outputChan              <-chan *spot.Spot
	secondaryFast           *dedup.SecondaryDeduper
	secondaryMed            *dedup.SecondaryDeduper
	secondarySlow           *dedup.SecondaryDeduper
	archivePeerSecondaryMed *dedup.SecondaryDeduper
	secondaryStage          *atomic.Uint64
	modeAssigner            *spot.ModeAssigner
	buf                     *buffer.RingBuffer
	telnet                  *telnet.Server
	peerManager             *peer.Manager
	tracker                 *stats.Tracker
	signalResolver          *spot.SignalResolver
	correctionCfg           config.CallCorrectionConfig
	ctyLookup               func() *cty.CTYDatabase
	metaCache               *callMetaCache
	harmonicDetector        *spot.HarmonicDetector
	harmonicCfg             config.HarmonicConfig
	freqAvg                 *spot.FrequencyAverager
	spotPolicy              config.SpotPolicy
	dash                    ui.Surface
	gridUpdate              func(call, grid string)
	gridLookup              func(call string) (string, bool, bool)
	gridLookupSync          func(call string) (string, bool, bool)
	unlicensedReporter      func(source, role, call, deCall, dxCall, mode string, freq float64)
	droppedCallLogger       *droppedCallLogger
	adaptiveMinReports      *spot.AdaptiveMinReports
	refresher               *adaptiveRefresher
	spotterReliability      spot.SpotterReliability
	spotterReliabilityCW    spot.SpotterReliability
	spotterReliabilityRTTY  spot.SpotterReliability
	confusionModel          *spot.ConfusionModel
	recentBandStore         spot.RecentSupportStore
	customSCPStore          *spot.CustomSCPStore
	whoSpotsMeStore         *spot.WhoSpotsMeStore
	broadcastKeepSSID       bool
	archiveWriter           *archive.Writer
	lastOutput              *atomic.Int64
	pathPredictor           *pathreliability.Predictor
	pathReport              *pathReportMetrics
	allowedBands            map[string]struct{}
	secondaryActive         bool
	stabilizerEnabled       bool
	telnetStabilizer        *telnetSpotStabilizer
	familySuppressor        *telnetFamilySuppressor
	temporal                *runtimeTemporalController
	temporalTimer           *time.Timer
	temporalTimerCh         <-chan time.Time
	ftConfidence            *ftConfidenceController
	ftRecentBandStore       *spot.RecentBandStore
	ftTimer                 *time.Timer
	ftTimerCh               <-chan time.Time
}

type outputSpotContext struct {
	spot                       *spot.Spot
	ctyDB                      *cty.CTYDatabase
	dirty                      bool
	modeUpper                  string
	stabilizerResolverKey      spot.ResolverSignalKey
	hasStabilizerResolverKey   bool
	stabilizerEvidenceEnqueued bool
}

func newOutputPipeline(
	deduplicator *dedup.Deduplicator,
	secondaryFast *dedup.SecondaryDeduper,
	secondaryMed *dedup.SecondaryDeduper,
	secondarySlow *dedup.SecondaryDeduper,
	archivePeerSecondaryMed *dedup.SecondaryDeduper,
	secondaryStage *atomic.Uint64,
	modeAssigner *spot.ModeAssigner,
	buf *buffer.RingBuffer,
	telnet *telnet.Server,
	peerManager *peer.Manager,
	tracker *stats.Tracker,
	signalResolver *spot.SignalResolver,
	correctionCfg config.CallCorrectionConfig,
	ctyLookup func() *cty.CTYDatabase,
	metaCache *callMetaCache,
	harmonicDetector *spot.HarmonicDetector,
	harmonicCfg config.HarmonicConfig,
	freqAvg *spot.FrequencyAverager,
	spotPolicy config.SpotPolicy,
	dash ui.Surface,
	gridUpdate func(call, grid string),
	gridLookup func(call string) (string, bool, bool),
	gridLookupSync func(call string) (string, bool, bool),
	unlicensedReporter func(source, role, call, deCall, dxCall, mode string, freq float64),
	droppedCallLogger *droppedCallLogger,
	adaptiveMinReports *spot.AdaptiveMinReports,
	refresher *adaptiveRefresher,
	spotterReliability spot.SpotterReliability,
	spotterReliabilityCW spot.SpotterReliability,
	spotterReliabilityRTTY spot.SpotterReliability,
	confusionModel *spot.ConfusionModel,
	recentBandStore spot.RecentSupportStore,
	customSCPStore *spot.CustomSCPStore,
	whoSpotsMeStore *spot.WhoSpotsMeStore,
	broadcastKeepSSID bool,
	archiveWriter *archive.Writer,
	lastOutput *atomic.Int64,
	pathPredictor *pathreliability.Predictor,
	pathReport *pathReportMetrics,
	allowedBands map[string]struct{},
) *outputPipeline {
	pipeline := &outputPipeline{
		outputChan:              deduplicator.GetOutputChannel(),
		secondaryFast:           secondaryFast,
		secondaryMed:            secondaryMed,
		secondarySlow:           secondarySlow,
		archivePeerSecondaryMed: archivePeerSecondaryMed,
		secondaryStage:          secondaryStage,
		modeAssigner:            modeAssigner,
		buf:                     buf,
		telnet:                  telnet,
		peerManager:             peerManager,
		tracker:                 tracker,
		signalResolver:          signalResolver,
		correctionCfg:           correctionCfg,
		ctyLookup:               ctyLookup,
		metaCache:               metaCache,
		harmonicDetector:        harmonicDetector,
		harmonicCfg:             harmonicCfg,
		freqAvg:                 freqAvg,
		spotPolicy:              spotPolicy,
		dash:                    dash,
		gridUpdate:              gridUpdate,
		gridLookup:              gridLookup,
		gridLookupSync:          gridLookupSync,
		unlicensedReporter:      unlicensedReporter,
		droppedCallLogger:       droppedCallLogger,
		adaptiveMinReports:      adaptiveMinReports,
		refresher:               refresher,
		spotterReliability:      spotterReliability,
		spotterReliabilityCW:    spotterReliabilityCW,
		spotterReliabilityRTTY:  spotterReliabilityRTTY,
		confusionModel:          confusionModel,
		recentBandStore:         recentBandStore,
		customSCPStore:          customSCPStore,
		whoSpotsMeStore:         whoSpotsMeStore,
		broadcastKeepSSID:       broadcastKeepSSID,
		archiveWriter:           archiveWriter,
		lastOutput:              lastOutput,
		pathPredictor:           pathPredictor,
		pathReport:              pathReport,
		allowedBands:            allowedBands,
		secondaryActive:         secondaryFast != nil || secondaryMed != nil || secondarySlow != nil,
		temporal:                newRuntimeTemporalController(correctionCfg),
		ftConfidence:            newFTConfidenceController(correctionCfg, tracker),
		ftRecentBandStore:       newFTRecentBandStore(correctionCfg),
	}
	pipeline.stabilizerEnabled = telnet != nil && correctionCfg.Enabled && correctionCfg.StabilizerEnabled && recentBandStore != nil
	if pipeline.stabilizerEnabled {
		pipeline.telnetStabilizer = newTelnetSpotStabilizer(
			time.Duration(correctionCfg.StabilizerDelaySeconds)*time.Second,
			correctionCfg.StabilizerMaxPending,
		)
		pipeline.telnetStabilizer.Start()
	}
	if telnet != nil && correctionCfg.Enabled && correctionCfg.FamilyPolicy.TelnetSuppression.Enabled {
		pipeline.familySuppressor = newTelnetFamilySuppressor(
			time.Duration(correctionCfg.FamilyPolicy.TelnetSuppression.WindowSeconds)*time.Second,
			correctionCfg.FamilyPolicy.TelnetSuppression.MaxEntries,
			spot.CorrectionFamilyPolicy{
				Configured:                 true,
				TruncationEnabled:          correctionCfg.FamilyPolicy.Truncation.Enabled,
				TruncationMaxLengthDelta:   correctionCfg.FamilyPolicy.Truncation.MaxLengthDelta,
				TruncationMinShorterLength: correctionCfg.FamilyPolicy.Truncation.MinShorterLength,
				TruncationAllowPrefix:      correctionCfg.FamilyPolicy.Truncation.AllowPrefixMatch,
				TruncationAllowSuffix:      correctionCfg.FamilyPolicy.Truncation.AllowSuffixMatch,
			},
			correctionCfg.FamilyPolicy.TelnetSuppression.FrequencyToleranceFallbackHz,
		)
	}
	return pipeline
}

func (p *outputPipeline) run() {
	if p.stabilizerEnabled {
		defer p.telnetStabilizer.Stop()
		p.startStabilizerReleaseLoop()
	}
	defer p.stopTemporalTimer()
	defer p.stopFTTimer()

	for {
		now := time.Now().UTC()
		p.releaseDueTemporal(now, false)
		p.scheduleTemporalTimer(now)
		p.releaseDueFT(now, false)
		p.scheduleFTTimer(now)

		select {
		case s, ok := <-p.outputChan:
			if !ok {
				p.releaseDueTemporal(time.Now().UTC().Add(24*time.Hour), true)
				p.releaseDueFT(time.Now().UTC().Add(24*time.Hour), true)
				return
			}
			p.processSpot(s, nil)
		case <-p.temporalTimerCh:
			// Timer-driven wakeup for lag/max-wait release when ingest is idle.
		case <-p.ftTimerCh:
			// Timer-driven wakeup for bounded FT corroboration release when ingest is idle.
		}
	}
}

func (p *outputPipeline) processSpot(s *spot.Spot, temporalRelease *runtimeTemporalRelease) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("processOutputSpots panic: %v\n%s", r, debug.Stack())
		}
	}()
	p.processSpotBody(s, temporalRelease)
}

func (p *outputPipeline) processSpotBody(s *spot.Spot, temporalRelease *runtimeTemporalRelease) {
	ctx, ok := p.prepareSpotContext(s)
	if !ok {
		return
	}
	if !p.applyResolverStage(&ctx, temporalRelease) {
		return
	}
	if !p.applyPostResolverAdjustments(&ctx) {
		return
	}
	if !p.applyFTConfidenceStage(&ctx, time.Now().UTC()) {
		return
	}
	if !p.finalizeSpotForMetrics(&ctx) {
		return
	}
	if !p.prepareFanoutSpot(&ctx) {
		return
	}
	p.deliverSpot(&ctx)
}

func (p *outputPipeline) stopTemporalTimer() {
	if p.temporalTimer == nil {
		return
	}
	if !p.temporalTimer.Stop() {
		select {
		case <-p.temporalTimer.C:
		default:
		}
	}
	p.temporalTimer = nil
	p.temporalTimerCh = nil
}

func (p *outputPipeline) scheduleTemporalTimer(now time.Time) {
	if p.temporal == nil || !p.temporal.Enabled() {
		p.stopTemporalTimer()
		return
	}
	nextDue, ok := p.temporal.NextDue()
	if !ok {
		p.stopTemporalTimer()
		return
	}
	delay := nextDue.Sub(now)
	if delay < 0 {
		delay = 0
	}
	if p.temporalTimer == nil {
		p.temporalTimer = time.NewTimer(delay)
	} else {
		if !p.temporalTimer.Stop() {
			select {
			case <-p.temporalTimer.C:
			default:
			}
		}
		p.temporalTimer.Reset(delay)
	}
	p.temporalTimerCh = p.temporalTimer.C
}
