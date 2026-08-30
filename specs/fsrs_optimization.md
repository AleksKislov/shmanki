# FSRS Weight Optimization

How this app moves from the pretrained FSRS defaults to weights fitted against its own
review history, and what has to exist before that is possible.

Companion to `specs/scheduler_algorithm.md`, which describes the scheduler as it runs today.

---

## Status

| Phase | What | State |
|-------|------|-------|
| 0 | Canonicalize the model — fix the difficulty indices, separate the product layer from FSRS | **Done** (2026-08-30) |
| 1 | Instrument `review_logs` with the fields fitting requires | **Done** (2026-08-30, migration 000016) |
| 2 | Replay + evaluation harness | **Not started** — needs review volume |
| 3 | Global weight fit across all users | Not started |
| 4 | Learn the `deriveRating` thresholds | Not started |
| 5 | Per-user weights with shrinkage toward global | Not started |
| 6 | Per-card difficulty priors from the population | Not started |

Phases 0 and 1 were done first because neither can be retrofitted: fitting weights against
history produced by incorrect formulas gives incorrect weights, and the four log columns
cannot be backfilled.

---

## Why the defaults are not the end state

`DefaultWeights` is the pretrained FSRS-4.5 vector from the open dataset. It was fitted
against **Anki decks where users self-grade** by pressing Again/Hard/Good/Easy. This app
derives the rating automatically from answer metadata (see *How rating is determined* in
`specs/scheduler_algorithm.md`), over token-assembly and block-ordering cards, in a
step-gated info graph. That is a different distribution of both items and ratings, so the
defaults are a reasonable prior and nothing more.

---

## Phase 1 — the logged fields (done)

Migration 000016 added four columns to `review_logs`. Each exists for a specific reason that
only matters later, which is why they are easy to forget and impossible to add retroactively.

| Column | Why fitting needs it |
|--------|---------------------|
| `params_version` | Attributes an outcome to the parameter set that scheduled it. Without it you cannot compare parameter sets, run an A/B, or exclude pre-fix rows from a fit. |
| `elapsed_days` | What *actually* happened, as opposed to `interval_before`, which is what was scheduled. The gap between the two is the signal the model learns from; users review early and late constantly. |
| `review_duration_ms` | Answer latency — a strong difficulty signal, and the input for any future latency-aware rating derivation. |
| `user_timezone` | Optimizers deduplicate to one review per card per **local** day. Without the reviewer's zone, reviews near midnight land in the wrong bucket. |

Rows predating the migration carry `params_version = 'legacy-pre-canonical'`. They came from
the broken difficulty and interval formulas and **must be excluded from every fit** — they
describe a different scheduler.

Client-supplied values are sanitized in `review.Service`: durations over 10 minutes are a tab
left open rather than thinking time and are stored as NULL, and the timezone string is
character-filtered before it reaches the database.

---

## Phase 2 — the replay and evaluation harness

Nothing downstream is trustworthy without this. It is what separates "the new weights feel
better" from "the new weights reduce held-out log-loss by 4%".

### Where it lives

In Go, in `internal/fsrs`, calling the **same** `UpdateDifficulty` / `StabilityAfterRecall` /
`StabilityAfterForgetting` / `NextInterval` the scheduler calls. Writing the evaluator in
Python against a reimplementation of the formulas means measuring something that is not the
production scheduler, and the two silently drift apart. The *optimizer* can be Python later;
the *evaluator* should not be.

Suggested layout:

```
backend/internal/fsrs/replay.go       // Replay(), Prediction, ReviewEvent
backend/internal/fsrs/metrics.go      // LogLoss(), CalibrationBins(), RMSE()
backend/cmd/fsrs-eval/main.go         // export + evaluate + print comparison table
```

### The core loop

```go
type ReviewEvent struct {
    ReviewedAt  time.Time
    Rating      Rating
    ElapsedDays float64
}

type Prediction struct {
    Predicted float64 // retrievability the model expected at review time
    Observed  bool    // whether the user actually recalled (rating > RatingAgain)
}

// Replay walks one card's review history in order, recording what the model would
// have predicted before each review, then applying the same state update the
// scheduler applies.
func Replay(history []ReviewEvent, weights [19]float64, cfg Config) []Prediction
```

The whole harness is this loop plus scoring. At each event, in this order:

1. Compute `R = Retrievability(event.ElapsedDays, S)` — **before** applying anything.
2. Record `Prediction{Predicted: R, Observed: event.Rating > RatingAgain}`.
3. Apply the state update exactly as `Scheduler.Schedule` does, and continue.

Getting step 1 and 2 the wrong way round — scoring against post-update state — produces
excellent-looking numbers that mean nothing.

### The export query

Per `(user_id, card_id)`, ordered by time:

```sql
SELECT user_id, card_id, reviewed_at, rating, elapsed_days, params_version
FROM review_logs
WHERE params_version <> 'legacy-pre-canonical'
ORDER BY user_id, card_id, reviewed_at;
```

