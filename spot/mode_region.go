package spot

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type regionModeClass string

const (
	regionModeClassUnknown      regionModeClass = ""
	regionModeClassCWSafe       regionModeClass = "cw_safe"
	regionModeClassVoiceDefault regionModeClass = "voice_default"
	regionModeClassMixed        regionModeClass = "mixed"
)

type regionModeSegment struct {
	FromKHz float64 `yaml:"from_khz"`
	ToKHz   float64 `yaml:"to_khz"`
	Class   string  `yaml:"class"`
}

type regionModeTable struct {
	VoiceModeByBand     map[string]string                         `yaml:"voice_mode_by_band"`
	UnknownRegionCWSafe map[string][]regionModeSegment            `yaml:"unknown_region_cw_safe"`
	Regions             map[string]map[string][]regionModeSegment `yaml:"regions"`
}

type RegionalModeResult struct {
	Mode       string
	Provenance ModeProvenance
}

var (
	regionModeOnce sync.Once
	regionModes    regionModeTable
)

const regionModePath = "data/config/iaru_mode_inference.yaml"

var builtInVoiceModeByBand = map[string]string{
	"160m": "LSB",
	"80m":  "LSB",
	"40m":  "LSB",
	"20m":  "USB",
	"17m":  "USB",
	"15m":  "USB",
	"12m":  "USB",
	"10m":  "USB",
}

var builtInUnknownRegionCWSafe = map[string][]regionModeSegment{
	"160m": {{FromKHz: 1810.0, ToKHz: 1830.0, Class: string(regionModeClassCWSafe)}},
	"80m":  {{FromKHz: 3500.0, ToKHz: 3535.0, Class: string(regionModeClassCWSafe)}},
	"40m":  {{FromKHz: 7000.0, ToKHz: 7030.0, Class: string(regionModeClassCWSafe)}},
	"30m":  {{FromKHz: 10110.0, ToKHz: 10130.0, Class: string(regionModeClassCWSafe)}},
	"20m":  {{FromKHz: 14000.0, ToKHz: 14070.0, Class: string(regionModeClassCWSafe)}},
	"17m":  {{FromKHz: 18068.0, ToKHz: 18095.0, Class: string(regionModeClassCWSafe)}},
	"15m":  {{FromKHz: 21000.0, ToKHz: 21070.0, Class: string(regionModeClassCWSafe)}},
	"12m":  {{FromKHz: 24890.0, ToKHz: 24915.0, Class: string(regionModeClassCWSafe)}},
	"10m":  {{FromKHz: 28000.0, ToKHz: 28070.0, Class: string(regionModeClassCWSafe)}},
}

