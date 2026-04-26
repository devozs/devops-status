<template>
  <details class="telemetry-link-help">
    <summary class="telemetry-link-help__summary">Help</summary>
    <div class="telemetry-link-help__body">
      <p>
        <strong>Interval</strong>
        — How often the scheduler runs this probe (minutes between runs) for this link.
      </p>
      <p>
        <strong>Window</strong>
        — How many of the latest operational results are read when deciding status. Larger windows smooth out noise but react more slowly.
      </p>
      <p>
        <strong>Fail→down</strong>
        — How many <em>consecutive</em> failed checks (within that window, evaluated from newest samples backward) are required before the target is treated as down and an incident can open.
      </p>
      <p>
        <strong>Success→up</strong>
        — How many consecutive successful checks are required before the target is treated as recovered / up again.
      </p>
      <p class="telemetry-link-help__note">
        QoS-tier probes use the same interval; operational up/down logic applies to pass/fail of the check. Service and environment telemetry links use these fields the same way.
      </p>
    </div>
  </details>
</template>

<style scoped>
.telemetry-link-help {
  margin: 0 0 12px;
  font-size: 0.85rem;
  line-height: 1.45;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 6px);
  padding: 0 10px;
  background: var(--color-bg-muted, rgba(0, 0, 0, 0.02));
}
.telemetry-link-help__summary {
  cursor: pointer;
  font-weight: 600;
  padding: 8px 0;
  list-style-position: outside;
}
.telemetry-link-help__body {
  padding: 0 0 10px 4px;
  color: var(--color-text-secondary);
}
.telemetry-link-help__body p {
  margin: 0 0 8px;
}
.telemetry-link-help__body p:last-child {
  margin-bottom: 0;
}
.telemetry-link-help__note {
  font-size: 0.8rem;
  opacity: 0.95;
}
</style>
