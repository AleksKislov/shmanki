package fsrs

import "time"

type Config struct {
	DesiredRetention              float64
	StepUnlockStabilityDays       float64
	ReviewStabilityThresholdDays  float64
	HierarchicalDifficultyPenalty float64
	LearningSteps                 []time.Duration
	RelearningSteps               []time.Duration
}

var DefaultConfig = Config{
	DesiredRetention:              0.90,
	StepUnlockStabilityDays:       14.0,
	ReviewStabilityThresholdDays:  21.0,
	HierarchicalDifficultyPenalty: 2.0,
	LearningSteps:                 []time.Duration{1 * time.Minute, 10 * time.Minute, 1 * time.Hour},
	RelearningSteps:               []time.Duration{10 * time.Minute, 1 * time.Hour},
}

func (c Config) withDefaults() Config {
	resolved := c
	if resolved.DesiredRetention <= 0 || resolved.DesiredRetention >= 1 {
		resolved.DesiredRetention = DefaultConfig.DesiredRetention
	}
	if resolved.StepUnlockStabilityDays <= 0 {
		resolved.StepUnlockStabilityDays = DefaultConfig.StepUnlockStabilityDays
	}
	if resolved.ReviewStabilityThresholdDays <= 0 {
		resolved.ReviewStabilityThresholdDays = DefaultConfig.ReviewStabilityThresholdDays
	}
	if resolved.HierarchicalDifficultyPenalty < 0 {
		resolved.HierarchicalDifficultyPenalty = DefaultConfig.HierarchicalDifficultyPenalty
	}
	if len(resolved.LearningSteps) == 0 {
		resolved.LearningSteps = append([]time.Duration(nil), DefaultConfig.LearningSteps...)
	}
	if len(resolved.RelearningSteps) == 0 {
		resolved.RelearningSteps = append([]time.Duration(nil), DefaultConfig.RelearningSteps...)
	}

	return resolved
}
