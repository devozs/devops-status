/** Strip ANSI escape sequences for plain-text clipboard output. */
export function stripAnsi(s: string): string {
  return s
    .replace(/\x1b\[[\d;?]*[A-Za-z]/g, '')
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    .replace(/\x1b[\][()#][\d;?]*/g, '')
}
