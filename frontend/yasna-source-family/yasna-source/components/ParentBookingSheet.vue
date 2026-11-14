<template>
  <BottomSheet
    :open="open"
    :title="covered ? 'Записаться на занятие' : 'Запись и оплата'"
    :subtitle="lesson ? `${lesson.time} · ${lesson.title}` : ''"
    @close="$emit('close')"
  >
    <div v-if="lesson && child" class="space-y-4">
      <div
        class="flex items-start gap-3 rounded-2xl border border-line/80 bg-paper/60 p-3.5"
      >
        <span
          class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-[10px] font-bold"
          :class="avatarTone(child.id)"
        >
          {{ initials(child.name) }}
        </span>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-bold">{{ child.name }}, {{ child.age }}</p>
          <p class="mt-0.5 text-[11px] text-muted">
            {{ formatDay(lesson.date) }} · {{ lesson.time }} ·
            {{ lesson.kind === "group" ? "групповое" : "индивидуальное" }}
          </p>
        </div>
        <span
          class="chip shrink-0"
          :class="
            covered ? 'bg-forest-soft text-forest' : 'bg-honey-soft text-honey'
          "
        >
          {{ covered ? "оплачено" : "нужна оплата" }}
        </span>
      </div>

      <div v-if="!covered">
        <span class="label">Тариф</span>
        <div class="space-y-1.5">
          <button
            v-for="tariff in tariffs"
            :key="tariff.id"
            type="button"
            class="w-full rounded-xl border p-3 text-left transition"
            :class="
              tariffId === tariff.id
                ? 'border-forest bg-forest-soft/50'
                : 'border-line bg-card hover:border-forest/30'
            "
            @click="tariffId = tariff.id"
          >
            <span class="flex items-center justify-between gap-3">
              <span>
                <span class="block text-sm font-bold">{{ tariff.title }}</span>
                <span class="mt-0.5 block text-[11px] text-muted">
                  {{ tariff.lessons }}
                  {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }}
                  · {{ tariff.days }} дней
                </span>
              </span>
              <span class="shrink-0 text-sm font-bold">{{
                formatMoney(tariff.price)
              }}</span>
            </span>
          </button>
        </div>
      </div>

      <div
        v-else
        class="rounded-xl bg-forest-soft/50 px-3 py-2.5 text-[11px] leading-relaxed text-forest"
      >
        У ребёнка есть оплаченный тариф: {{ coverageText }}. Запишем сразу, без
        ожидания.
      </div>

      <p
        v-if="error"
        class="rounded-xl bg-clay-soft px-3 py-2 text-xs font-semibold text-clay"
        role="alert"
      >
        {{ error }}
      </p>

      <button
        type="button"
        class="btn-primary w-full"
        :disabled="busy"
        @click="submit"
      >
        <i
          :class="busy ? 'fa-solid fa-spinner fa-spin' : 'fa-solid fa-check'"
          class="text-[11px]"
          aria-hidden="true"
        ></i>
        {{ busy ? "Оформляем…" : submitLabel }}
      </button>
      <p class="text-center text-[10px] leading-relaxed text-muted">
        Запись создаётся сразу и появляется у мастера. Подтверждение не
        требуется.
        <template v-if="!covered">
          Дальше откроется страница оплаты. Чек появится в «Истории оплат» после
          загрузки мастером.
        </template>
      </p>
    </div>
  </BottomSheet>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import BottomSheet from "./BottomSheet.vue";
import {
  avatarTone,
  formatDay,
  formatMoney,
  initials,
  plural,
  type Lesson,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";

const props = defineProps<{
  open: boolean;
  lesson: Lesson | null;
  childId: string;
  familyId: string;
}>();

const emit = defineEmits<{ close: []; booked: [] }>();

const { state, childById, subscriptionFor, createBooking } = useDemo();

const tariffId = ref("t1");
const busy = ref(false);
const error = ref("");

const tariffs = computed(() => state.settings.tariffs);
const child = computed(() =>
  props.childId ? childById(props.childId) : undefined,
);
const subscription = computed(() =>
  props.childId ? subscriptionFor(props.childId) : undefined,
);

const covered = computed(() => {
  const sub = subscription.value;
  if (!sub || !props.lesson) return false;
  if ((sub.singleCredits ?? 0) > 0) return true;
  return sub.remaining > 0 && sub.validUntil >= props.lesson.date;
});

const coverageText = computed(() => {
  const sub = subscription.value;
  if (!sub) return "";
  if (
    sub.remaining > 0 &&
    props.lesson &&
    sub.validUntil >= props.lesson.date
  ) {
    return `${sub.remaining} ${plural(sub.remaining, "занятие", "занятия", "занятий")} по абонементу`;
  }
  if ((sub.singleCredits ?? 0) > 0) {
    return `${sub.singleCredits} ${plural(sub.singleCredits, "разовое посещение", "разовых посещения", "разовых посещений")}`;
  }
  return "оплата не найдена";
});

const selectedTariff = computed(() =>
  tariffs.value.find((tariff) => tariff.id === tariffId.value),
);

const submitLabel = computed(() =>
  covered.value ? "Записаться" : "Перейти к оплате",
);

const checkoutHref = computed(() => {
  if (!props.lesson || !selectedTariff.value) return "";
  const params = new URLSearchParams({
    tariff: selectedTariff.value.id,
    child: props.childId,
    lesson: props.lesson.id,
    date: props.lesson.date,
  });
  return `/yasna/parent-checkout?${params.toString()}`;
});

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    tariffId.value = "t1";
    busy.value = false;
    error.value = "";
  },
);

function submit() {
  if (!props.lesson || !props.childId || !props.familyId) return;
  error.value = "";

  if (!covered.value) {
    if (!selectedTariff.value) {
      error.value = "Выберите тариф.";
      return;
    }
    if (typeof window !== "undefined") {
      window.location.href = checkoutHref.value;
    }
    return;
  }

  busy.value = true;
  const booked = createBooking(props.familyId, props.childId, props.lesson.id);
  busy.value = false;
  if (!booked.ok) {
    error.value = booked.message;
    return;
  }
  emit("booked");
  emit("close");
}
</script>
