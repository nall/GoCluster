package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"dxcluster/config"
	"dxcluster/spot"
)

func main() {
	var (
		dateFlag         = flag.String("date", "", "UTC date to replay (YYYY-MM-DD or YYYYMMDD)")
		replayConfigFlag = flag.String("replay-config", defaultReplayConfigPath, "Replay config YAML path")
		configDirFlag    = flag.String("config", "", "Cluster config directory override (defaults to replay config, DXC_CONFIG_PATH, or data/config)")
		archiveDirFlag   = flag.String("archive-dir", "", "Archive directory override (defaults to replay config)")
		forceDownload    = flag.Bool("force-download", false, "Force re-download of the RBN history zip (overrides replay config)")
	)
	flag.Parse()
	must(runReplay(replayRequest{
		dateValue:        *dateFlag,
		replayConfigPath: *replayConfigFlag,
		configDir:        *configDirFlag,
		archiveDir:       *archiveDirFlag,
		forceDownload:    *forceDownload,
	}))
}

func loadSpotterReliability(cfg config.CallCorrectionConfig) (base spot.SpotterReliability, cw spot.SpotterReliability, rtty spot.SpotterReliability, err error) {
	if relPath := strings.TrimSpace(cfg.SpotterReliabilityFile); relPath != "" {
		if _, statErr := os.Stat(relPath); statErr != nil {
			return nil, nil, nil, fmt.Errorf("call_correction.spotter_reliability_file missing/unreadable %s: %w", relPath, statErr)
		}
		rel, _, loadErr := spot.LoadSpotterReliability(relPath)
		if loadErr != nil {
			return nil, nil, nil, loadErr
		}
		base = rel
	}
	if relPath := strings.TrimSpace(cfg.SpotterReliabilityFileCW); relPath != "" {
		if _, statErr := os.Stat(relPath); statErr != nil {
			return nil, nil, nil, fmt.Errorf("call_correction.spotter_reliability_file_cw missing/unreadable %s: %w", relPath, statErr)
		}
		rel, _, loadErr := spot.LoadSpotterReliability(relPath)
		if loadErr != nil {
			return nil, nil, nil, loadErr
		}
		cw = rel
	}
	if relPath := strings.TrimSpace(cfg.SpotterReliabilityFileRTTY); relPath != "" {
		if _, statErr := os.Stat(relPath); statErr != nil {
			return nil, nil, nil, fmt.Errorf("call_correction.spotter_reliability_file_rtty missing/unreadable %s: %w", relPath, statErr)
		}
		rel, _, loadErr := spot.LoadSpotterReliability(relPath)
		if loadErr != nil {
			return nil, nil, nil, loadErr
		}
		rtty = rel
	}
	return base, cw, rtty, nil
}

func loadConfusionModel(cfg config.CallCorrectionConfig) (*spot.ConfusionModel, error) {
	if !cfg.ConfusionModelEnabled {
		return nil, nil
	}
	modelPath := strings.TrimSpace(cfg.ConfusionModelFile)
	if modelPath == "" {
		return nil, errors.New("call_correction.confusion_model_enabled=true but confusion_model_file is empty")
	}
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("call_correction.confusion_model_file missing/unreadable %s: %w", modelPath, err)
	}
	loaded, err := spot.LoadConfusionModel(modelPath)
	if err != nil {
		return nil, err
	}
	return loaded, nil
}
