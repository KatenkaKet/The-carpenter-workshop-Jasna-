<template>
  <section class="card min-w-0 p-4 sm:p-5" aria-label="Календарь занятий">
    <div class="flex items-center justify-between gap-3">
      <div>
        <p class="eyebrow">Календарь</p>
        <h2 class="mt-1 font-display text-xl font-semibold tracking-tight">
          {{ monthTitle }}
        </h2>
      </div>
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="grid h-9 w-9 place-items-center rounded-full border border-line bg-card text-muted transition hover:border-forest/30 hover:text-ink"
          aria-label="Предыдущий месяц"
          @click="shiftMonth(-1)"
        >
          <i class="fa-solid fa-chevron-left text-xs" aria-hidden="true"></i>
        </button>
        <button
          type="button"
          class="grid h-9 w-9 place-items-center rounded-full border border-line bg-card text-muted transition hover:border-forest/30 hover:text-ink"
          aria-label="Следующий месяц"
          @click="shiftMonth(1)"
        >
          <i class="fa-solid fa-chevron-right text-xs" aria-hidden="true"></i>
        </button>
      </div>
    </div>

    <div class="mt-4 grid grid-cols-7 gap-1">
      <span
        v-for="weekday in WEEKDAYS_SHORT"
        :key="weekday"
        class="pb-1 text-center text-[10px] font-bold uppercase tracking-wider text-muted/70"
      >
        {{ weekday }}
      </span>
    </div>

    <div class="grid grid-cols-7 gap-1">
      <template v-for="(cell, index) in cells" :key="index">
        <span v-if="!cell" class="h-[42px] sm:h-11"></span>
        <a
          v-else
          :href="`/yasna/day?date=${cell.iso}`"
          class="relative flex h-[42px] flex-col items-center justify-center gap-1 rounded-xl text-[13px] font-semibold transition sm:h-11"
          :class="cellClass(cell.iso)"
          :aria-label="`${cell.day} — открыть день`"
          :aria-current="cell.iso === selected ? 'date' : undefined"
        >
          <span>{{ cell.day }}</span>
          <span class="flex h-1.5 items-center gap-0.5">
            <span
              v-for="(kind, dotIndex) in dotsFor(cell.iso)"
              :key="dotIndex"
              class="h-1.5 w-1.5 rounded-full"
              :class="cell.iso === selected ? 'bg-card/80' : dotClass(kind)"
            ></span>
          </span>
        </a>
      </template>
    </div>

    <div
      class="mt-4 flex flex-wrap items-center gap-x-4 gap-y-2 border-t border-line/70 pt-3"
    >
      <span
        class="flex items-center gap-1.5 text-[11px] font-semibold text-muted"
      >
        <span class="h-1.5 w-1.5 rounded-full bg-honey"></span>
        групповое
      </span>
      <span
        class="flex items-center gap-1.5 text-[11px] font-semibold text-muted"
      >
        <span class="h-1.5 w-1.5 rounded-full bg-forest"></span>
        индивидуальное
      </span>
      <button
        v-if="!isCurrentMonth"
        type="button"
        class="ml-auto text-[11px] font-bold text-honey underline decoration-honey/40 underline-offset-2"
        @click="goToday"
      >
        К сегодняшнему дню
      </button>
    </div>

    <div class="mt-3 grid grid-cols-3 gap-2 border-t border-line/70 pt-3">
      <div class="rounded-xl bg-paper/70 px-2.5 py-2">
        <p class="text-[9px] font-bold uppercase tracking-wide text-muted">
          Занятий
        </p>
        <p class="mt-0.5 font-display text-lg font-semibold">
          {{ monthStats.lessons }}
        </p>
      </div>
      <div class="rounded-xl bg-paper/70 px-2.5 py-2">
        <p class="text-[9px] font-bold uppercase tracking-wide text-muted">
          Записей
        </p>
        <p class="mt-0.5 font-display text-lg font-semibold">
          {{ monthStats.records }}
        </p>
      </div>
      <div class="rounded-xl bg-paper/70 px-2.5 py-2">
        <p class="text-[9px] font-bold uppercase tracking-wide text-muted">
          Свободно
        </p>
        <p class="mt-0.5 font-display text-lg font-semibold">
          {{ monthStats.free }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import {
  DEMO_TODAY,
  WEEKDAYS_SHORT,
  dateParts,
  formatMonth,
  monthCells,
  type Lesson,
  type LessonKind,
} from "../shared/demo";

const props = defineProps<{
  selected: string;
  lessons: Lesson[];
}>();

const initial = dateParts(props.selected);
const viewYear = ref(initial.year);
const viewMonth = ref(initial.month);

const cells = computed(() => monthCells(viewYear.value, viewMonth.value));
const monthTitle = computed(() => formatMonth(viewYear.value, viewMonth.value));

const lessonsByDate = computed(() => {
  const map: Record<string, LessonKind[]> = {};
  for (const item of props.lessons) {
    const list = map[item.date] ?? [];
    if (!list.includes(item.kind)) list.push(item.kind);
    map[item.date] = list;
  }
  return map;
});

const monthStats = computed(() => {
  const prefix = `${viewYear.value}-${String(viewMonth.value + 1).padStart(2, "0")}`;
  let lessons = 0;
  let records = 0;
  let free = 0;
  for (const item of props.lessons) {
    if (!item.date.startsWith(prefix)) continue;
    lessons += 1;
    records += item.childIds.length;
    free += Math.max(0, item.capacity - item.childIds.length);
  }
  return { lessons, records, free };
});

const isCurrentMonth = computed(() => {
  const today = dateParts(DEMO_TODAY);
  return today.year === viewYear.value && today.month === viewMonth.value;
});

function shiftMonth(offset: number) {
  const next = new Date(Date.UTC(viewYear.value, viewMonth.value + offset, 1));
  viewYear.value = next.getUTCFullYear();
  viewMonth.value = next.getUTCMonth();
}

function goToday() {
  const today = dateParts(DEMO_TODAY);
  viewYear.value = today.year;
  viewMonth.value = today.month;
}

function dotsFor(iso: string): LessonKind[] {
  return (lessonsByDate.value[iso] ?? []).slice(0, 2);
}

function dotClass(kind: LessonKind): string {
  return kind === "group" ? "bg-honey" : "bg-forest";
}

function cellClass(iso: string): string {
  if (iso === props.selected) return "bg-forest text-card shadow-lift";
  if (iso === DEMO_TODAY)
    return "bg-honey-soft/70 text-ink ring-1 ring-honey/40";
  return "text-ink hover:bg-paper";
}
</script>
