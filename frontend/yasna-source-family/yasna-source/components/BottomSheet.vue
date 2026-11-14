<template>
  <Teleport to="body">
    <Transition name="sheet">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-6"
      >
        <button
          type="button"
          class="absolute inset-0 h-full w-full cursor-default bg-ink/40 backdrop-blur-[2px]"
          aria-label="Закрыть"
          @click="$emit('close')"
        ></button>
        <div
          class="relative z-10 max-h-[92vh] w-full overflow-y-auto rounded-t-[26px] border border-line bg-card p-5 shadow-lift sm:max-w-lg sm:rounded-[26px]"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
        >
          <div
            class="mx-auto mb-4 h-1 w-10 rounded-full bg-line sm:hidden"
          ></div>
          <div class="mb-4 flex items-start justify-between gap-4">
            <div>
              <h2 class="section-title">{{ title }}</h2>
              <p v-if="subtitle" class="mt-1 text-sm leading-snug text-muted">
                {{ subtitle }}
              </p>
            </div>
            <button
              type="button"
              class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-muted transition hover:bg-paper hover:text-ink"
              aria-label="Закрыть"
              @click="$emit('close')"
            >
              <i class="fa-solid fa-xmark" aria-hidden="true"></i>
            </button>
          </div>
          <slot />
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onUnmounted, watch } from "vue";

const props = defineProps<{
  open: boolean;
  title: string;
  subtitle?: string;
}>();
defineEmits<{ close: [] }>();

watch(
  () => props.open,
  (open) => {
    if (typeof document === "undefined") return;
    document.body.style.overflow = open ? "hidden" : "";
  },
);

onUnmounted(() => {
  if (typeof document === "undefined") return;
  document.body.style.overflow = "";
});
</script>
