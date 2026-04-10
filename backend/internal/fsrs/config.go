package fsrs

type Config struct {
	DesiredRetention              float64
	StepUnlockStabilityDays       float64
	ReviewStabilityThresholdDays  float64
	SupportReferenceStabilityDays float64
	HierarchicalDifficultyPenalty float64
}

var DefaultConfig = Config{
	DesiredRetention:              0.90,
	StepUnlockStabilityDays:       14.0,
	ReviewStabilityThresholdDays:  21.0,
	SupportReferenceStabilityDays: 21.0,
	HierarchicalDifficultyPenalty: 2.0,
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
	if resolved.SupportReferenceStabilityDays <= 0 {
		resolved.SupportReferenceStabilityDays = DefaultConfig.SupportReferenceStabilityDays
	}
	if resolved.HierarchicalDifficultyPenalty < 0 {
		resolved.HierarchicalDifficultyPenalty = DefaultConfig.HierarchicalDifficultyPenalty
	}

	return resolved
}
