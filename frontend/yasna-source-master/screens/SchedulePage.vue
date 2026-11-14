<template>
  <AppShell active="schedule">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Мастерская «Ясна» · Сургут</p>
        <h1 class="page-title mt-1.5">Расписание</h1>
        <p class="mt-1.5 text-sm text-muted">
          {{ monthLabel }} · сегодня {{ formatDay(DEMO_TODAY) }},
          {{ formatWeekday(DEMO_TODAY) }}
        </p>
      </div>
      <a :href="`/yasna/day?date=${DEMO_TODAY}`" class="btn-primary">
        <i class="fa-solid fa-plus text-xs" aria-hidden="true"></i>
        Записать на занятие
      </a>
    </div>

    <div class="mt-5 grid grid-cols-1 gap-4 lg:grid-cols-[1.02fr_0.98fr]">
      <MonthCalendar :selected="DEMO_TODAY" :lessons="state.lessons" />

      <div class="min-w-0 space-y-4">
        <section class="card p-4 sm:p-5" aria-labelledby="today-title">
          <div class="flex items-start justify-between gap-3">
            <div>
              <p class="eyebrow">Сегодня</p>
              <h2
                id="today-title"
                class="mt-1 font-display text-xl font-semibold tracking-tight"
              >
                {{ formatDateLong(DEMO_TODAY) }}
              </h2>
            </div>
            <a
              :href="`/yasna/day?date=${DEMO_TODAY}`"
              class="btn-quiet -mt-1 !px-2 text-xs"
            >
              Весь день
              <i
                class="fa-solid fa-chevron-right text-[10px]"
                aria-hidden="true"
              ></i>
            </a>
          </div>

          <div class="mt-3 grid grid-cols-3 gap-2">
            <div class="rounded-xl bg-paper/80 px-3 py-2">
              <p
                class="text-[10px] font-bold uppercase tracking-wider text-muted"
              >
                Занятий
              </p>
              <p class="mt-0.5 font-display text-xl font-semibold">
                {{ todayLessons.length }}
              </p>
            </div>
            <div class="rounded-xl bg-paper/80 px-3 py-2">
              <p
                class="text-[10px] font-bold uppercase tracking-wider text-muted"
              >
                Детей
              </p>
              <p class="mt-0.5 font-display text-xl font-semibold">
                {{ todayChildren }}
              </p>
            </div>
            <div class="rounded-xl bg-paper/80 px-3 py-2">
              <p
                class="text-[10px] font-bold uppercase tracking-wider text-muted"
              >
                Свободно
              </p>
              <p class="mt-0.5 font-display text-xl font-semibold">
                {{ freeSeats }}
              </p>
            </div>
          </div>

          <ul v-if="todayLessons.length" class="mt-3 divide-y divide-line/70">
            <li v-for="lesson in todayLessons" :key="lesson.id">
              <a
                :href="`/yasna/journal?date=${lesson.date}&lesson=${lesson.id}`"
                class="group flex items-start gap-3 py-3"
              >
                <span class="w-[46px] shrink-0">
                  <span class="block text-sm font-bold">{{ lesson.time }}</span>
                  <span class="mt-0.5 block text-[10px] text-muted"
                    >{{ lesson.duration }} мин</span
                  >
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-semibold leading-snug">{{
                    lesson.title
                  }}</span>
                  <span
                    class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-muted"
                  >
                    <i
                      :class="
                        lesson.kind === 'group'
                          ? 'fa-solid fa-user-group'
                          : 'fa-solid fa-user'
                      "
                      class="text-[10px]"
                      aria-hidden="true"
                    ></i>
                    {{ lesson.kind === "group" ? "группа" : "индивидуально" }} ·
                    {{ lesson.childIds.length }} из {{ lesson.capacity }}
                    <span
                      class="chip"
                      :class="statusMeta[statusOf(lesson)].className"
                    >
                      {{ statusMeta[statusOf(lesson)].label }}
                    </span>
                  </span>
                </span>
                <i
                  class="fa-solid fa-chevron-right mt-1.5 text-[10px] text-muted/40 transition group-hover:text-forest"
                  aria-hidden="true"
                ></i>
              </a>
            </li>
          </ul>

          <div
            v-else
            class="mt-4 rounded-xl border border-dashed border-line px-4 py-6 text-center"
          >
            <p class="text-sm font-semibold">На этот день занятий нет</p>
            <a
              :href="`/yasna/day?date=${DEMO_TODAY}`"
              class="mt-2 inline-block text-xs font-bold text-honey"
            >
              Создать занятие
            </a>
          </div>
        </section>

        <section
          v-if="attention.length"
          class="rounded-[22px] border border-honey/30 bg-honey-soft/55 p-4 sm:p-5"
          aria-labelledby="attention-title"
        >
          <div class="flex items-start gap-3">
            <span
              class="mt-0.5 grid h-9 w-9 shrink-0 place-items-center rounded-full bg-card text-honey"
            >
              <i
                class="fa-solid fa-triangle-exclamation"
                aria-hidden="true"
              ></i>
            </span>
            <div class="min-w-0 flex-1">
              <h2 id="attention-title" class="text-sm font-bold">
                Требуют внимания
              </h2>
              <ul class="mt-2 space-y-1.5">
                <li
                  v-for="row in attention"
                  :key="row.child.id"
                  class="flex items-baseline justify-between gap-3 text-xs"
                >
                  <span class="truncate">
                    <b>{{ row.child.name }}</b> · {{ row.sub.remaining }} из
                    {{ row.sub.total }}
                  </span>
                  <span
                    class="shrink-0 text-[10px] font-bold uppercase tracking-wide text-honey"
                  >
                    {{
                      row.sub.remaining === 0 ? "закончился" : "заканчивается"
                    }}
                  </span>
                </li>
              </ul>
              <a
                href="/yasna/subscriptions"
                class="mt-3 inline-flex items-center gap-1 text-[11px] font-bold text-ink underline decoration-ink/20 underline-offset-2"
              >
                Открыть абонементы
                <i
                  class="fa-solid fa-arrow-right text-[9px]"
                  aria-hidden="true"
                ></i>
              </a>
            </div>
          </div>
        </section>
      </div>
    </div>

    <div class="mt-4 grid gap-3 sm:grid-cols-2">
      <a
        v-for="tile in tiles"
        :key="tile.href"
        :href="tile.href"
        class="card group flex items-center gap-3 p-3.5 transition hover:border-forest/30 hover:shadow-lift"
      >
        <span
          class="grid h-11 w-11 shrink-0 place-items-center rounded-2xl text-[17px]"
          :class="tile.tone"
        >
          <i :class="tile.icon" aria-hidden="true"></i>
        </span>
        <span class="min-w-0 flex-1">
          <span class="block text-sm font-bold">{{ tile.title }}</span>
          <span class="mt-0.5 block truncate text-[11px] text-muted">{{
            tile.note
          }}</span>
        </span>
        <i
          class="fa-solid fa-chevron-right text-[10px] text-muted/40 transition group-hover:text-forest"
          aria-hidden="true"
        ></i>
      </a>
    </div>

    <p class="mt-5 text-center text-[11px] leading-relaxed text-muted/70">
      Демо-прототип · вымышленные данные · изменения сохраняются в этом браузере
    </p>

    <AppToast :message="toastMessage" />
  </AppShell>
