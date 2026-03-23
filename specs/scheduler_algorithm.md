# Custom FSRS Algorithm Specification

## Overview

This project implements a **custom variant of FSRS** (Free Spaced Repetition Scheduler).
FSRS is based on the two-component model of memory (DSR model) developed by Piotr Wozniak.

The three core memory parameters per card per user:

- **D** — Difficulty (1.0–10.0)
- **S** — Stability (days until R drops to 0.9)
- **R** — Retrievability (current recall probability, 0.0–1.0)

---

## Constants

```go
const (
    Decay  = -0.5
    Factor = 19.0 / 81.0 // ≈ 0.2346

    // Stability threshold to unlock next step in an InfoObject
    StepUnlockStabilityDays = 14.0

    // Default desired retention
    DesiredRetention = 0.9
)
```

---

## Default Weights (W)

Pretrained weights from the FSRS open dataset.
These are the starting weights before any per-user optimization.

```go
var DefaultWeights = [19]float64{
    0.4072, 1.1829, 3.1262, 15.4722, 7.2102,
    0.5316, 1.0651, 0.0589, 1.5330,  0.1544,
    1.0070, 1.9395, 0.1100, 0.2900,  2.2700,
    0.2500, 2.9898, 0.5100, 0.3400,
}
```

---

## Rating Scale

```go
type Rating int

const (
    RatingAgain Rating = 1 // wrong answer
    RatingHard  Rating = 2 // correct, but with one incorrect attempt
    RatingGood  Rating = 3 // correct, normal effort, but slow / uncertain
    RatingEasy  Rating = 4 // correct, instant recall
)
```

### How rating is determined in this app

Unlike standard FSRS (where users click Again/Good buttons),
this app **determines rating automatically** from the user's answer metadata.

The review submission payload includes:

- `answered_tokens`: final token sequence submitted for scoring
- `attempts`: ordered list of attempts made during the card review
- `wrong_attempts_count`: number of failed attempts before the final submission
- `distractor_clicks_count`: total number of distractor token clicks across the review
- `incorrect_tokens_clicked`: flat list of wrong tokens clicked for analytics/debugging

```
Final submission is incorrect                                        → RatingAgain (1)
Final submission is correct, but wrong_attempts_count > 0            → RatingHard (2)
Final submission is correct, wrong_attempts_count = 0,
and distractor_clicks_count > 0                                      → RatingGood (3)
Final submission is correct, wrong_attempts_count = 0,
and distractor_clicks_count = 0                                      → RatingEasy (4)
```

`attempts` is stored primarily for replay, debugging, and analytics.
The scheduler itself only needs the derived rating.

---

## Card Status

```go
type CardStatus string

const (
    StatusLocked    CardStatus = "locked"     // step prerequisites not met
    StatusNew       CardStatus = "new"        // never reviewed
    StatusLearning  CardStatus = "learning"   // S < 21 days
    StatusReview    CardStatus = "review"     // S >= 21 days
    StatusRelearning CardStatus = "relearning" // lapsed, being relearned
)
```

---

## Core Formulas

### Retrievability R(t)

```
R(t) = (1 + Factor × t / S) ^ Decay
```

```go
func Retrievability(t float64, s float64) float64 {
    return math.Pow(1+Factor*t/s, Decay)
}
```

Where `t` = days since last review, `s` = current stability.

### Initial Difficulty D₀

Set on the very first review based on rating:

```
D₀ = W[4] - W[5] × (rating - 3)
D₀ = clamp(D₀, 1.0, 10.0)
```

```go
func InitialDifficulty(rating Rating, w [19]float64) float64 {
    d := w[4] - w[5]*(float64(rating)-3)
    return clamp(d, 1.0, 10.0)
}
```

### Initial Stability S₀

Set on the very first review based on rating:

```
S₀ = W[rating - 1]   // W[0] for Again, W[1] for Hard, W[2] for Good, W[3] for Easy
```

```go
func InitialStability(rating Rating, w [19]float64) float64 {
    return math.Max(w[rating-1], 0.1)
}
```

### Difficulty Update

After each review:

```
D' = W[6] × D₀(3) + (1 - W[6]) × (D - W[5] × (rating - 3))
D' = clamp(D', 1.0, 10.0)
```

