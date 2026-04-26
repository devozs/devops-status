export type TelemetrySampleRow = {
  id: string
  sampled_at: string
  success: boolean
  latency_ms?: number | null
  source_trace?: string
  metadata_json?: Record<string, unknown>
}

export function formatSampleTime(iso: string): string {
  try {
    return new Date(iso).toISOString().replace('T', ' ').slice(0, 19)
  } catch {
    return iso
  }
}

export function traceSnippet(trace: string | undefined): string {
  const t = (trace || '').trim()
  if (!t) return '—'
  return t.length > 400 ? `${t.slice(0, 400)}…` : t
}

export function sampleHasCliLogs(s: TelemetrySampleRow): boolean {
  const m = s.metadata_json
  if (!m || typeof m !== 'object') return false
  const q = m.cli_qos_attempts
  return Boolean(
    m.cli_stdout ||
      m.cli_stderr ||
      (Array.isArray(q) && q.length > 0),
  )
}

export function formatCliSampleDetail(s: TelemetrySampleRow): string {
  const m = s.metadata_json
  if (!m || typeof m !== 'object') return ''
  const parts: string[] = []
  const pick = (k: string) => {
    const v = m[k]
    if (v === undefined || v === null || v === '') return
    parts.push(`${k}: ${typeof v === 'string' ? v : JSON.stringify(v)}`)
  }
  pick('cli_runner')
  pick('cli_image')
  pick('job')
  pick('namespace')
  pick('exit_code')
  if (typeof m.cli_stdout === 'string' && m.cli_stdout) {
    parts.push('--- cli_stdout ---\n' + m.cli_stdout)
  }
  if (typeof m.cli_stderr === 'string' && m.cli_stderr) {
    parts.push('--- cli_stderr ---\n' + m.cli_stderr)
  }
  if (Array.isArray(m.cli_qos_attempts) && m.cli_qos_attempts.length) {
    parts.push('--- cli_qos_attempts ---\n' + JSON.stringify(m.cli_qos_attempts, null, 2))
  }
  return parts.join('\n\n')
}