var builtInRegionSegments = map[string]map[string][]regionModeSegment{
	"R1": {
		"160m": {{1800.0, 1810.0, "mixed"}, {1810.0, 1838.0, "cw_safe"}, {1838.0, 1843.0, "mixed"}, {1843.0, 2000.0, "voice_default"}},
		"80m":  {{3500.0, 3570.0, "cw_safe"}, {3570.0, 3620.0, "mixed"}, {3620.0, 3800.0, "voice_default"}},
		"60m":  {{5351.5, 5366.5, "mixed"}},
		"40m":  {{7000.0, 7040.0, "cw_safe"}, {7040.0, 7060.0, "mixed"}, {7060.0, 7200.0, "voice_default"}},
		"30m":  {{10100.0, 10130.0, "cw_safe"}, {10130.0, 10150.0, "mixed"}},
		"20m":  {{14000.0, 14070.0, "cw_safe"}, {14070.0, 14112.0, "mixed"}, {14112.0, 14350.0, "voice_default"}},
		"17m":  {{18068.0, 18095.0, "cw_safe"}, {18095.0, 18120.0, "mixed"}, {18120.0, 18168.0, "voice_default"}},
		"15m":  {{21000.0, 21070.0, "cw_safe"}, {21070.0, 21151.0, "mixed"}, {21151.0, 21450.0, "voice_default"}},
		"12m":  {{24890.0, 24915.0, "cw_safe"}, {24915.0, 24940.0, "mixed"}, {24940.0, 24990.0, "voice_default"}},
		"10m":  {{28000.0, 28070.0, "cw_safe"}, {28070.0, 28320.0, "mixed"}, {28320.0, 29000.0, "voice_default"}, {29000.0, 29700.0, "mixed"}},
		"6m":   {{50000.0, 54000.0, "mixed"}},
	},
	"R2": {
		"160m": {{1800.0, 1810.0, "mixed"}, {1810.0, 1840.0, "cw_safe"}, {1840.0, 1850.0, "mixed"}, {1850.0, 2000.0, "voice_default"}},
		"80m":  {{3500.0, 3570.0, "cw_safe"}, {3570.0, 3600.0, "mixed"}, {3600.0, 4000.0, "voice_default"}},
		"60m":  {{5351.5, 5366.5, "mixed"}},
		"40m":  {{7000.0, 7040.0, "cw_safe"}, {7040.0, 7060.0, "mixed"}, {7060.0, 7300.0, "voice_default"}},
		"30m":  {{10100.0, 10130.0, "cw_safe"}, {10130.0, 10150.0, "mixed"}},
		"20m":  {{14000.0, 14070.0, "cw_safe"}, {14070.0, 14112.0, "mixed"}, {14112.0, 14350.0, "voice_default"}},
		"17m":  {{18068.0, 18095.0, "cw_safe"}, {18095.0, 18120.0, "mixed"}, {18120.0, 18168.0, "voice_default"}},
		"15m":  {{21000.0, 21070.0, "cw_safe"}, {21070.0, 21151.0, "mixed"}, {21151.0, 21450.0, "voice_default"}},
		"12m":  {{24890.0, 24915.0, "cw_safe"}, {24915.0, 24940.0, "mixed"}, {24940.0, 24990.0, "voice_default"}},
		"10m":  {{28000.0, 28070.0, "cw_safe"}, {28070.0, 28320.0, "mixed"}, {28320.0, 29000.0, "voice_default"}, {29000.0, 29700.0, "mixed"}},
		"6m":   {{50000.0, 54000.0, "mixed"}},
	},
	"R3": {
		"160m": {{1800.0, 1830.0, "cw_safe"}, {1830.0, 1840.0, "mixed"}, {1840.0, 2000.0, "voice_default"}},
		"80m":  {{3500.0, 3535.0, "cw_safe"}, {3535.0, 3600.0, "mixed"}, {3600.0, 3900.0, "voice_default"}},
		"60m":  {{5351.5, 5366.5, "mixed"}},
		"40m":  {{7000.0, 7030.0, "cw_safe"}, {7030.0, 7060.0, "mixed"}, {7060.0, 7200.0, "voice_default"}},
		"30m":  {{10100.0, 10110.0, "mixed"}, {10110.0, 10130.0, "cw_safe"}, {10130.0, 10150.0, "mixed"}},
		"20m":  {{14000.0, 14070.0, "cw_safe"}, {14070.0, 14112.0, "mixed"}, {14112.0, 14350.0, "voice_default"}},
		"17m":  {{18068.0, 18095.0, "cw_safe"}, {18095.0, 18120.0, "mixed"}, {18120.0, 18168.0, "voice_default"}},
		"15m":  {{21000.0, 21070.0, "cw_safe"}, {21070.0, 21150.0, "mixed"}, {21150.0, 21450.0, "voice_default"}},
		"12m":  {{24890.0, 24915.0, "cw_safe"}, {24915.0, 24940.0, "mixed"}, {24940.0, 24990.0, "voice_default"}},
		"10m":  {{28000.0, 28070.0, "cw_safe"}, {28070.0, 28320.0, "mixed"}, {28320.0, 29100.0, "voice_default"}, {29100.0, 29700.0, "mixed"}},
		"6m":   {{50000.0, 54000.0, "mixed"}},
	},
}

