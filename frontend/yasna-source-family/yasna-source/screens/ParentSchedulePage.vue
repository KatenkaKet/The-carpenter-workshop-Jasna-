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
    active="schedule"
    @logout="logout"
  >
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Кабинет семьи</p>
        <h1 class="page-title mt-1.5">Расписание</h1>
        <p class="mt-1.5 text-sm text-muted">
          Здравствуйте, {{ firstName }}. Выберите день и запишите ребёнка на
          занятие.
        </p>
      </div>
      <a :href="`/yasna/parent-day?date=${DEMO_TODAY}`" class="btn-primary">
        <i class="fa-solid fa-calendar-plus text-xs" aria-hidden="true"></i>
        Выбрать занятие
      </a>
    </div>

    <div v-if="children.length" class="mt-3 flex flex-wrap gap-1.5">
      <span
        v-for="child in children"
        :key="child.id"
        class="chip bg-card text-ink ring-1 ring-line"
      >
        <i
          class="fa-solid fa-child text-[9px] text-honey"
          aria-hidden="true"
        ></i>
        {{ child.name }}, {{ child.age }}
      </span>
    </div>

    <div class="mt-5 grid gap-4 lg:grid-cols-[1.02fr_0.98fr]">
      <MonthCalendar
        :selected="DEMO_TODAY"
        :lessons="state.lessons"
        base-path="/yasna/parent-day"
      />

      <section class="min-w-0" aria-labelledby="upcoming-title">
        <div class="flex items-baseline justify-between gap-3">
          <h2 id="upcoming-title" class="section-title">Ближайшие занятия</h2>
          <span class="text-[11px] font-semibold text-muted">
            {{ upcoming.length }}
            {{ plural(upcoming.length, "запись", "записи", "записей") }}
          </span>
        </div>

        <div v-if="upcoming.length" class="mt-3 space-y-3">
          <article
            v-for="row in upcoming"
            :key="row.booking.id"
            class="card p-4"
          >
            <div class="flex items-start gap-3">
              <div class="w-12 shrink-0 rounded-xl bg-paper py-1.5 text-center">
                <p
                  class="text-[9px] font-bold uppercase tracking-wide text-muted"
                >
                  {{ monthShort(row.lesson.date) }}
                </p>
                <p class="font-display text-lg font-semibold leading-tight">
                  {{ dayNumber(row.lesson.date) }}
                </p>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-start justify-between gap-2">
                  <h3 class="text-sm font-bold leading-snug">
                    {{ row.lesson.title }}
                  </h3>
                  <span class="chip shrink-0" :class="statusClass(row)">
                    {{ statusLabel(row) }}
                  </span>
                </div>
                <p class="mt-0.5 text-[11px] text-muted">
                  {{ row.lesson.time }} · {{ row.child?.name }} ·
                  {{ row.lesson.kind === "group" ? "группа" : "индивидуально" }}
                </p>
              </div>
            </div>

            <div
              v-if="row.booking.status === 'confirmed'"
              class="mt-3 flex items-center justify-between gap-2 rounded-xl bg-forest-soft/60 px-3 py-2.5"
            >
              <p class="text-[11px] leading-snug">
                <b>Вы записаны</b><br />
                <span class="text-muted">оплата: {{ paidLabel(row) }}</span>
              </p>
              <a
                href="/yasna/parent-payments"
                class="text-[10px] font-bold text-honey"
              >
                Оплаты
                <i
                  class="fa-solid fa-chevron-right text-[8px]"
                  aria-hidden="true"
                ></i>
              </a>
            </div>
          </article>
        </div>

        <div v-else class="card mt-3 px-5 py-8 text-center">
          <span
            class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
          >
            <i class="fa-solid fa-calendar-plus" aria-hidden="true"></i>
          </span>
          <p class="mt-3 font-display text-lg font-semibold">
            Пока нет записей
          </p>
          <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
            Выберите день в календаре — покажем свободные занятия.
          </p>
          <a
            :href="`/yasna/parent-day?date=${DEMO_TODAY}`"
            class="btn-primary mt-4"
          >
            Выбрать занятие
          </a>
        </div>

        <div class="mt-3 rounded-2xl border border-line/80 bg-card p-4">
          <p class="text-xs font-bold">Как проходит запись</p>
          <ol class="mt-2 space-y-2 text-[11px] leading-relaxed text-muted">
            <li class="flex gap-2">
              <span
                class="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-paper text-[9px] font-bold"
                >1</span
              >
              Покупаете 1 посещение или абонемент в «Тарифах» — либо кабинет
              предложит это при записи.
            </li>
            <li class="flex gap-2">
              <span
                class="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-paper text-[9px] font-bold"
                >2</span
              >
              Выбираете день и занятие, оплачиваете и записываетесь.
            </li>
            <li class="flex gap-2">
              <span
                class="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-paper text-[9px] font-bold"
                >3</span
              >
              Запись создаётся сразу, без ожидания. Чек появится после загрузки
              мастером.
            </li>
          </ol>
        </div>
      </section>
    </div>
  </ParentShell>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import MonthCalendar from "../components/MonthCalendar.vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  DEMO_TODAY,
  dateParts,
  MONTHS_GENITIVE,
  plural,
  type Child,
  type Lesson,
  type ParentBooking,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";

interface UpcomingRow {
  booking: ParentBooking;
  lesson: Lesson;
  child: Child | undefined;
}

const { state, bookingsForFamily, lessonById, childById } = useDemo();
const { family, children, ready, error, loginAs, logout } = useParent();

const firstName = computed(() => family.value?.parent.split(" ")[0] ?? "");

const upcoming = computed<UpcomingRow[]>(() => {
  if (!family.value) return [];
  const result: UpcomingRow[] = [];
  for (const booking of bookingsForFamily(family.value.id)) {
    if (booking.status !== "pending" && booking.status !== "confirmed")
      continue;
    const lesson = lessonById(booking.lessonId);
    if (!lesson || lesson.date < DEMO_TODAY) continue;
    result.push({
      booking,
      lesson,
      child: childById(booking.childId),
    });
  }
  return result.sort((a, b) =>
    `${a.lesson.date}${a.lesson.time}`.localeCompare(
      `${b.lesson.date}${b.lesson.time}`,
    ),
  );
});

function monthShort(iso: string): string {
  const { month } = dateParts(iso);
  return (MONTHS_GENITIVE[month] ?? "").slice(0, 3);
}

function dayNumber(iso: string): string {
  return String(dateParts(iso).day);
}

function statusLabel(row: UpcomingRow): string {
  if (row.booking.status === "confirmed") return "Записан";
  if (row.booking.status === "pending") return "Ждёт подтверждения";
  if (row.booking.status === "rejected") return "Отклонено";
  return "Отменено";
}

function statusClass(row: UpcomingRow): string {
  if (row.booking.status === "confirmed") return "bg-forest-soft text-forest";
  if (row.booking.status === "pending") return "bg-honey-soft text-honey";
  return "bg-paper text-muted";
}

function paidLabel(row: UpcomingRow): string {
  return row.booking.paidWith === "single" ? "разовое посещение" : "абонемент";
}
</script>
