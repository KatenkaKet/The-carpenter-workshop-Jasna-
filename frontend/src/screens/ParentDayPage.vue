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
    <a href="/yasna/parent" class="btn-quiet -ml-2 !min-h-0 !px-2 text-xs">
      <i class="fa-solid fa-chevron-left text-[10px]" aria-hidden="true"></i>
      Расписание
    </a>

    <div class="mt-3">
      <p class="eyebrow">{{ formatWeekday(date) }}</p>
      <h1 class="page-title mt-1.5">{{ formatDay(date) }}</h1>
      <p class="mt-1.5 text-sm text-muted">
        {{ lessons.length }}
        {{ plural(lessons.length, "занятие", "занятия", "занятий") }} ·
        {{ freeSeats }}
        {{
          plural(
            freeSeats,
            "свободное место",
            "свободных места",
            "свободных мест",
          )
        }}
      </p>
    </div>

    <div v-if="children.length > 1" class="mt-4">
      <p class="label">Кто идёт на занятие</p>
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="child in children"
          :key="child.id"
          type="button"
          class="flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-bold transition"
          :class="
            selectedChildId === child.id
              ? 'border-forest bg-forest text-card'
              : 'border-line bg-card text-ink hover:border-forest/30'
          "
          @click="selectedChildId = child.id"
        >
          <span
            class="grid h-5 w-5 place-items-center rounded-full text-[8px] font-bold"
            :class="
              selectedChildId === child.id
                ? 'bg-card/20 text-card'
                : avatarTone(child.id)
            "
          >
            {{ initials(child.name) }}
          </span>
          {{ child.name }}, {{ child.age }}
        </button>
      </div>
    </div>

    <div
      v-if="selectedChild"
      class="mt-3 flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line/80 bg-card px-3 py-2"
    >
      <p class="text-[11px] font-semibold text-muted">
        Оплачено:
        <b class="text-ink">{{ coverageLabel }}</b>
      </p>
      <a href="/yasna/parent-tariffs" class="text-[10px] font-bold text-honey">
        Купить тариф
        <i class="fa-solid fa-chevron-right text-[8px]" aria-hidden="true"></i>
      </a>
    </div>

    <div class="mt-3 flex flex-wrap gap-1.5">
      <button
        v-for="filter in filters"
        :key="filter.key"
        type="button"
        class="rounded-full border px-3 py-1.5 text-[11px] font-bold transition"
        :class="
          activeFilter === filter.key
            ? 'border-forest bg-forest text-card'
            : 'border-line bg-card text-muted hover:border-forest/30 hover:text-ink'
        "
        @click="activeFilter = filter.key"
      >
        {{ filter.label }}
      </button>
    </div>

    <div v-if="filteredLessons.length" class="mt-4 space-y-3">
      <article
        v-for="lesson in filteredLessons"
        :key="lesson.id"
        class="card p-4"
      >
        <div class="flex gap-3">
          <div class="w-[54px] shrink-0">
            <p class="text-base font-bold leading-tight">{{ lesson.time }}</p>
            <p class="mt-0.5 text-[10px] text-muted">
              {{ lesson.duration }} мин
            </p>
            <span
              class="mt-2 block h-1.5 w-1.5 rounded-full"
              :class="lesson.kind === 'group' ? 'bg-honey' : 'bg-forest'"
            ></span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <h2 class="text-sm font-bold leading-snug">
                  {{ lesson.title }} <span v-if="lesson.isMakeup" class="chip bg-honey-soft">Отработка</span>
                </h2>
                <p
                  class="mt-0.5 flex items-center gap-1.5 text-[11px] text-muted"
                >
                  <i
                    :class="
                      lesson.kind === 'group'
                        ? 'fa-solid fa-user-group'
                        : 'fa-solid fa-user'
                    "
                    class="text-[9px]"
                    aria-hidden="true"
                  ></i>
                  {{ lesson.kind === "group" ? "Групповое" : "Индивидуальное" }}
                  · {{ lesson.childIds.length }} из {{ lesson.capacity }}
                </p>
              </div>
              <span class="chip shrink-0" :class="chipClass(lesson)">
                {{ chipLabel(lesson) }}
              </span>
            </div>

            <div class="mt-3 flex items-end justify-between gap-2">
              <p class="text-[10px] font-semibold text-muted">
                {{ seatsLabel(lesson) }}
              </p>
              <button
                v-if="canBook(lesson)"
                type="button"
                class="btn-primary !min-h-[38px] shrink-0 !px-3 text-xs"
                @click="openBooking(lesson)"
              >
                <i class="fa-solid fa-plus text-[10px]" aria-hidden="true"></i>
                Записаться
              </button>
              <span
                v-else-if="bookingFor(lesson)"
                class="chip shrink-0 bg-forest-soft text-forest"
              >
                записаны
              </span>
            </div>
          </div>
        </div>
      </article>
    </div>

    <div v-else class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-calendar-xmark" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">
        Свободных занятий нет
      </p>
      <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
        Выберите другой день или другой формат.
      </p>
      <a href="/yasna/parent" class="btn-ghost mt-4">К календарю</a>
    </div>

    <ParentBookingSheet
      :open="bookingOpen"
      :lesson="bookingLesson"
      :child-id="selectedChild?.id ?? ''"
      :family-id="family.id"
      @close="bookingOpen = false"
      @booked="onBooked"
    />

    <AppToast :message="toastMessage" />
  </ParentShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import AppToast from "../components/AppToast.vue";
