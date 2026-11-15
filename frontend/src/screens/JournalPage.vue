<template>
  <AppShell active="journal">
    <div>
      <p class="eyebrow">Посещаемость</p>
      <h1 class="page-title mt-1.5">Журнал</h1>
      <p class="mt-1.5 text-sm text-muted">
        Отметьте посещение, пропуск или отмену занятия.
      </p>
    </div>

    <div class="card mt-5 flex items-center gap-1 p-1.5">
      <button
        type="button"
        class="grid h-10 w-10 shrink-0 place-items-center rounded-xl text-muted transition hover:bg-paper hover:text-ink"
        aria-label="Предыдущий день"
        @click="shiftDate(-1)"
      >
        <i class="fa-solid fa-chevron-left text-xs" aria-hidden="true"></i>
      </button>
      <div class="min-w-0 flex-1 text-center">
        <p class="truncate text-sm font-bold">{{ formatDateLong(date) }}</p>
        <p class="mt-0.5 text-[10px] font-semibold text-muted">
          {{ lessons.length }}
          {{ plural(lessons.length, "занятие", "занятия", "занятий") }}
          <button
            v-if="date !== DEMO_TODAY"
            type="button"
            class="ml-1 text-honey underline decoration-honey/40 underline-offset-2"
            @click="date = DEMO_TODAY"
          >
            сегодня
          </button>
        </p>
      </div>
      <button
        type="button"
        class="grid h-10 w-10 shrink-0 place-items-center rounded-xl text-muted transition hover:bg-paper hover:text-ink"
        aria-label="Следующий день"
        @click="shiftDate(1)"
      >
        <i class="fa-solid fa-chevron-right text-xs" aria-hidden="true"></i>
      </button>
    </div>

    <div
      v-if="lessons.length"
      class="mt-4 flex gap-2 overflow-x-auto pb-1 scroll-x"
    >
      <button
        v-for="lesson in lessons"
        :key="lesson.id"
        type="button"
        class="min-w-[168px] flex-1 rounded-2xl border p-3 text-left transition"
        :class="
          selected?.id === lesson.id
            ? 'border-forest bg-forest text-card shadow-lift'
            : 'border-line bg-card text-ink hover:border-forest/30'
        "
        @click="selectedId = lesson.id"
      >
        <span
          class="text-[10px] font-bold uppercase tracking-wide"
          :class="selected?.id === lesson.id ? 'text-card/70' : 'text-muted'"
        >
          {{ lesson.time }} · {{ lesson.duration }} мин
        </span>
        <span class="mt-1 block truncate text-sm font-bold">{{
          lesson.title
        }}</span>
        <span
          class="mt-1 flex items-center gap-1 text-[10px] font-semibold"
          :class="selected?.id === lesson.id ? 'text-card/70' : 'text-muted'"
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
          {{ lesson.childIds.length }} из {{ lesson.capacity }} детей
        </span>
      </button>
    </div>

    <section
      v-if="selected"
      class="mt-4 grid grid-cols-3 gap-2"
      aria-label="Итоги занятия"
    >
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Посетили
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold text-forest">
          {{ presentCount }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Пропустили
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold text-clay">
          {{ absentCount }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Отмены
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold">
          {{ cancelCount }}
        </p>
      </div>

    </section>

    <div
      v-if="selected"
      class="mt-4 flex flex-wrap items-center justify-between gap-2"
    >
      <h2 class="section-title">Дети на занятии</h2>
      <button
        type="button"
        class="btn-quiet !min-h-[36px] border border-line text-[11px]"
        @click="markAllPresent"
      >
        <i class="fa-solid fa-check-double text-[10px]" aria-hidden="true"></i>
        Отметить всех пришедшими
      </button>
    </div>

    <ul
      v-if="selected && rows.length"
      class="card mt-3 divide-y divide-line/70"
    >
      <li v-for="row in rows" :key="row.child.id" class="p-4">
        <div class="sm:flex sm:items-center sm:gap-4">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <span
              class="grid h-10 w-10 shrink-0 place-items-center rounded-full text-[11px] font-bold"
              :class="avatarTone(row.child.id)"
            >
              {{ initials(row.child.name) }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <p class="text-sm font-bold">
                  {{ row.child.name }}, {{ row.child.age }}
                </p>
                <span class="chip" :class="remainingTone(row.remaining)">
                  осталось {{ row.remaining }}
                </span>
              </div>
              <p class="mt-0.5 truncate text-[11px] text-muted">
                {{ row.family?.parent }} · {{ row.family?.phone }}
              </p>
            </div>
            <span
              v-if="row.mark"
              class="chip shrink-0"
              :class="markChip(row.mark.status)"
            >
              {{ markLabel(row.mark.status) }}
            </span>
          </div>

          <div
            class="mt-3 grid grid-cols-3 gap-1.5 sm:mt-0 sm:w-[320px] sm:shrink-0"
            role="group"
            :aria-label="`Отметка для ${row.child.name}`"
          >
            <button
              type="button"
              class="flex min-h-[40px] items-center justify-center gap-1.5 rounded-xl border text-[11px] font-bold transition"
              :class="markButton(row.mark?.status === 'present', 'present')"
              @click="apply(row.child.id, 'present')"
            >
              <i class="fa-solid fa-check text-[10px]" aria-hidden="true"></i>
              Пришёл
            </button>
            <button
              type="button"
              class="flex min-h-[40px] items-center justify-center gap-1.5 rounded-xl border text-[11px] font-bold transition"
              :class="markButton(row.mark?.status === 'absent', 'absent')"
              @click="apply(row.child.id, 'absent')"
            >
              <i class="fa-solid fa-xmark text-[10px]" aria-hidden="true"></i>
              Пропустил
            </button>
            <button
              type="button"
              class="flex min-h-[40px] items-center justify-center gap-1.5 rounded-xl border text-[11px] font-bold transition"
              :class="markButton(row.mark?.status === 'cancel', 'cancel')"
              @click="apply(row.child.id, 'cancel')"
            >
              <i
                class="fa-solid fa-arrow-rotate-left text-[10px]"
                aria-hidden="true"
              ></i>
              Отмена
            </button>
          </div>
        </div>

        <div v-if="row.mark && row.mark.history.length" class="mt-2">
          <button
            type="button"
            class="flex items-center gap-1.5 text-[10px] font-bold text-muted transition hover:text-ink"
            @click="toggleHistory(row.child.id)"
          >
            <i
              :class="
                expanded.includes(row.child.id)
                  ? 'fa-solid fa-chevron-up'
                  : 'fa-solid fa-chevron-down'
              "
              class="text-[8px]"
              aria-hidden="true"
            ></i>
            История изменений · {{ row.mark.history.length }}
          </button>
          <ol
            v-if="expanded.includes(row.child.id)"
            class="mt-1.5 space-y-1 rounded-xl border border-line/70 bg-paper/50 px-3 py-2"
          >
            <li
              v-for="(change, index) in row.mark.history"
              :key="index"
              class="flex items-baseline justify-between gap-3 text-[10px] text-muted"
            >
              <span>{{ change.label }}</span>
              <span class="shrink-0 tabular-nums">{{ change.at }}</span>
            </li>
          </ol>
        </div>
      </li>
    </ul>

    <div
      v-if="selected && !rows.length"
      class="card mt-3 px-5 py-10 text-center"
    >
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-user-plus" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">
        На занятие ещё никто не записан
      </p>
      <a :href="`/yasna/day?date=${date}`" class="btn-primary mt-4"
        >Записать ребёнка</a
      >
    </div>

    <div v-if="!lessons.length" class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-14 w-14 place-items-center rounded-full bg-paper text-xl text-muted"
      >
        <i class="fa-solid fa-calendar-xmark" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">
        В этот день занятий нет
      </p>
      <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
        Выберите другой день или создайте занятие в расписании.
      </p>
      <a :href="`/yasna/day?date=${date}`" class="btn-ghost mt-4"
        >Открыть день</a
      >
    </div>

    <p class="mt-5 text-center text-[11px] leading-relaxed text-muted/70">
      Изменения отметок сохраняются в истории · списание зависит от правил
      абонемента
    </p>

    <AppToast :message="toastMessage" />
  </AppShell>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import AppShell from "../components/AppShell.vue";
import AppToast from "../components/AppToast.vue";

import {
  DEMO_TODAY,
  addDays,
  avatarTone,
  formatDateLong,
  initials,
  markLabel,
  plural,
  type Child,
  type Family,
  type Mark,
  type MarkStatus,
  type Subscription,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const props = withDefaults(defineProps<{ date?: string; lesson?: string }>(), {
  date: "",
  lesson: "",
});

const {
  lessonsFor,
  lessonById,
  childById,
  familyForChild,
  subscriptionFor,
  markFor,
  setMark,
} = useDemo();
const { message: toastMessage, show: notify } = useToast();

const date = ref(props.date || DEMO_TODAY);
const selectedId = ref(props.lesson || "");

const lessons = computed(() => lessonsFor(date.value));

const selected = computed(() => {
  const match = selectedId.value ? lessonById(selectedId.value) : undefined;
  if (match && match.date === date.value) return match;
  return lessons.value[0];
});

watch(date, () => {
  selectedId.value = lessons.value[0]?.id ?? "";
});

interface StudentRow {
  child: Child;
  family: Family | undefined;
  sub: Subscription | undefined;
  mark: Mark | undefined;
  remaining: number;
}

const rows = computed<StudentRow[]>(() => {
  const lesson = selected.value;
  if (!lesson) return [];
  const result: StudentRow[] = [];
  for (const id of lesson.childIds) {
    const child = childById(id);
    if (!child) continue;
    const sub = subscriptionFor(id);
    result.push({
      child,
      family: familyForChild(child),
      sub,
      mark: markFor(lesson.id, id),
      remaining: sub?.remaining ?? 0,
    });
  }
  return result;
});

const presentCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "present").length,
);
const absentCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "absent").length,
);
const cancelCount = computed(
  () => rows.value.filter((row) => row.mark?.status === "cancel").length,
);


