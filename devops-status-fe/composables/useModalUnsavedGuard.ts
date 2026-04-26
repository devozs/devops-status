import { ref } from 'vue'

export type UnsavedCloseOptions = {
  isDirty: () => boolean
  message: string
  title?: string
  confirmLabel?: string
}

export function useModalUnsavedGuard() {
  const unsavedDialogOpen = ref(false)
  const unsavedDialogMessage = ref('')
  const unsavedDialogTitle = ref('Discard unsaved changes?')
  const unsavedConfirmLabel = ref('Discard')
  let pendingClose: (() => void) | null = null

  function requestClose(performClose: () => void, opts: UnsavedCloseOptions) {
    if (!opts.isDirty()) {
      performClose()
      return
    }
    pendingClose = performClose
    unsavedDialogMessage.value = opts.message
    unsavedDialogTitle.value = (opts.title && opts.title.trim()) || 'Discard unsaved changes?'
    unsavedConfirmLabel.value = (opts.confirmLabel && opts.confirmLabel.trim()) || 'Discard'
    unsavedDialogOpen.value = true
  }

  function confirmUnsavedDialog() {
    const fn = pendingClose
    pendingClose = null
    if (fn) {
      fn()
    }
    unsavedDialogOpen.value = false
  }

  function cancelUnsavedDialog() {
    unsavedDialogOpen.value = false
    pendingClose = null
  }

  return {
    requestClose,
    unsavedDialogOpen,
    unsavedDialogMessage,
    unsavedDialogTitle,
    unsavedConfirmLabel,
    confirmUnsavedDialog,
    cancelUnsavedDialog,
  }
}
