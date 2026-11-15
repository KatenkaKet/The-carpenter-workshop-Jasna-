<template>
  <AppShell active="schedule">
    <a href="/yasna" class="btn-quiet -ml-2 !min-h-0 !px-2 text-xs">
      <i class="fa-solid fa-chevron-left text-[10px]" aria-hidden="true"></i>
      Расписание
    </a>

    <div class="mt-3 flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">{{ formatWeekday(date) }}</p>
        <h1 class="page-title mt-1.5">{{ formatDay(date) }}</h1>
        <p class="mt-1.5 text-sm text-muted">
          {{ dayLessons.length }}
          {{ plural(dayLessons.length, "занятие", "занятия", "занятий") }} ·
          {{ totalEnrolled }}
          {{ plural(totalEnrolled, "запись", "записи", "записей") }}
        </p>
      </div>
      <a
        v-if="dayLessons.length"
        :href="`/yasna/journal?date=${date}`"
        class="btn-ghost text-xs"
      >
        <i class="fa-solid fa-clipboard-check" aria-hidden="true"></i>
        Журнал дня
      </a>
    </div>

    <div class="mt-5 grid gap-3 sm:grid-cols-2">
      <section
        class="relative min-w-0 overflow-hidden rounded-[22px] border border-honey/35 bg-gradient-to-br from-honey-soft to-card p-4 shadow-soft sm:p-5"
        aria-labelledby="group-title"
      >
        <div class="flex items-start justify-between">
          <span
            class="grid h-11 w-11 place-items-center rounded-2xl bg-card text-[18px] text-honey shadow-sm"
          >
            <i class="fa-solid fa-user-group" aria-hidden="true"></i>
          </span>
          <span class="chip bg-card/80 text-honey">Группа</span>
        </div>
        <h2 id="group-title" class="section-title mt-3">Групповое занятие</h2>
        <p class="mt-1 text-[13px] leading-snug text-muted">
          Несколько детей за одним верстаком. Выберите число мест.
        </p>

        <div
          class="mt-3 flex items-center justify-between rounded-xl border border-line/80 bg-card/90 p-1.5 pl-3"
        >
          <span class="text-[11px] font-bold uppercase tracking-wide text-muted"
            >Мест в группе</span
          >
          <div class="flex items-center gap-1">
            <button
              type="button"
              class="grid h-8 w-8 place-items-center rounded-lg border border-line bg-paper text-ink disabled:opacity-30"
              :disabled="groupSeats <= 2"
              aria-label="Меньше мест"
              @click="groupSeats = Math.max(2, groupSeats - 1)"
            >
              <i class="fa-solid fa-minus text-[10px]" aria-hidden="true"></i>
            </button>
            <span class="w-7 text-center font-display text-lg font-semibold">{{
              groupSeats
            }}</span>
            <button
              type="button"
              class="grid h-8 w-8 place-items-center rounded-lg border border-line bg-paper text-ink disabled:opacity-30"
              :disabled="groupSeats >= 8"
              aria-label="Больше мест"
              @click="groupSeats = Math.min(8, groupSeats + 1)"
            >
              <i class="fa-solid fa-plus text-[10px]" aria-hidden="true"></i>
            </button>
          </div>
        </div>

        <button
          type="button"
          class="btn-primary mt-3 w-full"
          @click="openCreate('group')"
        >
          Создать групповое
          <i class="fa-solid fa-arrow-right text-[10px]" aria-hidden="true"></i>
        </button>
      </section>

      <section
        class="relative min-w-0 overflow-hidden rounded-[22px] border border-forest/20 bg-gradient-to-br from-forest-soft to-card p-4 shadow-soft sm:p-5"
        aria-labelledby="individual-title"
      >
        <div class="flex items-start justify-between">
          <span
            class="grid h-11 w-11 place-items-center rounded-2xl bg-card text-[18px] text-forest shadow-sm"
          >
            <i class="fa-solid fa-user" aria-hidden="true"></i>
          </span>
          <span class="chip bg-card/80 text-forest">Один на один</span>
        </div>
        <h2 id="individual-title" class="section-title mt-3">Индивидуальное</h2>
        <p class="mt-1 text-[13px] leading-snug text-muted">
          Один ребёнок и мастер: свой темп, своя задача.
        </p>

        <div
          class="mt-3 flex items-center gap-2 rounded-xl border border-line/80 bg-card/90 p-1.5 pl-3"
        >
          <span class="text-[11px] font-bold uppercase tracking-wide text-muted"
            >Формат</span
          >
          <span class="ml-auto chip bg-paper text-forest">
            <i class="fa-solid fa-clock text-[9px]" aria-hidden="true"></i>
            90 минут
          </span>
        </div>

        <button
          type="button"
          class="btn-primary mt-3 w-full"
          @click="openCreate('individual')"
        >
          Создать индивидуальное
          <i class="fa-solid fa-arrow-right text-[10px]" aria-hidden="true"></i>
        </button>
      </section>
    </div>

    <section class="mt-6" aria-labelledby="day-title">
      <div class="flex items-baseline justify-between gap-3">
        <h2 id="day-title" class="section-title">Занятия дня</h2>
        <span class="text-[11px] font-semibold text-muted">
          {{ freeSeats }}
          {{
            plural(
              freeSeats,
              "свободное место",
              "свободных места",
              "свободных мест",
            )
          }}
        </span>
      </div>

      <div v-if="dayLessons.length" class="mt-3 space-y-3">
        <article v-for="lesson in dayLessons" :key="lesson.id" class="card p-4">
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
                  <h3 class="text-sm font-bold leading-snug">
                    {{ lesson.title }} <span v-if="lesson.isMakeup" class="chip bg-honey-soft">Отработка</span>
                  </h3>
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
                    {{
                      lesson.kind === "group" ? "Групповое" : "Индивидуальное"
                    }}
                    · {{ lesson.childIds.length }} из {{ lesson.capacity }}
                  </p>
                </div>
                <span
                  class="chip shrink-0"
                  :class="statusMeta[statusOf(lesson)].className"
                >
                  {{ statusMeta[statusOf(lesson)].label }}
                </span>
              </div>

              <div
                v-if="lesson.childIds.length"
                class="mt-2 flex flex-wrap items-center gap-1.5"
              >
                <span
                  v-for="child in enrolledPreview(lesson)"
                  :key="child.id"
                  class="grid h-6 w-6 place-items-center rounded-full text-[9px] font-bold ring-2 ring-card"
                  :class="avatarTone(child.id)"
                  :title="child.name"
                >
                  {{ initials(child.name) }}
                </span>
                <span
                  v-if="lesson.childIds.length > 5"
                  class="text-[10px] font-bold text-muted"
                >
                  +{{ lesson.childIds.length - 5 }}
                </span>
                <span class="ml-1 text-[10px] font-semibold text-muted">
                  {{
                    lesson.childIds.length === lesson.capacity
                      ? lesson.kind === "group"
                        ? "группа заполнена"
                        : "ребёнок записан"
                      : `свободно ${lesson.capacity - lesson.childIds.length}`
                  }}
                </span>
              </div>

              <div class="mt-3 flex flex-wrap gap-2">
                <a
                  :href="`/yasna/journal?date=${date}&lesson=${lesson.id}`"
                  class="btn-ghost !min-h-[38px] flex-1 text-xs sm:flex-none"
                >
                  <i
                    class="fa-solid fa-clipboard-check text-[11px]"
                    aria-hidden="true"
                  ></i>
                  Журнал
                </a>
                <button
                  v-if="lesson.childIds.length < lesson.capacity"
                  type="button"
                  class="btn-quiet !min-h-[38px] border border-line text-xs sm:flex-none"
                  @click="openEnroll(lesson)"
                >
                  <i
                    class="fa-solid fa-plus text-[10px]"
                    aria-hidden="true"
                  ></i>
                  Записать
                </button>
                <span v-else class="chip self-center bg-clay-soft text-clay">
                  <i class="fa-solid fa-lock text-[9px]" aria-hidden="true"></i>
                  мест нет
                </span>
              </div>
            </div>
          </div>
        </article>
      </div>

      <div v-else class="mt-3 card px-5 py-10 text-center">
        <span
          class="mx-auto grid h-14 w-14 place-items-center rounded-full bg-paper text-xl text-muted"
        >
          <i class="fa-solid fa-calendar-plus" aria-hidden="true"></i>
        </span>
        <p class="mt-3 font-display text-lg font-semibold">
          День пока свободен
        </p>
        <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
          Создайте групповое или индивидуальное занятие — оно сразу появится в
          календаре.
        </p>
      </div>
    </section>

    <BottomSheet
      :open="createOpen"
      :title="
        mode === 'group' ? 'Новое групповое занятие' : 'Индивидуальное занятие'
      "
      :subtitle="`${formatDay(date)}, ${formatWeekday(date)}`"
      @close="createOpen = false"
    >
      <div class="space-y-4">
        <label class="label">Время начала<input class="field" type="time" step="60" v-model="form.time" required></label>
        <label class="label">Длительность, минут<input class="field" type="number" min="1" max="720" step="1" v-model.number="form.duration" required></label>
        <p class="text-xs text-muted">Окончание: {{endTime}}</p>
        <label v-if="mode==='individual'" class="flex gap-2"><input type="checkbox" v-model="form.isMakeup">Отработка</label>
        <div v-if="mode === 'group'">
          <label class="label" for="lesson-title">Название группы</label>
          <input
            id="lesson-title"
            v-model="form.title"
            class="field"
            maxlength="70"
          />
        </div>

        <div v-if="mode === 'group'">
          <span class="label">Сколько мест</span>
          <div
            class="flex items-center justify-between rounded-xl border border-line bg-paper/60 p-2 pl-3"
          >
            <span class="text-xs font-semibold text-muted">Дети в группе</span>
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="grid h-8 w-8 place-items-center rounded-lg border border-line bg-card"
                :disabled="groupSeats <= 2"
                aria-label="Меньше мест"
                @click="groupSeats = Math.max(2, groupSeats - 1)"
              >
                <i class="fa-solid fa-minus text-[10px]" aria-hidden="true"></i>
              </button>
              <span
                class="w-7 text-center font-display text-lg font-semibold"
                >{{ groupSeats }}</span
              >
              <button
                type="button"
                class="grid h-8 w-8 place-items-center rounded-lg border border-line bg-card"
                :disabled="groupSeats >= 8"
                aria-label="Больше мест"
                @click="groupSeats = Math.min(8, groupSeats + 1)"
              >
                <i class="fa-solid fa-plus text-[10px]" aria-hidden="true"></i>
              </button>
            </div>
          </div>
        </div>

        <p
          v-if="formError"
          class="rounded-xl bg-clay-soft px-3 py-2 text-xs font-semibold text-clay"
          role="alert"
        >
          {{ formError }}
        </p>

        <label class="label"
          >Цена разового занятия для новой группы, ₽<input
            v-model.number="form.singlePrice"
            type="number"
            min="1"
            class="field"
        /></label>
        <p class="text-xs text-muted">
          При новом названии создаётся группа. Для неё применяются выбранные
          правила списаний и эта цена. Для существующей группы сохраняются её
          условия.
        </p>
        <button type="button" class="btn-primary w-full" @click="createLesson">
          <i class="fa-solid fa-plus text-xs" aria-hidden="true"></i>
          Создать занятие
        </button>
        <p class="text-center text-[10px] leading-relaxed text-muted">
          Вместимость проверяется автоматически: записать больше мест не
          получится.
        </p>
      </div>
    </BottomSheet>

    <BottomSheet
      :open="enrollOpen"
      title="Записать ребёнка"
      :subtitle="
        enrollLesson ? `${enrollLesson.time} · ${enrollLesson.title}` : ''
      "
      @close="enrollOpen = false"
    >
      <div v-if="enrollLesson" class="space-y-4">
        <div
          class="flex items-center justify-between rounded-xl bg-paper/80 px-3 py-2.5"
        >
          <span class="text-xs font-semibold text-muted">Свободных мест</span>
          <span class="font-display text-lg font-semibold">
            {{ enrollLesson.capacity - enrollLesson.childIds.length }}
            <span class="text-xs font-sans text-muted"
              >из {{ enrollLesson.capacity }}</span
            >
          </span>
        </div>

        <div class="max-h-72 space-y-1.5 overflow-y-auto pr-1">
          <button
            v-for="row in enrollOptions"
            :key="row.child.id"
            type="button"
            class="flex w-full items-center gap-2.5 rounded-xl border p-2.5 text-left transition"
            :class="
              row.state === 'enrolled' || row.state === 'full'
                ? 'cursor-not-allowed border-line/60 bg-paper/50 opacity-60'
                : 'border-line bg-card hover:border-forest/30'
            "
            :disabled="row.state !== 'ok'"
            @click="pickChild(row.child.id)"
          >
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full text-[10px] font-bold"
              :class="avatarTone(row.child.id)"
            >
              {{ initials(row.child.name) }}
            </span>
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-bold"
                >{{ row.child.name }}, {{ row.child.age }}</span
              >
              <span class="block text-[11px] text-muted">{{ row.family }}</span>
            </span>
            <span
              class="chip"
              :class="
                row.state === 'enrolled'
                  ? 'bg-forest-soft text-forest'
                  : 'bg-paper text-muted'
              "
            >
              {{
                row.state === "enrolled"
                  ? "уже записан"
                  : row.state === "full"
                    ? "мест нет"
                    : `${row.remaining} зан.`
              }}
            </span>
          </button>
        </div>

        <p
          class="rounded-xl bg-honey-soft/60 px-3 py-2 text-[11px] leading-relaxed text-ink"
        >
          Одно и то же занятие нельзя записать дважды — повторная запись
          заблокирована.
        </p>
      </div>
    </BottomSheet>

    <AppToast :message="toastMessage" />
  </AppShell>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import AppShell from "../components/AppShell.vue";
