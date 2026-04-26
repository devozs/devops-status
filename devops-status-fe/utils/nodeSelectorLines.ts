/**
 * Parse KEY=value lines for Kubernetes nodeSelector (optional # comments).
 * Keys allow label-style names e.g. topology.kubernetes.io/zone
 */
export function parseNodeSelectorLines(
  text: string,
): { ok: true; node_selector: Record<string, string> } | { ok: false; error: string } {
  const node_selector: Record<string, string> = {}
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line || line.startsWith('#')) continue
    const eq = line.indexOf('=')
    if (eq <= 0) {
      return { ok: false, error: `Invalid line ${i + 1}: expected key=value` }
    }
    const key = line.slice(0, eq).trim()
    const val = line.slice(eq + 1).trim()
    if (!keyValid(key)) {
      return { ok: false, error: `Invalid node selector key on line ${i + 1}: ${key}` }
    }
    if (!valueValid(val)) {
      return { ok: false, error: `Invalid node selector value on line ${i + 1} (max 63 chars, no newlines)` }
    }
    node_selector[key] = val
  }
  return { ok: true, node_selector }
}

function keyValid(k: string): boolean {
  if (k.length < 1 || k.length > 253) return false
  for (let j = 0; j < k.length; j++) {
    const c = k.charCodeAt(j)
    if (
      (c >= 0x61 && c <= 0x7a) ||
      (c >= 0x41 && c <= 0x5a) ||
      (c >= 0x30 && c <= 0x39) ||
      c === 0x2d ||
      c === 0x5f ||
      c === 0x2e ||
      c === 0x2f
    ) {
      continue
    }
    return false
  }
  return true
}

function valueValid(v: string): boolean {
  if (v.length > 63 || /[\x00-\x08\x0b\x0c\x0e-\x1f]/.test(v)) return false
  return true
}

export function formatNodeSelectorLines(sel: Record<string, string> | undefined): string {
  if (!sel || Object.keys(sel).length === 0) return ''
  const keys = Object.keys(sel).sort()
  return keys.map((k) => `${k}=${sel[k] ?? ''}`).join('\n')
}
