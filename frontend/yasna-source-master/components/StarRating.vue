<template>
  <div
    class="flex items-center gap-1"
    role="group"
    aria-label="Оценка за занятие"
  >
    <button
      v-for="value in 5"
      :key="value"
      type="button"
      class="grid h-9 w-8 place-items-center rounded-lg text-[15px] transition hover:bg-honey-soft"
      :class="value <= shown ? 'text-honey' : 'text-line'"
      :aria-label="`Оценка ${value}`"
      :aria-pressed="value === modelValue"
      @click="$emit('update:modelValue', value)"
      @mouseenter="hovered = value"
      @mouseleave="hovered = 0"
    >
      <i class="fa-solid fa-star" aria-hidden="true"></i>
    </button>
    <span v-if="modelValue" class="ml-1 text-[11px] font-bold text-muted">{{
      caption
    }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";

const props = defineProps<{ modelValue?: number }>();
defineEmits<{ "update:modelValue": [value: number] }>();

const hovered = ref(0);
const shown = computed(() => hovered.value || props.modelValue || 0);
const caption = computed(() => {
  const labels: Record<number, string> = {
    1: "нужно стараться",
    2: "есть над чем работать",
    3: "старание",
    4: "хорошо",
    5: "отлично",
  };
  return labels[props.modelValue ?? 0] ?? "";
});
</script>
