package spot

// ModeProvenance records how Spot.Mode was assigned.
// It is intentionally separate from Mode so downstream consumers can
// distinguish trusted evidence from regional product-policy outcomes.
type ModeProvenance string

const (
	ModeProvenanceUnknown              ModeProvenance = ""
	ModeProvenanceSourceExplicit       ModeProvenance = "source_explicit"
	ModeProvenanceCommentExplicit      ModeProvenance = "comment_explicit"
	ModeProvenanceRecentEvidence       ModeProvenance = "recent_evidence"
	ModeProvenanceDigitalFrequency     ModeProvenance = "digital_frequency"
	ModeProvenanceRegionalCW           ModeProvenance = "regional_cw"
	ModeProvenanceRegionalVoiceDefault ModeProvenance = "regional_voice_default"
	ModeProvenanceRegionalMixedBlank   ModeProvenance = "regional_mixed_blank"
	ModeProvenanceRegionalUnknownBlank ModeProvenance = "regional_unknown_blank"
)

// IsReusableEvidence reports whether a provenance class is allowed to seed or
// refresh reusable inference state.
func (p ModeProvenance) IsReusableEvidence() bool {
	switch p {
	case ModeProvenanceSourceExplicit,
		ModeProvenanceCommentExplicit,
		ModeProvenanceRecentEvidence,
		ModeProvenanceDigitalFrequency:
		return true
	default:
		return false
	}
}
