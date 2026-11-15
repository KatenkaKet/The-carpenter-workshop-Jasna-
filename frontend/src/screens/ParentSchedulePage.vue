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

      <ManageBookings />
    </div>
  </ParentShell>
</template>

<script setup lang="ts">
import ManageBookings from "../components/ManageBookings.vue";
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
