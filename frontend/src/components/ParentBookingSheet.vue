<template>
  <BottomSheet
    :open="open"
    title="Записаться на занятие"
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
          Запись на занятие
        </span>
      </div>

<label class="label">Способ учёта<select class="field" v-model="funding"><option value="subscription">Абонемент — доступно {{available}}</option><option value="single_lesson">Разовое занятие, отдельная оплата</option></select></label>
      <p class="text-sm text-muted">
        При выборе абонемента одно занятие резервируется. Разовое занятие оплачивается отдельно по правилам группы.
      </p>
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
        {{ busy ? "Оформляем…" : "Записаться" }}
      </button>
      <p class="text-center text-[10px] leading-relaxed text-muted">
        Запись создаётся сразу и появляется у мастера. Подтверждение не
        требуется.
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
import {serverSubscriptions} from "../shared/use-demo";
import { useDemo } from "../shared/use-demo";

const props = defineProps<{
  open: boolean;
  lesson: Lesson | null;
  childId: string;
  familyId: string;
}>();

const emit = defineEmits<{ close: []; booked: [] }>();

const { state, childById, subscriptionFor, createBooking } = useDemo();

const funding=ref("subscription");
const available=computed(()=>serverSubscriptions.value.filter(s=>s.child_id===props.childId&&s.group_id===props.lesson?.groupId&&s.valid_from.slice(0,10)<=(props.lesson?.date||'')&&s.valid_until.slice(0,10)>=(props.lesson?.date||'')&&s.status!=="suspended").reduce((n,s)=>n+s.lessons_available,0));
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
  return sub.remaining > 0 && sub.validUntil >= (props.lesson?.date||'');
});

const coverageText = computed(() => {
  const sub = subscription.value;
  if (!sub) return "";
  if (
    sub.remaining > 0 &&
    props.lesson &&
    sub.validUntil >= (props.lesson?.date||'')
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
    date: (props.lesson?.date||''),
  });
  return `/yasna/parent-checkout?${params.toString()}`;
});

watch(
  () => props.open,
  (open) => {
    if (!open) return;
    tariffId.value = state.settings.tariffs[0]?.id || "";
    busy.value = false;
    error.value = "";
  },
);

async function submit() {
  if (!props.lesson || !props.childId || !props.familyId) return;
  error.value = "";

  busy.value = true;
  const booked = await createBooking(
    props.familyId,
    props.childId,
    props.lesson.id,
    funding.value,
  );
  busy.value = false;
  if (!booked.ok) {
    error.value = booked.message;
    return;
  }
  emit("booked");
  emit("close");
}
</script>
