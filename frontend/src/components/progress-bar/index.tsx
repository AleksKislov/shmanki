import { component$ } from "@builder.io/qwik";

interface Props {
  value: number; // 0–1
  label?: string;
}

export const ProgressBar = component$<Props>(({ value, label }) => {
  const pct = Math.round(Math.min(1, Math.max(0, value)) * 100);
  return (
    <div class="flex flex-col gap-1">
      {label && <span class="text-xs text-base-content/60">{label}</span>}
      <progress class="progress progress-primary w-full" value={pct} max={100} />
    </div>
  );
});
