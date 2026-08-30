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
)

type Config struct {
    DesiredRetention             float64
    StepUnlockStabilityDays      float64
    ReviewStabilityThresholdDays float64
    HierarchicalSupportPenalty   float64
    MaximumIntervalDays          float64
    LearningSteps                []time.Duration
    RelearningSteps              []time.Duration
    ParamsVersion                string
}

var DefaultConfig = Config{
    DesiredRetention:             0.90,
    StepUnlockStabilityDays:      14.0,
    ReviewStabilityThresholdDays: 21.0,
    HierarchicalSupportPenalty:   0.30,
    MaximumIntervalDays:          180.0,
    LearningSteps:                []time.Duration{1 * time.Minute, 10 * time.Minute, 1 * time.Hour},
    RelearningSteps:              []time.Duration{10 * time.Minute, 1 * time.Hour},
    ParamsVersion:                DefaultParamsVersion, // "fsrs-4.5-default-v2"
}
```

`ParamsVersion` identifies the model and parameter set this scheduler is running. It is
written to every `review_logs` row so an outcome can later be attributed to the parameters
that scheduled it. Bump it whenever weights, config defaults, or the update formulas change;
without it, optimizer runs and A/B comparisons cannot separate one parameter set from
another. See `specs/fsrs_optimization.md`.

`MaximumIntervalDays` caps how far into the future a review-mode card can be scheduled,
regardless of how high its stability climbs. Without a cap, a long streak of `Easy` ratings
on a low-difficulty card can drift the interval out to years — technically consistent with
the model, but not a useful scheduling decision for a learning app. 180 days is a reasonable
default ceiling; raise or lower it per deployment.

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

These thresholds are hand-picked, and a bad rating mapping corrupts every downstream weight
fit — the pretrained weights were tuned against self-graded Anki reviews, not auto-derived
ones. They are therefore treated as tunable hyperparameters rather than fixed constants; see
`specs/fsrs_optimization.md`.

---

## Card Status

```go
type CardStatus string

const (
    StatusLocked     CardStatus = "locked"     // step prerequisites not met
    StatusNew        CardStatus = "new"        // never reviewed
    StatusLearning   CardStatus = "learning"   // in learning steps (minutes-based)
    StatusReview     CardStatus = "review"     // graduated, scheduled by day-based FSRS
    StatusRelearning CardStatus = "relearning" // lapsed, being relearned
)
```

---

## Learning And Relearning Steps

This scheduler is a hybrid:

1. **Step mode (minutes-based)** for `new`, `learning`, and `relearning` cards.
2. **FSRS mode (days-based)** for `review` cards.

### Default Step Lists

- `LearningSteps`: `[1 minute, 10 minutes, 1 hour]`
- `RelearningSteps`: `[10 minutes, 1 hour]`

### Step Transition Rules

For cards currently in `learning` or `relearning`:

- **Again**: reset to step 0
- **Hard**: stay on current step
- **Good**: move to next step (or graduate if at the final step)
- **Easy**: graduate immediately

### Graduation Rules

- **From learning (new cards):** initialize `Stability` and `Difficulty` from the graduating rating (`InitialStability`, `InitialDifficulty`), then schedule first day-based interval via `NextInterval`.
- **From relearning (lapsed review cards):** keep the decayed stability created by `StabilityAfterForgetting` at lapse time, then schedule next day-based interval from that decayed stability.

Cards in step mode are still persisted in `card_states` with due times in minutes/hours and `interval_days` stored as fractional days.

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
D' = W[7] × D₀(4) + (1 - W[7]) × (D - W[6] × (rating - 3))
D' = clamp(D', 1.0, 10.0)
```

`W[6]` scales the per-rating delta; `W[7]` is the mean-reversion coefficient, pulling D back
toward the D₀(Easy) baseline so extreme values from random answers do not persist. `W[5]`
belongs to `InitialDifficulty` only and must not appear here.

```go
func UpdateDifficulty(difficulty float64, rating Rating, weights [19]float64) float64 {
    target := InitialDifficulty(RatingEasy, weights)
    delta := difficulty - weights[6]*(float64(rating)-3)
    updated := weights[7]*target + (1-weights[7])*delta
    return clamp(updated, MinDifficulty, MaxDifficulty)
}
```

> **Fixed 2026-08-30:** this formula was off by one weight index — it used `w[5]` for the
> delta and `w[6]` for mean reversion, leaving `w[7]` (0.0589) dead code. Because the default
> `w[6]` is **1.0651**, the mean-reversion coefficient exceeded 1, which made the weight on
> accumulated difficulty negative (−0.0651). Two consequences: difficulty was pinned in a
> ~7.1–7.6 band regardless of review history, and it moved in the *wrong direction* —
> `Again` lowered difficulty, `Easy` raised it. Measured before the fix, from D=5.0:
> `Again → 7.28`, `Easy → 7.39`; five consecutive lapses landed on 7.15 instead of 10.0.
> Difficulty was effectively noise, and since the old `NextInterval` multiplied by
> `(11 - D)/10`, every interval was scaled by a near-constant ≈0.37.
>
> Regression tests now assert monotonicity across ratings, that repeated lapses accumulate
> toward max difficulty, and that the formula matches the canonical indices exactly.

