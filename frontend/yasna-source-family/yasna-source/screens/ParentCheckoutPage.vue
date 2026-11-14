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
    <a :href="backHref" class="btn-quiet -ml-2 !min-h-0 !px-2 text-xs">
      <i class="fa-solid fa-chevron-left text-[10px]" aria-hidden="true"></i>
      Назад
    </a>

    <div v-if="success" class="mx-auto mt-5 w-full max-w-lg">
      <div class="card px-5 py-8 text-center sm:px-8">
        <span
          class="mx-auto grid h-16 w-16 place-items-center rounded-full bg-forest-soft text-2xl text-forest"
        >
          <i class="fa-solid fa-check" aria-hidden="true"></i>
        </span>
        <p class="eyebrow mt-4">Оплата прошла</p>
        <h1 class="page-title mt-1.5">
          {{ booked ? "Вы записаны" : "Тариф оплачен" }}
        </h1>
        <p class="mt-2 text-sm text-muted">
          {{ child?.name }} · {{ tariff?.title }} ·
          {{ formatMoney(tariff?.price ?? 0) }}
        </p>
        <p v-if="lesson" class="mt-1 text-[11px] text-muted">
          {{ formatDay(lesson.date) }}, {{ lesson.time }} · {{ lesson.title }}
        </p>

        <div
          v-if="warning"
          class="mt-4 rounded-xl bg-honey-soft/60 px-3 py-2.5 text-[11px] leading-relaxed text-ink"
        >
          {{ warning }}
        </div>

        <p v-else class="mt-4 text-[11px] leading-relaxed text-muted">
          <template v-if="booked">
            Запись уже активна и видна мастеру. Чек появится в «Истории оплат»
            после загрузки мастером.
          </template>
          <template v-else>
            Тариф добавлен в кабинет. Теперь можно выбрать день и записаться.
            Чек появится в «Истории оплат» после загрузки мастером.
          </template>
        </p>

        <a href="/yasna/parent" class="btn-primary mt-5 w-full">К расписанию</a>
        <a href="/yasna/parent-payments" class="btn-ghost mt-2 w-full"
          >История оплат</a
        >
      </div>
    </div>

    <div v-else class="mt-4 grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="card p-5 sm:p-6" aria-labelledby="checkout-title">
        <p class="eyebrow">Оплата</p>
        <h1 id="checkout-title" class="page-title mt-1.5">Оплата тарифа</h1>
        <p class="mt-1.5 text-sm text-muted">
          Проверьте состав и нажмите «Оплатить». Это демонстрационный платёж.
        </p>

        <div
          v-if="child && tariff"
          class="mt-5 rounded-2xl border border-line/80 bg-paper/60 p-4"
        >
          <div class="flex items-start gap-3">
            <span
              class="grid h-10 w-10 shrink-0 place-items-center rounded-full text-[11px] font-bold"
              :class="avatarTone(child.id)"
            >
              {{ initials(child.name) }}
            </span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-bold">{{ child.name }}, {{ child.age }}</p>
              <p class="mt-0.5 text-[11px] text-muted">{{ tariff.title }}</p>
            </div>
          </div>
          <p class="mt-3 text-[11px] leading-relaxed text-muted">
            {{ tariff.lessons }}
            {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }} ·
            действует {{ tariff.days }} дней
          </p>
          <div
            v-if="lesson"
            class="mt-3 rounded-xl bg-card px-3 py-2.5 text-[11px]"
          >
            <b>Запись:</b> {{ formatDay(lesson.date) }}, {{ lesson.time }} ·
            {{ lesson.title }}
          </div>
        </div>

        <div class="mt-4 rounded-2xl border border-line/80 bg-card p-4">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
            Способ оплаты
          </p>
          <div
            class="mt-2 flex items-center gap-3 rounded-xl border border-forest bg-forest-soft/40 p-3"
          >
            <span
              class="grid h-9 w-9 place-items-center rounded-full bg-card text-forest"
            >
              <i class="fa-solid fa-bolt" aria-hidden="true"></i>
            </span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-bold">СБП или карта</p>
              <p class="text-[10px] text-muted">
                Демонстрационный платёж, деньги не списываются
              </p>
            </div>
            <i
              class="fa-solid fa-circle-check text-forest"
              aria-hidden="true"
            ></i>
          </div>
        </div>
      </section>

      <aside class="space-y-3">
        <div class="card p-5">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
            Итого
          </p>
          <p class="mt-1 font-display text-3xl font-semibold">
            {{ formatMoney(tariff?.price ?? 0) }}
          </p>
          <p v-if="tariff" class="mt-1 text-[11px] text-muted">
            {{ tariff.lessons }}
            {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }} ·
            {{ tariff.days }} дней
          </p>

          <p
            v-if="blocked"
            class="mt-4 rounded-xl bg-clay-soft px-3 py-2.5 text-xs font-semibold leading-relaxed text-clay"
            role="alert"
          >
            {{ blocked }}
          </p>
          <p
            v-else-if="payError"
            class="mt-4 rounded-xl bg-clay-soft px-3 py-2.5 text-xs font-semibold leading-relaxed text-clay"
            role="alert"
          >
            {{ payError }}
          </p>

          <button
            type="button"
            class="btn-primary mt-4 w-full"
            :disabled="!canPay || paying"
            @click="pay"
          >
            <i
              :class="
                paying ? 'fa-solid fa-spinner fa-spin' : 'fa-solid fa-lock'
              "
              class="text-[11px]"
              aria-hidden="true"
            ></i>
            {{
              paying
                ? "Проводим оплату…"
                : `Оплатить ${formatMoney(tariff?.price ?? 0)}`
            }}
          </button>
          <p class="mt-2 text-center text-[10px] leading-relaxed text-muted">
            <template v-if="lesson">
              Нажимая «Оплатить», вы подтверждаете демо-платёж. Запись создастся
              сразу, без подтверждения мастером.
            </template>
            <template v-else>
              Нажимая «Оплатить», вы подтверждаете демо-платёж. Тариф появится в
              кабинете — после этого можно записаться на занятие.
            </template>
          </p>
        </div>

        <p
          class="rounded-2xl bg-paper/70 px-4 py-3 text-[10px] leading-relaxed text-muted"
        >
          Чек формирует мастер в «Мой налог» и прикрепляет к оплате. После этого
          он появится в «Истории оплат» и его можно будет скачать.
        </p>
      </aside>
    </div>

    <AppToast :message="toastMessage" />
  </ParentShell>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import AppToast from "../components/AppToast.vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  DEMO_NOW_MINUTES,
  DEMO_TODAY,
  avatarTone,
  formatDay,
  formatMoney,
  initials,
  plural,
  timeToMinutes,
  type Lesson,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";
