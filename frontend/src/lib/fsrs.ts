import type { CardStatus, MasteryLevel } from "./types";

/**
 * Maps FSRS card status + stability to a human-readable mastery level.
 * Expert:   review status, stability >= 30 days
 * Mastered: review status, stability >= 14 days
 * Learned:  review status, stability < 14 days
 * Learning: learning or relearning
 * New:      new or locked
 */
export function getMasteryLevel(status: CardStatus, stability: number): MasteryLevel {
  if (status === "review") {
    if (stability >= 30) return "expert";
    if (stability >= 14) return "mastered";
    return "learned";
  }
  if (status === "learning" || status === "relearning") return "learning";
  return "new";
}

/**
 * Returns retrievability as a 0–100 percentage.
 */
export function retrievabilityPercent(retrievability: number): number {
  return Math.round(retrievability * 100);
}

/**
 * Formats stability in a human-friendly way.
 */
export function formatStability(stability: number): string {
  if (stability < 1) return `${Math.round(stability * 24)}h`;
  return `${Math.round(stability)}d`;
}
