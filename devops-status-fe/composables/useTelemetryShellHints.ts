import { reactive } from 'vue'
import type { ShellHintKind, TelemetryShellHintRow } from '~/types/telemetry'
import { SHELL_HINT_KINDS } from '~/types/telemetry'

type ApiFetch = (path: string, opts?: Record<string, unknown>) => Promise<unknown>

/**
 * Loads and caches GET /api/admin/telemetry-shell-hints for picker UIs.
 */
export function useTelemetryShellHints(apiFetch: ApiFetch) {
  const loaded = reactive(
    Object.fromEntries(SHELL_HINT_KINDS.map((k) => [k, false])) as Record<ShellHintKind, boolean>,
  )
  const byKind = reactive(
    Object.fromEntries(SHELL_HINT_KINDS.map((k) => [k, [] as TelemetryShellHintRow[]])) as Record<
      ShellHintKind,
      TelemetryShellHintRow[]
    >,
  )

  async function ensureLoaded(kind: ShellHintKind): Promise<void> {
    if (loaded[kind]) return
    const rows = (await apiFetch(`/api/admin/telemetry-shell-hints?kind=${encodeURIComponent(kind)}`)) as
      | TelemetryShellHintRow[]
      | unknown
    byKind[kind] = Array.isArray(rows) ? rows : []
    loaded[kind] = true
  }

  async function ensureAllLoaded(): Promise<void> {
    await Promise.all(SHELL_HINT_KINDS.map((k) => ensureLoaded(k)))
  }

  function invalidate() {
    for (const k of SHELL_HINT_KINDS) {
      loaded[k] = false
      byKind[k] = []
    }
  }

  return { byKind, loaded, ensureLoaded, ensureAllLoaded, invalidate }
}
