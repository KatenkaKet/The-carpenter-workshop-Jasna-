<template>
  <ParentAccess v-if="ready && !family" :error="error" @demo="loginAs" />

  <div
    v-else-if="!ready"
    class="grid min-h-screen place-items-center text-center"
  >
    <div>
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-forest text-card"
      >
        <i class="fa-solid fa-seedling" aria-hidden="true"></i>
      </span>
      <p class="mt-3 text-sm font-semibold text-muted">Открываем кабинет…</p>
    </div>
  </div>

  <ParentShell
    v-else-if="family"
    :family="family"
    active="history"
    @logout="logout"
  >
    <div>
      <p class="eyebrow">Посещения детей</p>
      <h1 class="page-title mt-1.5">История посещений</h1>
      <p class="mt-1.5 text-sm text-muted">
        Все занятия, отметки мастера и списания по абонементу.
      </p>
    </div>

    <div class="mt-4 grid grid-cols-3 gap-2">
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Посещений
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold text-forest">
          {{ visitedCount }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Пропусков
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold text-clay">
          {{ missedCount }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Отмен
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold">
          {{ cancelledCount }}
        </p>
      </div>
    </div>

    <div v-if="children.length > 1" class="mt-4 flex flex-wrap gap-1.5">
      <button
        type="button"
        class="rounded-full border px-3 py-1.5 text-[11px] font-bold transition"
        :class="
          selectedChildId === 'all'
            ? 'border-forest bg-forest text-card'
            : 'border-line bg-card text-muted hover:text-ink'
        "
        @click="selectedChildId = 'all'"
      >
        Все дети
      </button>
      <button
        v-for="child in children"
        :key="child.id"
        type="button"
        class="rounded-full border px-3 py-1.5 text-[11px] font-bold transition"
        :class="
          selectedChildId === child.id
            ? 'border-forest bg-forest text-card'
            : 'border-line bg-card text-muted hover:text-ink'
        "
        @click="selectedChildId = child.id"
      >
        {{ child.name }}
      </button>
    </div>

    <ul v-if="rows.length" class="card mt-4 divide-y divide-line/70">
      <li
        v-for="row in rows"
        :key="`${row.lesson.id}:${row.child.id}`"
        class="flex items-start gap-3 p-3.5"
      >
        <div class="w-11 shrink-0 rounded-xl bg-paper py-1.5 text-center">
          <p class="text-[9px] font-bold uppercase tracking-wide text-muted">
            {{ monthShort(row.lesson.date) }}
          </p>
          <p class="font-display text-base font-semibold leading-tight">
            {{ dayNumber(row.lesson.date) }}
          </p>
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-start justify-between gap-2">
            <h2 class="text-sm font-bold leading-snug">
              {{ row.lesson.title }}
            </h2>
            <span class="chip shrink-0" :class="statusClass(row.mark?.status)">
              {{ statusLabel(row.mark?.status) }}
            </span>
          </div>
          <p class="mt-0.5 text-[11px] text-muted">
            {{ row.child.name }} · {{ row.lesson.time }} ·
            {{ row.lesson.kind === "group" ? "группа" : "индивидуально" }}
          </p>
          <p class="mt-1 text-[10px] font-semibold text-muted">
            {{ row.mark?.charged ? "Списано 1 занятие" : "Без списания" }}
            <span v-if="row.remaining !== null">
              · осталось {{ row.remaining }}</span
            >
          </p>
        </div>
      </li>
    </ul>

    <div v-else class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-clock-rotate-left" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">История пока пустая</p>
      <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
        После первого занятия здесь появятся отметки мастера и списания.
      </p>
      <a href="/yasna/parent" class="btn-primary mt-4">Выбрать занятие</a>
    </div>
  </ParentShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  DEMO_NOW_MINUTES,
  DEMO_TODAY,
  dateParts,
  MONTHS_GENITIVE,
  timeToMinutes,
  type Child,
  type Lesson,
  type Mark,
  type MarkStatus,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";

interface VisitRow {
  lesson: Lesson;
  child: Child;
  mark: Mark | undefined;
  remaining: number | null;
}

const { state, markFor, subscriptionFor } = useDemo();
const { family, children, ready, error, loginAs, logout } = useParent();

const selectedChildId = ref("all");

watch(
  children,
  (list) => {
    if (
      selectedChildId.value !== "all" &&
      !list.some((child) => child.id === selectedChildId.value)
    ) {
      selectedChildId.value = "all";
    }
  },
  { immediate: true },
);

function isDone(lesson: Lesson): boolean {
  if (lesson.date < DEMO_TODAY) return true;
  if (lesson.date > DEMO_TODAY) return false;
  return timeToMinutes(lesson.time) + lesson.duration <= DEMO_NOW_MINUTES;
}

const rows = computed<VisitRow[]>(() => {
  if (!family.value) return [];
  const result: VisitRow[] = [];
  for (const child of children.value) {
    if (selectedChildId.value !== "all" && selectedChildId.value !== child.id) {
      continue;
    }
    for (const lesson of state.lessons) {
      if (!lesson.childIds.includes(child.id) || !isDone(lesson)) continue;
      const subscription = subscriptionFor(child.id);
      result.push({
        lesson,
        child,
        mark: markFor(lesson.id, child.id),
        remaining: subscription ? subscription.remaining : null,
      });
    }
  }
  return result.sort((a, b) =>
    `${b.lesson.date}${b.lesson.time}`.localeCompare(
      `${a.lesson.date}${a.lesson.time}`,
    ),
  );
});

const visitedCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "present").length,
);
const missedCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "absent").length,
);
const cancelledCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "cancel").length,
);

function monthShort(iso: string): string {
  const { month } = dateParts(iso);
  return (MONTHS_GENITIVE[month] ?? "").slice(0, 3);
}

function dayNumber(iso: string): string {
  return String(dateParts(iso).day);
}

function statusLabel(status: MarkStatus | undefined): string {
  if (status === "present") return "Посетил";
  if (status === "absent") return "Пропустил";
  if (status === "cancel") return "Отмена";
  return "Без отметки";
}

function statusClass(status: MarkStatus | undefined): string {
  if (status === "present") return "bg-forest-soft text-forest";
  if (status === "absent") return "bg-clay-soft text-clay";
  if (status === "cancel") return "bg-paper text-muted";
  return "bg-paper text-muted";
}
</script>