import { useToast } from "../shared/use-toast";

const props = withDefaults(
  defineProps<{
    tariff?: string;
    child?: string;
    lesson?: string;
    date?: string;
  }>(),
  { tariff: "", child: "", lesson: "", date: "" },
);

const { state, childById, lessonById, payTariff, createBooking } = useDemo();
const { family, ready, error, loginAs, logout } = useParent();
const { message: toastMessage, show: notify } = useToast();

const tariff = computed(() =>
  state.settings.tariffs.find((item) => item.id === props.tariff),
);
const child = computed(() =>
  props.child
    ? childById(props.child)
    : family.value?.childIds[0]
      ? childById(family.value.childIds[0]!)
      : undefined,
);
const lesson = computed(() =>
  props.lesson ? lessonById(props.lesson) : undefined,
);

const backHref = computed(() => {
  if (lesson.value) return `/yasna/parent-day?date=${lesson.value.date}`;
  return "/yasna/parent-tariffs";
});

const paying = ref(false);
const success = ref(false);
const booked = ref(false);
const warning = ref("");
const payError = ref("");

function isPast(target: Lesson): boolean {
  if (target.date < DEMO_TODAY) return true;
  if (target.date > DEMO_TODAY) return false;
  return timeToMinutes(target.time) <= DEMO_NOW_MINUTES;
}

const blocked = computed(() => {
  if (!child.value) return "Не выбран ребёнок.";
  if (!tariff.value) return "Тариф не найден.";
  if (child.value.familyId !== family.value?.id)
    return "Ребёнок из другой семьи.";
  if (!lesson.value) return "";
  if (isPast(lesson.value)) return "Это занятие уже прошло.";
  if (lesson.value.childIds.includes(child.value.id)) {
    return "Ребёнок уже записан на это занятие.";
  }
  if (lesson.value.childIds.length >= lesson.value.capacity) {
    return "Свободных мест на занятии нет.";
  }
  const active = state.bookings.find(
    (booking) =>
      booking.lessonId === lesson.value?.id &&
      booking.childId === child.value?.id &&
      (booking.status === "pending" || booking.status === "confirmed"),
  );
  if (active) return "Ребёнок уже записан на это занятие.";
  return "";
});

const canPay = computed(() =>
  Boolean(tariff.value && child.value && !blocked.value),
);

function pay() {
  if (
    !tariff.value ||
    !child.value ||
    !family.value ||
    !canPay.value ||
    paying.value
  ) {
    return;
  }
  paying.value = true;
  payError.value = "";
  warning.value = "";

  const paid = payTariff(child.value.id, tariff.value.id, {
    silentParent: Boolean(lesson.value),
  });
  if (!paid.ok) {
    payError.value = paid.message;
    paying.value = false;
    return;
  }

  if (lesson.value) {
    const bookedResult = createBooking(
      family.value.id,
      child.value.id,
      lesson.value.id,
    );
    if (bookedResult.ok) {
      booked.value = true;
    } else {
      warning.value =
        "Оплата прошла, но записаться на это занятие не удалось. Тариф остался в кабинете — выберите другой день.";
    }
  }

  paying.value = false;
  success.value = true;
  notify(
    booked.value
      ? "Оплата прошла, вы записаны"
      : "Оплата прошла. Тариф добавлен в кабинет",
  );
}
</script>