Mean reversion pulls D back toward the baseline (D₀ for rating=3)
to prevent extreme values from random answers.

```go
func UpdateDifficulty(d float64, rating Rating, w [19]float64) float64 {
    baseline := InitialDifficulty(RatingGood, w)
    d2 := w[6]*baseline + (1-w[6])*(d-w[5]*(float64(rating)-3))
    return clamp(d2, 1.0, 10.0)
}
```

### Stability After Successful Review (SInc)

```
SInc = S × e^(W[17] × (11 - D) × S^(-W[18]) × (e^(W[18]×(1-R)) - 1) + 1)
```

Note: the project uses a 19-element weight array `w[0]..w[18]`.
In code, the final multiplier term uses `w[18]`; there is no `w[19]`.

```go
func StabilityAfterRecall(s, d, r float64, rating Rating, w [19]float64) float64 {
    hardPenalty := 1.0
    easyBonus := 1.0
    if rating == RatingHard {
        hardPenalty = w[15]
    }
    if rating == RatingEasy {
        easyBonus = w[16]
    }
    sinc := math.Exp(w[17]*(11-d)*math.Pow(s, -w[18])*(math.Exp((1-r)*w[18])-1)+1)
    return s * sinc * hardPenalty * easyBonus
}
```

### Stability After Forgetting (Lapse)

When user answers Again:

```
S' = W[11] × D^(-W[12]) × ((S+1)^W[13] - 1) × e^(W[14]×(1-R))
```

```go
func StabilityAfterForgetting(s, d, r float64, w [19]float64) float64 {
    return w[11] * math.Pow(d, -w[12]) *
        (math.Pow(s+1, w[13])-1) *
        math.Exp(w[14]*(1-r))
}
```

---

## Next Interval Calculation

The next interval is the number of days until R drops to DesiredRetention:

```
interval = S / Factor × (DesiredRetention^(1/Decay) - 1)
```

```go
func NextInterval(s float64) float64 {
    return math.Round(s / Factor * (math.Pow(DesiredRetention, 1/Decay) - 1))
}
```

Minimum interval is 1 day.

---

## Full Review Cycle

```go
func (sc *Scheduler) Schedule(state CardState, rating Rating, now time.Time) CardState {
    w := sc.weights
    t := now.Sub(state.LastReview).Hours() / 24 // days since last review

    r := Retrievability(t, state.Stability)

    var newS, newD float64

    if rating == RatingAgain {
        newS = StabilityAfterForgetting(state.Stability, state.Difficulty, r, w)
        newD = UpdateDifficulty(state.Difficulty, rating, w)
        if state.Status == StatusReview {
            state.Lapses++
        }
        state.Status = StatusRelearning
    } else {
        newS = StabilityAfterRecall(state.Stability, state.Difficulty, r, rating, w)
        newD = UpdateDifficulty(state.Difficulty, rating, w)
        state.Reps++
        if newS >= 21 {
            state.Status = StatusReview
        } else {
            state.Status = StatusLearning
        }
    }

    state.Stability     = newS
    state.Difficulty    = newD
    state.Retrievability = Retrievability(0, newS) // R right after review = ~1.0
    state.IntervalDays  = math.Max(NextInterval(newS), 1)
    state.LastReview    = now
    state.DueDate       = now.Add(time.Duration(state.IntervalDays * 24) * time.Hour)

    return state
}
```

---

## Step Unlock Logic

An InfoObject has cards grouped by `step` (0, 1, 2, ...).
Cards of step N are **locked** until all cards of step N-1 have `Stability >= 14 days`.

```go
func ShouldUnlockStep(states []CardState, step int) bool {
    for _, s := range states {
        if s.Step == step-1 && s.Stability < StepUnlockStabilityDays {
            return false
        }
    }
    return true
}
```

This is checked after every review submission.

---

## Mastery Levels

Used for UI display only — not stored in DB, computed on the fly:

```go
func MasteryLevel(s float64) string {
    switch {
    case s < 7:
        return "new"       // < 1 week
    case s < 21:
        return "learning"  // 1–3 weeks
    case s < 90:
        return "learned"   // 3 weeks – 3 months
    case s < 365:
        return "mastered"  // 3 months – 1 year
    default:
        return "expert"    // > 1 year
    }
}
```
