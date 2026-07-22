<template>
  <button
    type="button"
    @click="toggle"
    class="toggle-control"
    :class="{ 'toggle-control--active': modelValue }"
    :disabled="disabled"
    role="switch"
    :aria-checked="modelValue"
    :aria-disabled="disabled ? 'true' : undefined"
    :aria-label="ariaLabel"
  >
    <span
      class="toggle-thumb"
      aria-hidden="true"
    />
  </button>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue: boolean
  disabled?: boolean
  ariaLabel?: string
}>(), {
  disabled: false
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

function toggle() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>
