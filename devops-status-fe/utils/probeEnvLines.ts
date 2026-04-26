/** Parse KEY=value lines for telemetry Job env (optional leading # comments, blank lines). */
export function parseProbeEnvLines(text: string): { ok: true; env: Record<string, string> } | { ok: false; error: string } {
  const env: Record<string, string> = {}
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i]
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const eq = line.indexOf('=')
    if (eq <= 0) {
      return { ok: false, error: `Invalid env line ${i + 1}: expected KEY=value` }
    }
    const key = line.slice(0, eq).trim()
    const val = line.slice(eq + 1)
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) {
      return { ok: false, error: `Invalid env key on line ${i + 1}: ${key}` }
    }
    env[key] = val
  }
  return { ok: true, env }
}

export function formatProbeEnvLines(env: Record<string, string> | undefined): string {
  if (!env || Object.keys(env).length === 0) return ''
  const keys = Object.keys(env).sort()
  return keys.map((k) => `${k}=${env[k] ?? ''}`).join('\n')
}