import AppToast from "../components/AppToast.vue";
import BottomSheet from "../components/BottomSheet.vue";
import {
  DEMO_TODAY,
  avatarTone,
  formatDay,
  formatWeekday,
  initials,
  lessonStatus,
  plural,
  type Child,
  type Lesson,
  type LessonStatus,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const props = defineProps<{ date: string }>();

const {
  state,
  childById,
  familyForChild,
  subscriptionFor,
  lessonById,
  lessonsFor,
  addLesson,
  enroll,
} = useDemo();
const { message: toastMessage, show: notify } = useToast();

const date = computed(() => props.date || DEMO_TODAY);
const dayLessons = computed(() => lessonsFor(date.value));
const totalEnrolled = computed(() =>
  dayLessons.value.reduce((sum, lesson) => sum + lesson.childIds.length, 0),
);
const freeSeats = computed(() =>
  dayLessons.value.reduce(
    (sum, lesson) =>
      sum + Math.max(0, lesson.capacity - lesson.childIds.length),
    0,
  ),
);

const groupSeats = ref(6);
const createOpen = ref(false);
const mode = ref<"group" | "individual">("group");
const formError = ref("");
const form = reactive({
  isMakeup: false,
  singlePrice: 1000,
  time: "16:30",
  duration: 90,
  title: "Столярная мастерская · 7–10 лет",
  childId: "",
});

const endTime=computed(()=>{const t=new Date(date.value+'T'+form.time+':00+05:00');if(!Number.isFinite(t.getTime())||!Number.isFinite(form.duration))return '—';return new Date(t.getTime()+form.duration*60000).toLocaleTimeString('ru-RU',{timeZone:'Asia/Yekaterinburg',hour:'2-digit',minute:'2-digit'});});

const enrollOpen = ref(false);
const enrollLessonId = ref("");
const enrollLesson = computed(() =>
  enrollLessonId.value ? lessonById(enrollLessonId.value) : undefined,
);

const childOptions = computed(() =>
  state.children.map((child) => {
    const family = familyForChild(child);
    return {
      child,
      family: family ? family.parent : "",
      remaining: subscriptionFor(child.id)?.remaining ?? 0,
    };
  }),
);

type EnrollState = "ok" | "enrolled" | "full";

const enrollOptions = computed(() => {
  const lesson = enrollLesson.value;
  return state.children.map((child) => {
    const family = familyForChild(child);
    let status: EnrollState = "ok";
    if (lesson?.childIds.includes(child.id)) status = "enrolled";
    else if (lesson && lesson.childIds.length >= lesson.capacity)
      status = "full";
    return {
      child,
      family: family ? family.parent : "",
      remaining: subscriptionFor(child.id)?.remaining ?? 0,
      state: status,
    };
  });
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

function enrolledPreview(lesson: Lesson) {
  return lesson.childIds
    .map((id) => childById(id))
    .filter((child): child is Child => Boolean(child))
    .slice(0, 5);
}

function openCreate(next: "group" | "individual") {
  mode.value = next;
  formError.value = "";
  form.time = next === "group" ? "16:30" : "13:00";
  form.duration = 90;
  form.title = "Столярная мастерская · 7–10 лет";
  form.isMakeup = false;
  form.childId = "";
  createOpen.value = true;
}

async function createLesson() {
  formError.value = "";
  const existing = state.lessons.some(
    (lesson) => lesson.date === date.value && lesson.time === form.time,
  );
  if (existing) {
    formError.value = "На это время уже есть занятие. Выберите другое время.";
    return;
  }

  await addLesson({
    singlePrice: form.singlePrice,
    date: date.value,
    time: form.time,
    duration: form.duration,
    kind: mode.value,
    title:
      mode.value === "group"
        ? form.title.trim() || "Групповое занятие"
        : "Индивидуальное занятие",
    capacity: mode.value === "group" ? groupSeats.value : 1,
    childIds: [],
    isMakeup: mode.value==="individual"&&form.isMakeup,
  });

  createOpen.value = false;
  notify("Занятие добавлено в расписание");
}

function openEnroll(lesson: Lesson) {
  enrollLessonId.value = lesson.id;
  enrollOpen.value = true;
}

async function pickChild(childId: string) {
  const lesson = enrollLesson.value;
  if (!lesson) return;
  const result = await enroll(lesson.id, childId);
  if (result === "ok") {
    const name = childById(childId)?.name ?? "Ребёнок";
    notify(`${name} записан(а) на ${lesson.time}`);
    enrollOpen.value = false;
  } else if (result === "duplicate") {
    notify("Этот ребёнок уже записан на занятие");
  } else if (result === "full") {
    notify("Свободных мест нет");
  }
}
</script>