### Stability After Successful Review (SInc)

```
S' = S × (1 + e^W[8] × (11 - D) × S^(-W[9]) × (e^(W[10]×(1-R)) - 1) × hardPenalty × easyBonus)
```

`W[8]`, `W[9]`, and `W[10]` are the weights the FSRS optimizer fits specifically for this
term — matched against the reference FSRS-4.5 formula. `W[17]`/`W[18]` belong to a separate
same-day/short-term stability adjustment that this scheduler does not need (same-day
granularity is already handled by the minutes-based learning/relearning steps above), so
they are intentionally unused.

> **Fixed 2026-08-24:** an earlier version of this formula read from `w[17]`/`w[18]` instead
> of `w[8]`/`w[9]`/`w[10]`, and wrapped the whole expression in `exp(x + 1)` instead of
> `1 + x`. That meant the pretrained FSRS weights were being fed into the wrong slots —
> `w[7]` through `w[10]` were dead code — so stability growth for every `review`-mode card
> did not match what those weights were tuned to produce. Fixed to use the correct indices
> and the correct additive form. (`w[7]` remained unused until the difficulty-update fix on
> 2026-08-30; see **Difficulty Update** above.)

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
    growth := math.Exp(w[8]) * (11 - d) * math.Pow(s, -w[9]) * (math.Exp((1-r)*w[10]) - 1) * hardPenalty * easyBonus
    return math.Max(s*(1+growth), MinStability)
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

## Hierarchical Support And The Interval Modifier

For cards with prerequisites, the scheduler computes a support coefficient from the previous step.
In the current step-based info-graph model, `Pred(c)` is approximated as all cards from step `N-1`.

### Prerequisite Mastery

```go
func MeasureMastery(stability float64, referenceDays float64) float64 {
    if stability <= 0 {
        return 0
    }
    return math.Min(stability/referenceDays, 1)
}
```

This matches:

```
M_p = min(S_p / S_ref, 1)
S_ref = ReviewStabilityThresholdDays = 21 days
```

### Hierarchical Support Coefficient

Equal weights are used for all predecessor cards from the previous step:

```
H_c = average(M_p for p in Pred(c))
```

```go
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
```

If a card has no prerequisites, `H_c = 1`.
The current implementation reuses `DefaultConfig.ReviewStabilityThresholdDays` as the named constant for `S_ref`.

### Interval Modifier

A card whose prerequisite steps are still weak should come back before the scaffolding
underneath it decays. That adjustment is applied as a **separate multiplier on the interval**,
not as an addition to difficulty:

```
M_c = 1 - λ × (1 - H_c)
λ = HierarchicalSupportPenalty = 0.30, clamped to [0, MaxSupportPenalty = 0.9]
```

```go
func IntervalModifier(hierarchicalSupport float64, penalty float64) float64 {
    penalty = clamp(penalty, 0, MaxSupportPenalty)
    return 1 - penalty*(1-clamp(hierarchicalSupport, 0, 1))
}
```

`M_c` is 1 at full support and `1 - λ` at zero support. It is returned to the client as
`intervalModifier` for transparency/debugging, but it is not persisted in `card_states`.

**Why this is a separate multiplier and not part of difficulty.** Difficulty already feeds
stability growth in `StabilityAfterRecall`. Folding hierarchical support into difficulty and
then letting difficulty scale the interval counts it twice, and — more importantly — detaches
`DesiredRetention` from the retention users actually experience. Keeping the two layers apart
means the 19 weights stay a standard FSRS vector that an off-the-shelf optimizer can fit,
while λ becomes a single product knob that can be tuned on its own against replayed review
logs. See `specs/fsrs_optimization.md`.

> **Changed 2026-08-30:** this was previously `EffectiveDifficulty`, adding
> `HierarchicalDifficultyPenalty = 2.0` difficulty points to `D_base` and feeding the result
> into a `(11 - D_eff)/10` factor inside `NextInterval`. λ = 0.30 was chosen because it
> reproduces roughly the support effect the old form intended at a nominal difficulty of 5.
> The API field `state.effectiveDifficulty` became `state.intervalModifier`.

---

## Next Interval Calculation

`NextInterval` is the canonical FSRS interval: the number of days after which retrievability
decays to `DesiredRetention`. Difficulty is deliberately absent — it already acts through
stability growth.

```
interval = round(S / Factor × (DesiredRetention^(1/Decay) - 1))
```

