package spot

// NormalizeVoiceMode normalizes voice modes (SSB) to USB/LSB by frequency.
// Key aspects: USB above 10 MHz, LSB below.
// Upstream: comment parsing, regional voice-default policy, and spot construction.
// Downstream: strings.ToUpper.
// NormalizeVoiceMode maps generic SSB to LSB/USB depending on frequency.
func NormalizeVoiceMode(mode string, freqKHz float64) string {
	upper := normalizeUpperASCIITrim(mode)
	if upper == "SSB" {
		if freqKHz >= 10000 {
			return "USB"
		}
		return "LSB"
	}
	return upper
}
