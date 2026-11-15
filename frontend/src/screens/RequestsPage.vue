<template>
  <AppShell active="schedule">
    <a href="/yasna" class="btn-quiet -ml-2 !min-h-0 !px-2 text-xs">
      <i class="fa-solid fa-chevron-left text-[10px]" aria-hidden="true"></i>
      Расписание
    </a>

    <div class="mt-3">
      <p class="eyebrow">Записи из кабинета родителя</p>
      <h1 class="page-title mt-1.5">Записи</h1>
      <p class="mt-1.5 text-sm text-muted">
        Родитель оплачивает и записывается сразу. Подтверждение мастером не
        требуется — запись уже активна.
      </p>
    </div>

    <section class="mt-5" aria-labelledby="all-title">
      <div class="flex items-baseline justify-between">
        <h2 id="all-title" class="section-title">Все записи</h2>
        <span class="text-[10px] font-semibold text-muted">{{
          all.length
        }}</span>
      </div>
      <div v-if="all.length" class="card mt-3 divide-y divide-line/70">
        <div
          v-for="row in all"
          :key="row.booking.id"
          class="flex items-center gap-3 p-3.5"
        >
          <span
            class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-[10px] font-bold"
            :class="avatarTone(row.booking.childId)"
          >
            {{ initials(row.child?.name ?? "Р") }}
          </span>
          <div class="min-w-0 flex-1">
            <p class="text-xs font-bold">
              {{ row.child?.name }} · {{ row.lesson?.title }}
            </p>
            <p class="text-[10px] text-muted">
              {{ row.family?.parent }} ·
              {{
                row.lesson
                  ? `${formatDay(row.lesson.date)}, ${row.lesson.time}`
                  : ""
              }}
            </p>
            <p class="mt-0.5 text-[10px] font-semibold text-forest">
              {{
                row.booking.paidWith === "single"
                  ? "оплачено разовое посещение"
                  : "оплачено по абонементу"
              }}
            </p>
          </div>
          <span
            class="chip shrink-0"
            :class="statusMeta[row.booking.status].className"
          >
            {{ statusMeta[row.booking.status].label }}
          </span>
        </div>
      </div>
      <div v-else class="card mt-3 px-5 py-10 text-center">
        <span
          class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
        >
          <i class="fa-solid fa-inbox" aria-hidden="true"></i>
        </span>
        <p class="mt-3 font-display text-lg font-semibold">Записей пока нет</p>
        <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
          Когда родитель запишется из кабинета, запись появится здесь.
        </p>
      </div>
    </section>
  </AppShell>
</template>

<script setup lang="ts">
import { computed } from "vue";
import AppShell from "../components/AppShell.vue";
import {
  avatarTone,
  formatDay,
  initials,
  type Child,
  type Family,
  type Lesson,
  type ParentBooking,
  type ParentBookingStatus,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";

interface RequestRow {
  booking: ParentBooking;
  child: Child | undefined;
  family: Family | undefined;
  lesson: Lesson | undefined;
}

const { state, childById, familyById, lessonById } = useDemo();

function enrich(booking: ParentBooking): RequestRow {
  return {
    booking,
    child: childById(booking.childId),
    family: familyById(booking.familyId),
    lesson: lessonById(booking.lessonId),
  };
}

const all = computed(() =>
  state.bookings
    .slice()
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    .map(enrich),
);

const statusMeta: Record<
  ParentBookingStatus,
  { label: string; className: string }
> = {
  pending: { label: "Ждёт", className: "bg-honey-soft text-honey" },
  confirmed: { label: "Записан", className: "bg-forest-soft text-forest" },
  rejected: { label: "Отменена", className: "bg-clay-soft text-clay" },
  cancelled: { label: "Отменена", className: "bg-paper text-muted" },
};
</script>