Then deduplicate to the first review per card per **local** day
(`reviewed_at AT TIME ZONE COALESCE(user_timezone, 'UTC')`) — this is what `user_timezone`
was added for. Same-day repeats are learning-step reps, not independent recall observations,
and including them biases the fit.

### Metrics

- **Log-loss** — the headline number. Every comparison is against this.
- **RMSE over retrievability bins** — calibration; the metric the public FSRS benchmark reports.
- **A printed calibration table** — predicted-R bucket → actual recall rate → count. This is
  the one a human can read: it says in plain terms "when the model predicts 90%, users
  actually recall 96%."
- **Measured vs desired retention**, per user. Also worth running continuously in production
  as a health metric: when the two diverge, parameters are stale.

### Splits

**By time, never at random.** Train on reviews before date T, evaluate after. A random split
leaks future state into past predictions and will report that everything works perfectly.
Split by `params_version` as well, so rows from different schedulers are never pooled.

### What it unlocks immediately, before any ML

This is the argument for building it early — none of these need an optimizer, torch, or
Python:

1. **A baseline.** One log-loss number for the current defaults. Every later claim is measured
   against it.
2. **Tuning `HierarchicalSupportPenalty`.** λ = 0.30 was chosen by reasoning, not measurement.
   It is one dimension — sweep a range, read the loss, pick the minimum.
3. **Tuning the `deriveRating` thresholds.** Two dimensions (`wrong_attempts_count`,
   `distractor_clicks_count`), and likely the single highest-yield change available, because a
   bad rating mapping corrupts every downstream fit.
4. **A drift regression test.** Replay must reproduce the `stability_after` already stored in
   the logs. If it does not, the offline model and the production scheduler have diverged and
   every fitting result built on it is worthless. Worth wiring into CI.

### The honest limitation

This measures **prediction** quality — how well the model predicts recall — not **policy**
quality, i.e. whether the schedule it produces is a good one. You cannot evaluate intervals
you never scheduled. "Should `DesiredRetention` be 0.85?" needs simulation or a live A/B,
which is what `params_version` sets up.

---

## Phase 3 — global weight fit

Fit **one** weight vector across all users' logs pooled. Usually the largest single gain over
the defaults, it benefits every user including brand-new ones, and it becomes the prior for
per-user fitting.

**Do not write the optimizer.** Use `fsrs-optimizer` (Python, pip) or `fsrs-rs` (Rust) as a
scheduled job that reads `review_logs` and writes a parameter row. The loss is binary
cross-entropy between predicted retrievability and observed recall, minimized by gradient
descent.

A new vector ships only if it beats the incumbent on the **held-out** split by a real margin.
Keep every fit — weights, `fitted_at`, review count, holdout loss, previous id — so a bad fit
can be rolled back.

---

## Phase 4 — learn the rating derivation

`deriveRating` compresses rich answer metadata into four buckets using hand-picked
thresholds. Grid-search them against the same held-out log-loss. Cheap, needs only the
Phase 2 harness, and likely worth more than squeezing the 19 weights — the weights cannot
compensate for a rating signal that mislabels its inputs.

`review_duration_ms` opens a second axis here: a correct-first-try answer that took 40 seconds
is not the same event as one that took 3 seconds, and both currently derive `Easy`.

---

## Phase 5 — per-user weights with shrinkage

**Do not fit users independently.** Fit each user with an L2 penalty pulling toward the global
vector:

```
loss = BCE(observed, predicted) + λ · ||w_user − w_global||²
```

with λ scaled by review count. A user with 80 reviews stays essentially on the global
weights; a user with 5,000 gets a genuinely personal vector. This is the standard
empirical-Bayes answer to "how much data is enough?", and it gives a smooth ramp instead of a
cliff at an arbitrary threshold. (For reference, Anki's own guidance is ~400 reviews minimum
and 1,000+ preferred before independent fitting is trustworthy.)

Serving: a `user_fsrs_params` table; the Go scheduler reads the user's current row, falling
back to the global vector, falling back to `DefaultWeights`. `Config.ParamsVersion` becomes
the id of that row rather than a constant string.

### Rescheduling on a weight change

Full logs exist, so **replay each card's history under the new weights to recompute S and D**
rather than applying them only to future reviews — otherwise stored state and active weights
stay inconsistent for months. Cap how far any due date may jump so nobody opens the app to
400 cards.

---

## Phase 6 — per-card difficulty priors

Cards are shared across users through `premade_decks`, so population outcomes per card are
available. Estimate a per-card difficulty prior (Elo- or IRT-style) and initialize D from it
instead of from the first rating alone. This targets cold-start, where FSRS is weakest.

This only works now that difficulty responds to review history at all — before the 2026-08-30
fix it was pinned near 7.2 and inverted.

---

## Guardrails

- Never pool rows across `params_version` values in a single fit.
- Never evaluate on a random split.
- Never ship a vector that has not beaten the incumbent on held-out data.
- Keep fit history and make rollback a one-row update.
- Bump `ParamsVersion` on *any* change to weights, config defaults, or update formulas —
  including changes that look cosmetic.