func loadRegionModes() {
	regionModeOnce.Do(func() {
		regionModes.VoiceModeByBand = cloneStringMap(builtInVoiceModeByBand)
		regionModes.UnknownRegionCWSafe = cloneSegmentMap(builtInUnknownRegionCWSafe)
		regionModes.Regions = cloneRegionSegmentMap(builtInRegionSegments)

		paths := []string{regionModePath, filepath.Join("..", regionModePath)}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var table regionModeTable
			if err := yaml.Unmarshal(data, &table); err != nil {
				log.Printf("Warning: unable to parse regional mode table (%s): %v", path, err)
				return
			}
			if len(table.VoiceModeByBand) > 0 {
				regionModes.VoiceModeByBand = table.VoiceModeByBand
			}
			if len(table.UnknownRegionCWSafe) > 0 {
				regionModes.UnknownRegionCWSafe = table.UnknownRegionCWSafe
			}
			if len(table.Regions) > 0 {
				regionModes.Regions = table.Regions
			}
			return
		}
	})
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneSegmentMap(src map[string][]regionModeSegment) map[string][]regionModeSegment {
	dst := make(map[string][]regionModeSegment, len(src))
	for key, segments := range src {
		dst[key] = append([]regionModeSegment(nil), segments...)
	}
	return dst
}

func cloneRegionSegmentMap(src map[string]map[string][]regionModeSegment) map[string]map[string][]regionModeSegment {
	dst := make(map[string]map[string][]regionModeSegment, len(src))
	for region, bands := range src {
		dstBands := make(map[string][]regionModeSegment, len(bands))
		for band, segments := range bands {
			dstBands[band] = append([]regionModeSegment(nil), segments...)
		}
		dst[region] = dstBands
	}
	return dst
}

func normalizeRegionModeClass(class string) regionModeClass {
	switch strings.TrimSpace(strings.ToLower(class)) {
	case string(regionModeClassCWSafe):
		return regionModeClassCWSafe
	case string(regionModeClassVoiceDefault):
		return regionModeClassVoiceDefault
	case string(regionModeClassMixed):
		return regionModeClassMixed
	default:
		return regionModeClassUnknown
	}
}

func classifySegment(segments []regionModeSegment, freqKHz float64) regionModeClass {
	for _, segment := range segments {
		if freqKHz < segment.FromKHz || freqKHz > segment.ToKHz {
			continue
		}
		return normalizeRegionModeClass(segment.Class)
	}
	return regionModeClassUnknown
}

func regionModeBandForFrequency(freqKHz float64) string {
	band := strings.TrimSpace(NormalizeBand(FreqToBand(freqKHz)))
	if band == "" || band == "???" {
		return ""
	}
	return band
}

func regionVoiceModeForBand(band string, freqKHz float64) string {
	loadRegionModes()
	mode := strings.TrimSpace(regionModes.VoiceModeByBand[band])
	if mode == "" {
		return ""
	}
	return NormalizeVoiceMode(mode, freqKHz)
}

// ClassifyRegionalMode applies the final region-aware frequency policy after
// explicit mode, recent-evidence reuse, and digital inference are exhausted.
// It returns a radio mode only for cw_safe and voice_default segments.
func ClassifyRegionalMode(s *Spot) RegionalModeResult {
	if s == nil {
		return RegionalModeResult{}
	}
	loadRegionModes()

	band := regionModeBandForFrequency(s.Frequency)
	if band == "" {
		return RegionalModeResult{Provenance: ModeProvenanceRegionalUnknownBlank}
	}

	region := s.DXMetadata.IARURegion
	if region != IARURegionUnknown {
		segments := regionModes.Regions[string(region)][band]
		switch classifySegment(segments, s.Frequency) {
		case regionModeClassCWSafe:
			return RegionalModeResult{Mode: "CW", Provenance: ModeProvenanceRegionalCW}
		case regionModeClassVoiceDefault:
			if mode := regionVoiceModeForBand(band, s.Frequency); mode != "" {
				return RegionalModeResult{Mode: mode, Provenance: ModeProvenanceRegionalVoiceDefault}
			}
			return RegionalModeResult{Provenance: ModeProvenanceRegionalMixedBlank}
		case regionModeClassMixed:
			return RegionalModeResult{Provenance: ModeProvenanceRegionalMixedBlank}
		default:
			return RegionalModeResult{Provenance: ModeProvenanceRegionalMixedBlank}
		}
	}

	switch classifySegment(regionModes.UnknownRegionCWSafe[band], s.Frequency) {
	case regionModeClassCWSafe:
		return RegionalModeResult{Mode: "CW", Provenance: ModeProvenanceRegionalCW}
	default:
		return RegionalModeResult{Provenance: ModeProvenanceRegionalUnknownBlank}
	}
}
