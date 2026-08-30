package fsrs

import "math"

const (
	Decay           = -0.5
	Factor          = 19.0 / 81.0
	MinDifficulty   = 1.0
	MaxDifficulty   = 10.0
	MinStability    = 0.1
	MinIntervalDays = 1.0
	// MaxSupportPenalty bounds the hierarchical interval modifier so a card can
	// never be scheduled at less than 10% of its FSRS interval.
	MaxSupportPenalty = 0.9
)

var DefaultWeights = [19]float64{
	0.4072, 1.1829, 3.1262, 15.4722, 7.2102,
	0.5316, 1.0651, 0.0589, 1.5330, 0.1544,
	1.0070, 1.9395, 0.1100, 0.2900, 2.2700,
	0.2500, 2.9898, 0.5100, 0.3400,
}

type Rating int

const (
	RatingAgain Rating = 1
	RatingHard  Rating = 2
	RatingGood  Rating = 3
	RatingEasy  Rating = 4
)

type CardStatus string

const (
	StatusLocked     CardStatus = "locked"
	StatusNew        CardStatus = "new"
	StatusLearning   CardStatus = "learning"
	StatusReview     CardStatus = "review"
	StatusRelearning CardStatus = "relearning"
)

func Retrievability(daysSinceReview float64, stability float64) float64 {
	if stability <= 0 {
		return 0
	}

	return math.Pow(1+Factor*daysSinceReview/stability, Decay)
}

func InitialDifficulty(rating Rating, weights [19]float64) float64 {
	difficulty := weights[4] - weights[5]*(float64(rating)-3)
	return clamp(difficulty, MinDifficulty, MaxDifficulty)
}

func InitialStability(rating Rating, weights [19]float64) float64 {
	return math.Max(weights[int(rating)-1], MinStability)
}

// UpdateDifficulty applies the FSRS difficulty update: a per-rating delta scaled
// by w[6], followed by mean reversion toward D0(Easy) scaled by w[7].
//
// The weight indices matter. w[5] is the D0 slope and belongs only to
// InitialDifficulty; w[6] is the update delta and w[7] is the mean-reversion
// coefficient. Using w[6] (1.0651 in the defaults) as the reversion coefficient
// makes the weight on accumulated difficulty negative, which pins difficulty near
// D0(Good) and inverts the response to ratings.
func UpdateDifficulty(difficulty float64, rating Rating, weights [19]float64) float64 {
	target := InitialDifficulty(RatingEasy, weights)
	delta := difficulty - weights[6]*(float64(rating)-3)
	updated := weights[7]*target + (1-weights[7])*delta
	return clamp(updated, MinDifficulty, MaxDifficulty)
}

func StabilityAfterRecall(stability float64, difficulty float64, retrievability float64, rating Rating, weights [19]float64) float64 {
	hardPenalty := 1.0
	easyBonus := 1.0
	if rating == RatingHard {
		hardPenalty = weights[15]
	}
	if rating == RatingEasy {
		easyBonus = weights[16]
	}

	growth := math.Exp(weights[8]) * (11 - difficulty) * math.Pow(stability, -weights[9]) * (math.Exp((1-retrievability)*weights[10]) - 1) * hardPenalty * easyBonus
	return math.Max(stability*(1+growth), MinStability)
}

func StabilityAfterForgetting(stability float64, difficulty float64, retrievability float64, weights [19]float64) float64 {
	next := weights[11] * math.Pow(difficulty, -weights[12]) * (math.Pow(stability+1, weights[13]) - 1) * math.Exp(weights[14]*(1-retrievability))
	return math.Max(next, MinStability)
}

func MeasureMastery(stability float64, referenceDays float64) float64 {
	if referenceDays <= 0 {
		referenceDays = DefaultConfig.ReviewStabilityThresholdDays
	}
	if stability <= 0 {
		return 0
	}

	return math.Min(stability/referenceDays, 1)
}

func HierarchicalSupport(stabilities []float64, referenceDays float64) float64 {
	if len(stabilities) == 0 {
		return 1
	}

	total := 0.0
	for _, stability := range stabilities {
		total += MeasureMastery(stability, referenceDays)
	}

	return clamp(total/float64(len(stabilities)), 0, 1)
}

// IntervalModifier is the product layer sitting on top of FSRS, not part of it.
// A card whose prerequisite steps are still weak gets a shortened interval so it
// comes back before the scaffolding underneath it decays.
//
// It is deliberately a separate multiplier rather than an addition to difficulty:
// difficulty already feeds stability growth, so folding support into it double
// counts and detaches DesiredRetention from the retention users actually get.
// Keeping it separate leaves the 19 weights a standard FSRS vector that an
// off-the-shelf optimizer can fit, and makes this knob measurable on its own.
//
// Returns 1 at full support and 1-penalty at zero support.
func IntervalModifier(hierarchicalSupport float64, penalty float64) float64 {
	penalty = clamp(penalty, 0, MaxSupportPenalty)
	return 1 - penalty*(1-clamp(hierarchicalSupport, 0, 1))
}

// NextInterval is the canonical FSRS interval: the number of days after which
// retrievability decays to desiredRetention. Difficulty is deliberately absent --
// it already acts through stability growth in StabilityAfterRecall. Product-level
// adjustments belong in IntervalModifier.
func NextInterval(stability float64, desiredRetention float64) float64 {
	interval := stability / Factor * (math.Pow(desiredRetention, 1/Decay) - 1)
	return math.Max(math.Round(interval), MinIntervalDays)
}

func MasteryLevel(stability float64) string {
	switch {
	case stability < 7:
		return "new"
	case stability < DefaultConfig.ReviewStabilityThresholdDays:
		return "learning"
	case stability < 90:
		return "learned"
	case stability < 365:
		return "mastered"
	default:
		return "expert"
	}
}

func clamp(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}
