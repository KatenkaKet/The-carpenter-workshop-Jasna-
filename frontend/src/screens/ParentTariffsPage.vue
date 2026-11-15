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
    active="tariffs"
    @logout="logout"
  >
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Оплата занятий</p>
        <h1 class="page-title mt-1.5">Тарифы</h1>
        <p class="mt-1.5 text-sm text-muted">
          Купите одно посещение или абонемент — потом запишитесь на свободный
          день.
        </p>
      </div>
      <a href="/yasna/parent" class="btn-ghost text-xs">
        <i class="fa-solid fa-calendar-days" aria-hidden="true"></i>
        В расписание
      </a>
    </div>

    <div v-if="children.length > 1" class="mt-4">
      <p class="label">Для кого</p>
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
          {{ child.name }}, {{ child.age }}
        </button>
      </div>
    </div>

    <div
      class="card mt-3 flex flex-wrap items-center justify-between gap-3 p-4"
    >
      <div>
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Доступно для новых записей
        </p>
        <p class="mt-1 text-sm font-bold">
          <template v-if="coveredRemaining > 0">
            {{ coveredRemaining }}
            {{ plural(coveredRemaining, "занятие", "занятия", "занятий") }} по
            абонементу
          </template>
          <template v-else>Абонемента нет</template>
          <template v-if="singleCredits > 0">
            <span class="text-honey"> + {{ singleCredits }} разовое</span>
          </template>
        </p>
        <p
          v-if="subscription && coveredRemaining > 0"
          class="mt-0.5 text-[10px] text-muted"
        >
          Действует до {{ formatDay(subscription.validUntil) }}
        </p>
        <p v-else-if="singleCredits > 0" class="mt-0.5 text-[10px] text-muted">
          Разовое посещение можно использовать на любой свободный день
        </p>
      </div>
      <span
        class="chip"
        :class="
          coveredRemaining > 0 || singleCredits > 0
            ? 'bg-forest-soft text-forest'
            : 'bg-honey-soft text-honey'
        "
      >
        {{
          coveredRemaining > 0 || singleCredits > 0
            ? "можно записываться"
            : "нужна оплата"
        }}
      </span>
    </div>

    <section id="tariffs" class="mt-5" aria-labelledby="tariffs-title">
      <h2 id="tariffs-title" class="section-title">Тарифы мастерской</h2>
      <div class="mt-3 grid gap-3 sm:grid-cols-3">
        <article
          v-for="tariff in tariffs"
          :key="tariff.id"
          class="card flex flex-col p-4"
          :class="props.tariff === tariff.id ? 'ring-2 ring-honey/60' : ''"
        >
          <div class="flex items-start justify-between gap-2">
            <span class="chip bg-paper text-muted">
              {{ tariff.lessons === 1 ? "разово" : "абонемент" }}
            </span>
            <span
              v-if="tariff.lessons === 8"
              class="chip bg-honey-soft text-honey"
              >выгодно</span
            >
          </div>
          <h3 class="mt-2 font-display text-lg font-semibold leading-tight">
            {{ tariff.title
            }}
          </h3>
          <p class="mt-1 text-[11px] leading-relaxed text-muted">
            {{ tariff.lessons }}
            {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }} ·
            действует {{ tariff.days }} дней
          </p>
          <p class="mt-3 font-display text-2xl font-semibold">
            {{ formatMoney(tariff.price) }}
          </p>
          <a
            :href="checkoutHref(tariff)"
            class="btn-primary mt-3 w-full text-xs"
          >
            {{ tariff.lessons === 1 ? "Купить посещение" : "Купить абонемент" }}
          </a>
        </article>
      </div>
    </section>

    <section v-if="dateValue" class="mt-6" aria-labelledby="day-lessons-title">
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 id="day-lessons-title" class="section-title">
            Занятия на {{ formatDay(dateValue) }}
          </h2>
          <p class="mt-0.5 text-[11px] text-muted">
            Абонемент начисляется после подтверждения оплаты сервисом.
          </p>
        </div>
        <a
          :href="`/yasna/parent-day?date=${dateValue}`"
          class="text-[11px] font-bold text-honey"
        >
          Открыть день
          <i
            class="fa-solid fa-chevron-right text-[8px]"
            aria-hidden="true"
          ></i>
        </a>
      </div>

      <div v-if="dayLessons.length" class="mt-3 space-y-2.5">
        <article
          v-for="lesson in dayLessons"
          :key="lesson.id"
          class="card flex items-center gap-3 p-3.5"
        >
          <div class="w-12 shrink-0 text-center">
            <p class="text-sm font-bold">{{ lesson.time }}</p>
            <p class="text-[9px] text-muted">
              {{ lesson.kind === "group" ? "группа" : "индив." }}
            </p>
          </div>
          <div class="min-w-0 flex-1">
            <p class="truncate text-xs font-bold">{{ lesson.title }} <span v-if="lesson.isMakeup" class="chip bg-honey-soft">Отработка</span></p>
            <p class="mt-0.5 text-[10px] text-muted">
              {{ lesson.childIds.length }} из {{ lesson.capacity }} ·
              {{
                lesson.childIds.length >= lesson.capacity
                  ? "мест нет"
                  : "есть места"
              }}
            </p>
          </div>
          <span
            v-if="bookingFor(lesson)"
            class="chip shrink-0 bg-honey-soft text-honey"
          >
            {{
              bookingFor(lesson)?.status === "confirmed"
                ? "записан"
                : "ждёт мастера"
            }}
          </span>
          <button
            v-else-if="canBook(lesson)"
            type="button"
            class="btn-primary shrink-0 !min-h-[34px] !px-2.5 text-[10px]"
            @click="book(lesson)"
          >
            Записаться
          </button>
          <a
            v-else
            href="#tariffs"
            class="btn-quiet shrink-0 !min-h-[34px] border border-line !px-2.5 text-[10px]"
          >
            Купить тариф
          </a>
        </article>
      </div>
      <div v-else class="card mt-3 px-5 py-8 text-center text-xs text-muted">
        На этот день занятий нет
      </div>
    </section>

    <div class="mt-6 rounded-2xl border border-line/80 bg-card p-4">
      <p class="text-xs font-bold">Как это работает</p>
      <p class="mt-1 text-[11px] leading-relaxed text-muted">
        Тариф можно купить сразу после входа. Затем выберите день в расписании:
        запись создаётся сразу при наличии мест. Абонемент можно оплатить здесь;
        запись сама по себе не означает оплату.
      </p>
      <a href="/yasna/parent" class="btn-primary mt-3 text-xs"
        >Выбрать занятие</a
      >
    </div>

    <AppToast :message="toastMessage" />
  <SubscriptionBalances/>
