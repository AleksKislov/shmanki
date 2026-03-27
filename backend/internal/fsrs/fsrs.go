package fsrs

import "math"

const (
	Decay           = -0.5
	Factor          = 19.0 / 81.0
	MinStability    = 0.1
	MinIntervalDays = 1.0
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
	return clamp(difficulty, 1, 10)
}

func InitialStability(rating Rating, weights [19]float64) float64 {
	return math.Max(weights[int(rating)-1], MinStability)
}

func UpdateDifficulty(difficulty float64, rating Rating, weights [19]float64) float64 {
	baseline := InitialDifficulty(RatingGood, weights)
	updated := weights[6]*baseline + (1-weights[6])*(difficulty-weights[5]*(float64(rating)-3))
	return clamp(updated, 1, 10)
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

	stabilityIncrease := math.Exp(weights[17]*(11-difficulty)*math.Pow(stability, -weights[18])*(math.Exp((1-retrievability)*weights[18])-1) + 1)
	return math.Max(stability*stabilityIncrease*hardPenalty*easyBonus, MinStability)
}

func StabilityAfterForgetting(stability float64, difficulty float64, retrievability float64, weights [19]float64) float64 {
	next := weights[11] * math.Pow(difficulty, -weights[12]) * (math.Pow(stability+1, weights[13]) - 1) * math.Exp(weights[14]*(1-retrievability))
	return math.Max(next, MinStability)
}

func NextInterval(stability float64, desiredRetention float64) float64 {
	interval := math.Round(stability / Factor * (math.Pow(desiredRetention, 1/Decay) - 1))
	return math.Max(interval, MinIntervalDays)
}

func MasteryLevel(stability float64) string {
	switch {
	case stability < 7:
		return "new"
	case stability < 21:
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
