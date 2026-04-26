/** Suggest a unique telemetry name when duplicating, given existing definition names. */
export function suggestDuplicateName(sourceName: string, existingNames: string[]): string {
  const base = sourceName.replace(/( \(copy(?: \d+)?\))+$/i, '').trim() || sourceName.trim()
  let candidate = `${base} (copy)`
  let n = 2
  const set = new Set(existingNames)
  while (set.has(candidate)) {
    candidate = `${base} (copy ${n})`
    n++
  }
  return candidate
}