```go
func NextInterval(stability float64, desiredRetention float64) float64 {
    interval := stability / Factor * (math.Pow(desiredRetention, 1/Decay) - 1)
    return math.Max(math.Round(interval), MinIntervalDays)
}
```

Minimum interval is 1 day. The scheduler then applies the product layer and the fuzz/cap:

```go
state.IntervalDays = s.finalizeInterval(
    NextInterval(stability, s.config.DesiredRetention) * state.IntervalModifier,
)
```

The defining property, asserted in tests across a grid of S and DR:
`Retrievability(NextInterval(S, DR), S) == DR`. Any extra factor inside `NextInterval` breaks
this, which is exactly how `DesiredRetention` silently stops meaning what it says.

> **Fixed 2026-08-30:** `NextInterval` previously multiplied the FSRS interval by
> `(11 - D_eff)/10`, which is not part of FSRS. Combined with the difficulty bug above (which
> pinned D near 7.2), every interval was scaled by a near-constant **≈0.37** — so users
> reviewed roughly 2.6× more often than `DesiredRetention: 0.90` asked for. Retention actually
> delivered at the due date, measured before the fix:
>
> | D_eff | interval | R at due date | configured |
> |-------|----------|---------------|------------|
> | 3.0   | 16d      | 0.918         | 0.900      |
> | 5.0   | 12d      | 0.936         | 0.900      |
> | 7.2   | 7d       | 0.961         | 0.900      |
> | 9.0   | 4d       | 0.977         | 0.900      |
>
> Because a shorter interval means a higher R at review time, which in turn means a smaller
> stability gain, the error compounded. An all-`Good` sequence from graduation:
>
> | rep | old interval | old S | new interval | new S |
> |-----|--------------|-------|--------------|-------|
> | 1   | 1d           | 3.13  | 3d           | 3.13  |
> | 4   | 5d           | 12.24 | 42d          | 41.52 |
> | 8   | 22d          | 57.67 | 180d (capped)| 569.51|
>
> **Two constants are now miscalibrated as a result** and should be revisited: stability
> climbs far faster than before, so `StepUnlockStabilityDays = 14` unlocks steps much sooner,
> and the `MasteryLevel` thresholds (7 / 21 / 90 / 365) inflate. Both were implicitly tuned
> against intervals that were ~2.6× too short.

### Fuzz And Maximum Interval

Every interval computed by `NextInterval` (on graduation and on ordinary review) is passed
through `Scheduler.finalizeInterval` before being stored:

```go
const (
    FuzzMinIntervalDays = 3.0
    FuzzFactor          = 0.05
)

func (s *Scheduler) finalizeInterval(days float64) float64 {
    if days >= FuzzMinIntervalDays {
        spread := days * FuzzFactor
        days += (s.fuzz()*2 - 1) * spread
    }
    if days > s.config.MaximumIntervalDays {
        days = s.config.MaximumIntervalDays
    }
    return math.Max(math.Round(days), MinIntervalDays)
}
```

- **Fuzz** randomizes intervals of 3+ days by up to ±5%, so cards learned in the same session
  don't all become due on the exact same future date. Intervals shorter than 3 days are left
  exact, since learning/relearning steps already provide fine-grained timing there.
- **Maximum interval** hard-caps the result at `MaximumIntervalDays` (180 by default) so a long
  streak of high ratings can't push a card's next review years into the future.

`Scheduler.fuzz` is `rand.Float64` by default and is overridden with a fixed function in tests
for deterministic assertions.

---

## Full Review Cycle

At a high level, `Schedule` now branches by status:

1. **Initial state (`new` / `locked`)**
   - `Easy`: immediate graduation to `review`
   - `Again` / `Hard`: start `learning` at step 0
   - `Good`: step 1 if available, otherwise immediate graduation

2. **Learning step mode (`learning`)**
   - Applies the step transition rules above using `LearningSteps`
   - Graduation initializes fresh FSRS state from graduating rating

3. **Relearning step mode (`relearning`)**
   - Applies the step transition rules above using `RelearningSteps`
   - Graduation reuses the decayed stability from lapse time

4. **Review mode (`review`)**
   - `Again`: apply `StabilityAfterForgetting`, increment lapses, move to `relearning` step 0
   - `Hard` / `Good` / `Easy`: standard day-based FSRS update and next interval

The concrete implementation is in `backend/internal/fsrs/scheduler.go`.

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

Before calling `Schedule`, the review service loads the previous step's `stability` values,
computes `H_c`, and passes it into the pure scheduler.

---

## Mastery Levels

Used for UI display only — not stored in DB, computed on the fly.

> **Recalibration pending:** these thresholds were chosen while intervals were ~2.6× shorter
> than the model intended (see **Next Interval Calculation**). Stability now grows much
> faster, so cards reach `learned` / `mastered` sooner than they used to. The same applies to
> `StepUnlockStabilityDays = 14`.

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
