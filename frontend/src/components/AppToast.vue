<template>
  <Transition name="toast">
    <div
      v-if="message"
      class="pointer-events-none fixed inset-x-0 bottom-24 z-[60] flex justify-center px-4 lg:bottom-8"
      role="status"
      aria-live="polite"
    >
      <div
        class="flex items-center gap-2 rounded-full bg-ink px-4 py-2.5 text-sm font-semibold text-card shadow-lift"
      >
        <i class="fa-solid fa-circle-check text-honey" aria-hidden="true"></i>
        {{ message }}
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
defineProps<{ message: string }>();
</script>