</template>

<script setup lang="ts">
import { computed } from "vue";
import AppShell from "../components/AppShell.vue";
import AppToast from "../components/AppToast.vue";
import MonthCalendar from "../components/MonthCalendar.vue";
import {
  DEMO_TODAY,
  dateParts,
  formatDateLong,
  formatDay,
  formatMonth,
  formatWeekday,
  lessonStatus,
  type Lesson,
  type LessonStatus,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const { state, lessonsFor, subscriptionFor } = useDemo();
const { message: toastMessage, show: notify } = useToast();

const todayLessons = computed(() => lessonsFor(DEMO_TODAY));

const todayChildren = computed(() => {
  const ids = new Set<string>();
  for (const lesson of todayLessons.value) {
    for (const id of lesson.childIds) ids.add(id);
  }
  return ids.size;
});

const freeSeats = computed(() =>
  todayLessons.value.reduce(
    (sum, lesson) =>
      sum + Math.max(0, lesson.capacity - lesson.childIds.length),
    0,
  ),
);

const attention = computed(() => {
  const result: Array<{
    child: (typeof state.children)[number];
    sub: NonNullable<ReturnType<typeof subscriptionFor>>;
  }> = [];
  for (const child of state.children) {
    const sub = subscriptionFor(child.id);
    if (sub && sub.remaining <= 1) result.push({ child, sub });
  }
  return result.sort((a, b) => a.sub.remaining - b.sub.remaining).slice(0, 3);
});

const monthLabel = computed(() => {
  const today = dateParts(DEMO_TODAY);
  return formatMonth(today.year, today.month);
});

function statusOf(lesson: Lesson): LessonStatus {
  return lessonStatus(lesson);
}

const statusMeta: Record<LessonStatus, { label: string; className: string }> = {
  done: { label: "Завершено", className: "bg-paper text-muted" },
  now: { label: "Идёт", className: "bg-forest-soft text-forest" },
  soon: { label: "Скоро", className: "bg-honey-soft text-honey" },
  planned: { label: "Запланировано", className: "bg-paper text-muted" },
};

const tiles = [
  {
    href: "/yasna/journal",
    title: "Журнал посещаемости",
    note: "Отметки, пропуски и оценки",
    icon: "fa-solid fa-clipboard-check",
    tone: "bg-forest-soft text-forest",
  },
  {
    href: "/yasna/subscriptions",
    title: "Абонементы",
    note: "Остатки, сроки и правила",
    icon: "fa-solid fa-ticket",
    tone: "bg-honey-soft text-honey",
  },
  {
    href: "/yasna/clients",
    title: "Клиенты",
    note: "Родители и ученики",
    icon: "fa-solid fa-users",
    tone: "bg-clay-soft text-clay",
  },
  {
    href: "/yasna/finance",
    title: "Финансы",
    note: "Оплаты и чеки клиентов",
    icon: "fa-solid fa-ruble-sign",
    tone: "bg-[#EAE5F2] text-lilac",
  },
];
</script>