import ParentBookingSheet from "../components/ParentBookingSheet.vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  DEMO_NOW_MINUTES,
  DEMO_TODAY,
  avatarTone,
  formatDay,
  formatWeekday,
  initials,
  plural,
  timeToMinutes,
  type Lesson,
  type ParentBooking,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";
import { useToast } from "../shared/use-toast";

const props = defineProps<{ date: string }>();

const { state, lessonsFor, subscriptionFor } = useDemo();
const { family, children, ready, error, loginAs, logout } = useParent();
const { message: toastMessage, show: notify } = useToast();

const date = computed(() => props.date);
const activeFilter = ref<"all" | "group" | "individual">("all");
const selectedChildId = ref("");

watch(
  children,
  (list) => {
    if (!list.some((child) => child.id === selectedChildId.value)) {
      selectedChildId.value = list[0]?.id ?? "";
    }
  },
  { immediate: true },
);

const selectedChild = computed(
  () =>
    children.value.find((child) => child.id === selectedChildId.value) ??
    children.value[0],
);

const filters = [
  { key: "all", label: "Все занятия" },
  { key: "group", label: "Групповые" },
  { key: "individual", label: "Индивидуальные" },
] as const;

const lessons = computed(() => lessonsFor(date.value));
const filteredLessons = computed(() =>
  lessons.value.filter(
    (lesson) =>
      activeFilter.value === "all" || lesson.kind === activeFilter.value,
  ),
);
const freeSeats = computed(() =>
  lessons.value.reduce(
    (sum, lesson) =>
      sum + Math.max(0, lesson.capacity - lesson.childIds.length),
    0,
  ),
);

function bookingFor(lesson: Lesson): ParentBooking | undefined {
  const child = selectedChild.value;
  if (!child) return undefined;
  return state.bookings.find(
    (booking) =>
      booking.lessonId === lesson.id &&
      booking.childId === child.id &&
      (booking.status === "pending" || booking.status === "confirmed"),
  );
}

function isPast(lesson: Lesson): boolean {
  if (lesson.date < DEMO_TODAY) return true;
  if (lesson.date > DEMO_TODAY) return false;
  return timeToMinutes(lesson.time) <= DEMO_NOW_MINUTES;
}

function canBook(lesson: Lesson): boolean {
  if (isPast(lesson)) return false;
  if (!selectedChild.value) return false;
  if (bookingFor(lesson)) return false;
  if (lesson.childIds.includes(selectedChild.value.id)) return false;
  return lesson.childIds.length < lesson.capacity;
}

function seatsLabel(lesson: Lesson): string {
  if (isPast(lesson)) return "Занятие прошло";
  if (bookingFor(lesson)) return "Вы записаны на это занятие";
  if (lesson.childIds.includes(selectedChild.value?.id ?? ""))
    return "Ребёнок уже записан";
  const free = lesson.capacity - lesson.childIds.length;
  if (free <= 0) return "Свободных мест нет";
  return `Свободно ${free} ${plural(free, "место", "места", "мест")}`;
}

function chipLabel(lesson: Lesson): string {
  if (isPast(lesson)) return "Прошло";
  if (bookingFor(lesson)?.status === "confirmed") return "Записано";
  if (bookingFor(lesson)) return "Ждёт мастера";
  if (lesson.childIds.length >= lesson.capacity) return "Мест нет";
  return lesson.kind === "group" ? "Группа" : "Один на один";
}

function chipClass(lesson: Lesson): string {
  if (isPast(lesson)) return "bg-paper text-muted";
  if (bookingFor(lesson)?.status === "confirmed") {
    return "bg-forest-soft text-forest";
  }
  if (bookingFor(lesson)) return "bg-honey-soft text-honey";
  if (lesson.childIds.length >= lesson.capacity)
    return "bg-clay-soft text-clay";
  return "bg-paper text-muted";
}

const bookingOpen = ref(false);
const bookingLesson = ref<Lesson | null>(null);

function openBooking(lesson: Lesson) {
  bookingLesson.value = lesson;
  bookingOpen.value = true;
}

function onBooked() {

}

const selectedSubscription = computed(() =>
  selectedChild.value ? subscriptionFor(selectedChild.value.id) : undefined,
);
const coverageLabel = computed(() => {
  const sub = selectedSubscription.value;
  if (!sub) return "нет оплаченных занятий";
  if (sub.remaining > 0 && sub.validUntil >= date.value) {
    return `${sub.remaining} ${plural(sub.remaining, "занятие", "занятия", "занятий")} по абонементу`;
  }
  if ((sub.singleCredits ?? 0) > 0) {
    return `${sub.singleCredits} ${plural(sub.singleCredits, "разовое посещение", "разовых посещения", "разовых посещений")}`;
  }
  return "нет оплаченных занятий";
});
</script>