</ParentShell>
</template>

<script setup lang="ts">
import SubscriptionBalances from "../components/SubscriptionBalances.vue";
import { computed, ref, watch } from "vue";
import AppToast from "../components/AppToast.vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  DEMO_NOW_MINUTES,
  DEMO_TODAY,
  formatDay,
  formatMoney,
  plural,
  timeToMinutes,
  type Child,
  type Lesson,
  type ParentBooking,
  type Tariff,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";
import { useToast } from "../shared/use-toast";

const props = withDefaults(
  defineProps<{ date?: string; child?: string; tariff?: string }>(),
  { date: "", child: "", tariff: "" },
);

const { state, lessonsFor, createBooking, subscriptionFor, bookingsForFamily } =
  useDemo();
const { family, children, ready, error, loginAs, logout } = useParent();
const { message: toastMessage, show: notify } = useToast();

const tariffs = computed(() => state.settings.tariffs);
const selectedChildId = ref("");
const dateValue = computed(() =>
  /^\d{4}-\d{2}-\d{2}$/.test(props.date) ? props.date : "",
);

watch(
  children,
  (list) => {
    if (!list.some((child) => child.id === selectedChildId.value)) {
      const preferred = list.find((child) => child.id === props.child);
      selectedChildId.value = preferred?.id ?? list[0]?.id ?? "";
    }
  },
  { immediate: true },
);

const selectedChild = computed(
  () =>
    children.value.find((child) => child.id === selectedChildId.value) ??
    children.value[0],
);

const subscription = computed(() =>
  selectedChild.value ? subscriptionFor(selectedChild.value.id) : undefined,
);
const coveredRemaining = computed(() => {
  const sub = subscription.value;
  if (!sub) return 0;
  if (sub.validUntil < (dateValue.value || DEMO_TODAY)) return 0;
  return sub.remaining;
});
const singleCredits = computed(() => subscription.value?.singleCredits ?? 0);

const dayLessons = computed(() =>
  dateValue.value ? lessonsFor(dateValue.value) : [],
);

function isPast(lesson: Lesson): boolean {
  if (lesson.date < DEMO_TODAY) return true;
  if (lesson.date > DEMO_TODAY) return false;
  return timeToMinutes(lesson.time) <= DEMO_NOW_MINUTES;
}

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

function canBook(lesson: Lesson): boolean {
  if (isPast(lesson) || !selectedChild.value) return false;
  if (bookingFor(lesson)) return false;
  if (lesson.childIds.includes(selectedChild.value.id)) return false;
  if (lesson.childIds.length >= lesson.capacity) return false;
  return true;
}

async function book(lesson: Lesson) {
  if (!family.value || !selectedChild.value) return;
  const result = await createBooking(
    family.value.id,
    selectedChild.value.id,
    lesson.id,
  );
  if (!result.ok) {
    notify(result.message);
    return;
  }

}

function checkoutHref(tariff: Tariff): string {
  const params = new URLSearchParams({ tariff: tariff.id });
  if (selectedChild.value) params.set("child", selectedChild.value.id);
  if (dateValue.value) params.set("date", dateValue.value);
  return `/yasna/parent-checkout?${params.toString()}`;
}
</script>
