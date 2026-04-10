<script setup lang="ts">
const props = defineProps<{
  steps: string[]
  currentStep: number
}>()

function state(i: number): 'done' | 'active' | 'todo' {
  if (i < props.currentStep) return 'done'
  if (i === props.currentStep) return 'active'
  return 'todo'
}
</script>

<template>
  <div class="modal-stepper" role="navigation" aria-label="Steps">
    <template v-for="(label, i) in steps" :key="i">
      <div class="modal-stepper__step" :class="'modal-stepper__step--' + state(i)">
        <span class="modal-stepper__dot-wrap">
          <span class="modal-stepper__dot" />
        </span>
        <span class="modal-stepper__label">{{ label }}</span>
      </div>
      <div
        v-if="i < steps.length - 1"
        class="modal-stepper__connector"
        :class="{ 'modal-stepper__connector--done': i < currentStep }"
        aria-hidden="true"
      />
    </template>
  </div>
</template>

<style scoped>
.modal-stepper {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  flex-wrap: wrap;
  gap: 0;
  margin: 0 0 22px;
  padding-bottom: 4px;
}

.modal-stepper__step {
  display: flex;
  align-items: center;
  gap: 8px;
}

.modal-stepper__dot-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
}

.modal-stepper__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid var(--color-border-strong);
  background: var(--color-bg);
  box-sizing: border-box;
}

.modal-stepper__step--active .modal-stepper__dot {
  width: 12px;
  height: 12px;
  border-width: 2px;
  border-color: var(--color-primary);
  background: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-primary-muted);
}

.modal-stepper__step--done .modal-stepper__dot {
  border-color: var(--color-primary);
  background: var(--color-primary);
}

.modal-stepper__label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  white-space: nowrap;
}

.modal-stepper__step--active .modal-stepper__label {
  color: var(--color-text);
  font-weight: 600;
}

.modal-stepper__step--done .modal-stepper__label {
  color: var(--color-text);
}

.modal-stepper__connector {
  width: 24px;
  height: 2px;
  margin: 0 6px;
  background: var(--color-border);
  border-radius: 1px;
  flex-shrink: 0;
  align-self: center;
}

.modal-stepper__connector--done {
  background: var(--color-primary-muted);
}
</style>
