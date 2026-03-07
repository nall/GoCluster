package spot

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// IARURegion identifies the DX station's IARU region for region-aware
// frequency policy. The value is derived from CTY/DXCC metadata and stored on
// spot metadata so downstream consumers do not repeat the lookup.
type IARURegion string

const (
	IARURegionUnknown IARURegion = ""
	IARURegion1       IARURegion = "R1"
	IARURegion2       IARURegion = "R2"
	IARURegion3       IARURegion = "R3"
)

type iaruRegionTable struct {
	DefaultsByContinent map[string]string `yaml:"defaults_by_continent"`
	ADIFOverrides       map[int]string    `yaml:"adif_overrides"`
}

var (
	iaruRegionOnce sync.Once
	iaruRegions    iaruRegionTable
)

const iaruRegionPath = "data/config/iaru_regions.yaml"

var builtInRegionDefaults = map[string]IARURegion{
	"AF": IARURegion1,
	"AS": IARURegion3,
	"EU": IARURegion1,
	"NA": IARURegion2,
	"OC": IARURegion3,
	"SA": IARURegion2,
}

var builtInRegionOverrides = map[int]IARURegion{
	10:  IARURegion3, // Amsterdam & St. Paul Is.
	14:  IARURegion1, // Armenia
	15:  IARURegion1, // Asiatic Russia
	18:  IARURegion1, // Azerbaijan
	33:  IARURegion3, // Chagos Islands
	75:  IARURegion1, // Georgia
	111: IARURegion3, // Heard Island
	130: IARURegion1, // Kazakhstan
	135: IARURegion1, // Kyrgyzstan
	131: IARURegion3, // Kerguelen Islands
	207: IARURegion3, // Rodriguez Island
	215: IARURegion1, // Cyprus
	262: IARURegion1, // Tajikistan
	280: IARURegion1, // Turkmenistan
	283: IARURegion1, // UK Base Areas on Cyprus
	292: IARURegion1, // Uzbekistan
	304: IARURegion1, // Bahrain
	330: IARURegion1, // Iran
	333: IARURegion1, // Iraq
	336: IARURegion1, // Israel
	342: IARURegion1, // Jordan
	348: IARURegion1, // Kuwait
	354: IARURegion1, // Lebanon
	363: IARURegion1, // Mongolia
	370: IARURegion1, // Oman
	376: IARURegion1, // Qatar
	378: IARURegion1, // Saudi Arabia
	384: IARURegion1, // Syria
	390: IARURegion1, // Asiatic Turkey
	391: IARURegion1, // United Arab Emirates
	492: IARURegion1, // Yemen
	510: IARURegion1, // Palestine
}

// NormalizeIARURegion normalizes a config/runtime region token to R1/R2/R3.
func NormalizeIARURegion(region string) IARURegion {
	switch strings.TrimSpace(strings.ToUpper(region)) {
	case "R1", "1", "REGION1", "REGION 1":
		return IARURegion1
	case "R2", "2", "REGION2", "REGION 2":
		return IARURegion2
	case "R3", "3", "REGION3", "REGION 3":
		return IARURegion3
	default:
		return IARURegionUnknown
	}
}

func loadIARURegions() {
	iaruRegionOnce.Do(func() {
		iaruRegions.DefaultsByContinent = make(map[string]string, len(builtInRegionDefaults))
		iaruRegions.ADIFOverrides = make(map[int]string, len(builtInRegionOverrides))
		for continent, region := range builtInRegionDefaults {
			iaruRegions.DefaultsByContinent[continent] = string(region)
		}
		for adif, region := range builtInRegionOverrides {
			iaruRegions.ADIFOverrides[adif] = string(region)
		}

		paths := []string{iaruRegionPath, filepath.Join("..", iaruRegionPath)}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var table iaruRegionTable
			if err := yaml.Unmarshal(data, &table); err != nil {
				log.Printf("Warning: unable to parse IARU region map (%s): %v", path, err)
				return
			}
			if len(table.DefaultsByContinent) > 0 {
				iaruRegions.DefaultsByContinent = table.DefaultsByContinent
			}
			if len(table.ADIFOverrides) > 0 {
				iaruRegions.ADIFOverrides = table.ADIFOverrides
			}
			return
		}
	})
}

// ResolveIARURegion maps CTY/DXCC metadata to an IARU region. Unknown is
// returned when the ADIF/continent cannot be resolved deterministically.
func ResolveIARURegion(adif int, continent string) IARURegion {
	loadIARURegions()
	if adif > 0 {
		if region := NormalizeIARURegion(iaruRegions.ADIFOverrides[adif]); region != IARURegionUnknown {
			return region
		}
	}
	continent = strings.TrimSpace(strings.ToUpper(continent))
	if continent == "" {
		return IARURegionUnknown
	}
	return NormalizeIARURegion(iaruRegions.DefaultsByContinent[continent])
}
