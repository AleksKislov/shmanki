package fsrs

import "time"

// DefaultParamsVersion identifies the model and parameter set that produced a
// schedule. It is written to every review log so an outcome can later be
// attributed to the parameters that scheduled it -- without that, optimizer runs
// and A/B comparisons cannot separate one parameter set from another.
//
// Bump it whenever weights, config defaults, or the update formulas change.
// When per-user optimized weights land this becomes the id of a stored
// parameter row rather than a constant.
const DefaultParamsVersion = "fsrs-4.5-default-v2"

type Config struct {
	DesiredRetention             float64
	StepUnlockStabilityDays      float64
	ReviewStabilityThresholdDays float64
	// HierarchicalSupportPenalty is the fraction of the FSRS interval removed
	// when a card has zero prerequisite support, in [0, MaxSupportPenalty].
	// See fsrs.IntervalModifier.
	HierarchicalSupportPenalty float64
	MaximumIntervalDays        float64
	LearningSteps              []time.Duration
	RelearningSteps            []time.Duration
	ParamsVersion              string
}

var DefaultConfig = Config{
	DesiredRetention:             0.90,
	StepUnlockStabilityDays:      14.0,
	ReviewStabilityThresholdDays: 21.0,
	// 0.30 keeps roughly the support effect the old effective-difficulty form
	// intended at a nominal difficulty of 5. It is a product knob now, tunable
	// on its own against replayed review logs.
	HierarchicalSupportPenalty: 0.30,
	MaximumIntervalDays:        180.0,
	LearningSteps:              []time.Duration{1 * time.Minute, 10 * time.Minute, 1 * time.Hour},
	RelearningSteps:            []time.Duration{10 * time.Minute, 1 * time.Hour},
	ParamsVersion:              DefaultParamsVersion,
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
	if resolved.HierarchicalSupportPenalty < 0 || resolved.HierarchicalSupportPenalty > MaxSupportPenalty {
		resolved.HierarchicalSupportPenalty = DefaultConfig.HierarchicalSupportPenalty
	}
	if resolved.MaximumIntervalDays <= 0 {
		resolved.MaximumIntervalDays = DefaultConfig.MaximumIntervalDays
	}
	if len(resolved.LearningSteps) == 0 {
		resolved.LearningSteps = append([]time.Duration(nil), DefaultConfig.LearningSteps...)
	}
	if len(resolved.RelearningSteps) == 0 {
		resolved.RelearningSteps = append([]time.Duration(nil), DefaultConfig.RelearningSteps...)
	}
	if resolved.ParamsVersion == "" {
		resolved.ParamsVersion = DefaultParamsVersion
	}

	return resolved
}