const expanded = ref<string[]>([]);

function shiftDate(offset: number) {
  date.value = addDays(date.value, offset);
}

function remainingTone(remaining: number): string {
  if (remaining <= 0) return "bg-clay-soft text-clay";
  if (remaining === 1) return "bg-honey-soft text-honey";
  return "bg-forest-soft text-forest";
}

function markChip(status: MarkStatus): string {
  if (status === "present") return "bg-forest-soft text-forest";
  if (status === "absent") return "bg-clay-soft text-clay";
  return "bg-paper text-muted";
}

function markButton(active: boolean, tone: MarkStatus): string {
  if (active) {
    if (tone === "present") return "border-forest bg-forest text-card";
    if (tone === "absent") return "border-clay bg-clay text-white";
    return "border-ink bg-ink text-card";
  }
  return "border-line bg-card text-muted hover:border-forest/30 hover:text-ink";
}

async function apply(childId: string, status: MarkStatus) {
  const lesson = selected.value;
  if (!lesson) return;
  const result = await setMark(lesson.id, childId, status);
  const name = childById(childId)?.name ?? "Ребёнок";
  if (status === "present") {
    const tail = result.delta < 0 ? ` · осталось ${result.remaining}` : "";
    notify(`${name}: посещение отмечено${tail}`);
  } else if (status === "absent") {
    notify(`${name}: пропуск отмечен`);
  } else {
    notify(`${name}: занятие не списано`);
  }
}



function toggleHistory(childId: string) {
  expanded.value = expanded.value.includes(childId)
    ? expanded.value.filter((id) => id !== childId)
    : [...expanded.value, childId];
}

async function markAllPresent() {
  const lesson = selected.value;
  if (!lesson) return;
  let count = 0;
  for (const row of rows.value) {
    if (row.mark?.status !== "present") {
      await setMark(lesson.id, row.child.id, "present");
      count += 1;
    }
  }
  notify(count ? `Отмечено пришедшими: ${count}` : "Все уже отмечены");
}
</script>
