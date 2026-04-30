import { component$ } from "@builder.io/qwik";
import type { CardStatus, MasteryLevel } from "~/lib/types";
import { getMasteryLevel } from "~/lib/fsrs";

interface Props {
  status: CardStatus;
  stability: number;
}

const badgeClass: Record<MasteryLevel, string> = {
  new: "badge-neutral",
  learning: "badge-warning",
  learned: "badge-info",
  mastered: "badge-success",
  expert: "badge-primary",
};

const labelKey: Record<MasteryLevel, string> = {
  new: "New",
  learning: "Learning",
  learned: "Learned",
  mastered: "Mastered",
  expert: "Expert",
};

export const MasteryBadge = component$<Props>(({ status, stability }) => {
  const level = getMasteryLevel(status, stability);
  return (
    <span class={`badge badge-sm ${badgeClass[level]}`}>{labelKey[level]}</span>
  );
});
