/** API / DB store probe scheduling as seconds; admin telemetry link forms edit minutes. */

export function intervalMinFromSec(sec: number): number {
  return sec / 60
}

export function intervalSecFromMin(min: number): number {
  if (!Number.isFinite(min) || min <= 0) return 1
  return Math.max(1, Math.round(min * 60))
}

/** Compact label for read-only views (e.g. resource map). */
export function formatTelemetryIntervalMinLabel(sampleIntervalSec: number): string {
  const m = sampleIntervalSec / 60
  if (!Number.isFinite(m)) return '—'
  const decimals = m >= 1 ? 2 : 4
  const raw = m.toFixed(decimals).replace(/\.?0+$/, '')
  return `${raw} min`
}
