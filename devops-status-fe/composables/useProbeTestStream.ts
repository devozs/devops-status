/** Matches devops-status-be/internal/middleware/csrf.go */
const CSRF_COOKIE = 'csrf_token'
const CSRF_HEADER = 'X-CSRF-Token'

function readCsrfCookie(): string | undefined {
  if (import.meta.server || typeof document === 'undefined') return undefined
  const m = document.cookie.match(new RegExp(`(?:^|;\\s*)${CSRF_COOKIE}=([^;]*)`))
  if (!m?.[1]) return undefined
  try {
    return decodeURIComponent(m[1])
  } catch {
    return m[1]
  }
}

function resolveApiBase(): string {
  const config = useRuntimeConfig()
  const rawPublic = config.public.apiBase as string | undefined
  const rawInternal = (config as { apiBaseInternal?: string }).apiBaseInternal as string | undefined

  if (import.meta.server && typeof rawInternal === 'string' && rawInternal.trim() !== '') {
    return rawInternal.replace(/\/$/, '')
  }

  if (typeof rawPublic === 'string' && rawPublic.trim() !== '') {
    return rawPublic.replace(/\/$/, '')
  }

  if (import.meta.dev) {
    return 'http://127.0.0.1:8080'
  }

  if (import.meta.server) {
    try {
      return useRequestURL().origin
    } catch {
      return ''
    }
  }
  return ''
}

function parseSSEBlock(block: string): { event: string; data: string } | null {
  const lines = block.split('\n')
  let event = 'message'
  const dataLines: string[] = []
  for (const line of lines) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trimStart())
    }
  }
  if (dataLines.length === 0) return null
  return { event, data: dataLines.join('\n') }
}

export type ProbeStreamHandlers = {
  onHost?: (meta: Record<string, unknown>) => void
  onLog?: (stream: string, chunkUtf8: string) => void
  onHeartbeat?: () => void
}

/** Thrown when the request was aborted (user Stop / navigation). */
export class ProbeStreamAbortedError extends Error {
  override readonly name = 'ProbeStreamAbortedError'
  constructor(message = 'Verify cancelled') {
    super(message)
    Object.setPrototypeOf(this, new.target.prototype)
  }
}

export type PostProbeTestStreamOptions = {
  signal?: AbortSignal
}

function isAbortError(e: unknown): boolean {
  if (e instanceof DOMException && e.name === 'AbortError') return true
  if (e instanceof Error && e.name === 'AbortError') return true
  return false
}

/**
 * POST /api/admin/probes/test/stream — same JSON body as /probes/test; parses SSE until `done`.
 */
export async function postProbeTestStream(
  bodyJson: string,
  handlers: ProbeStreamHandlers,
  options?: PostProbeTestStreamOptions,
): Promise<Record<string, unknown>> {
  const method = 'POST'
  const base = resolveApiBase()
  const pathOnly = '/api/admin/probes/test/stream'
  if (import.meta.client && !readCsrfCookie()) {
    await $fetch(`${base}/api/admin/meta`, {
      method: 'GET',
      credentials: 'include',
    })
  }
  const csrf = readCsrfCookie()

  let res: Response
  try {
    res = await fetch(`${base}${pathOnly}`, {
      method,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(csrf ? { [CSRF_HEADER]: csrf } : {}),
      },
      body: bodyJson,
      signal: options?.signal,
    })
  } catch (e: unknown) {
    if (isAbortError(e)) throw new ProbeStreamAbortedError()
    throw e
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const j = (await res.json()) as { error?: string }
      if (j?.error) msg = j.error
    } catch {
      const t = await res.text()
      if (t) msg = t.slice(0, 500)
    }
    throw new Error(msg)
  }

  const reader = res.body?.getReader()
  if (!reader) {
    throw new Error('no response body')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let donePayload: Record<string, unknown> | null = null

  while (true) {
    let value: Uint8Array | undefined
    let done: boolean
    try {
      const r = await reader.read()
      value = r.value
      done = r.done
    } catch (e: unknown) {
      if (isAbortError(e)) throw new ProbeStreamAbortedError()
      throw e
    }
    if (value) {
      buffer += decoder.decode(value, { stream: true })
    }
    while (true) {
      const sep = buffer.indexOf('\n\n')
      if (sep === -1) break
      const block = buffer.slice(0, sep)
      buffer = buffer.slice(sep + 2)
      const parsed = parseSSEBlock(block)
      if (!parsed) continue
      if (parsed.event === 'host') {
        try {
          const meta = JSON.parse(parsed.data) as { meta?: Record<string, unknown> }
          if (meta.meta && handlers.onHost) handlers.onHost(meta.meta)
        } catch {
          /* ignore */
        }
      } else if (parsed.event === 'log') {
        try {
          const row = JSON.parse(parsed.data) as { stream?: string; chunk?: string }
          const stream = String(row.stream || 'stdout')
          const b64 = row.chunk
          if (b64 && handlers.onLog) {
            const bin = atob(b64)
            const bytes = new Uint8Array(bin.length)
            for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
            const chunkUtf8 = new TextDecoder().decode(bytes)
            handlers.onLog(stream, chunkUtf8)
          }
        } catch {
          /* ignore */
        }
      } else if (parsed.event === 'heartbeat') {
        handlers.onHeartbeat?.()
      } else if (parsed.event === 'done') {
        try {
          donePayload = JSON.parse(parsed.data) as Record<string, unknown>
        } catch {
          donePayload = { success: false, error: 'invalid done payload' }
        }
      }
    }
    if (done) {
      break
    }
  }

  if (!donePayload) {
    throw new Error('stream ended without done event')
  }
  return donePayload
}
