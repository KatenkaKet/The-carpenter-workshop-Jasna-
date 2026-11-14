<template>
  <AppShell active="clients">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Родители и ученики</p>
        <h1 class="page-title mt-1.5">Клиенты</h1>
        <p class="mt-1.5 text-sm text-muted">
          Семья, дети, остатки абонементов и контакты — в одном месте.
        </p>
      </div>
      <button type="button" class="btn-primary" @click="openAdd">
        <i class="fa-solid fa-plus text-xs" aria-hidden="true"></i>
        Добавить семью
      </button>
    </div>

    <div class="mt-4 grid grid-cols-3 gap-2">
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Семей
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold">
          {{ state.families.length }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Детей
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold">
          {{ state.children.length }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
          Без абонемента
        </p>
        <p class="mt-0.5 font-display text-2xl font-semibold text-clay">
          {{ emptyCount }}
        </p>
      </div>
    </div>

    <div class="card mt-4 p-2">
      <label class="flex items-center gap-2 rounded-xl bg-paper/70 px-3">
        <i
          class="fa-solid fa-magnifying-glass text-xs text-muted"
          aria-hidden="true"
        ></i>
        <input
          v-model="search"
          type="search"
          class="min-h-[42px] w-full bg-transparent text-sm text-ink placeholder:text-muted/60 focus:outline-none"
          placeholder="Имя родителя, ребёнка или телефон"
          aria-label="Поиск по клиентам"
        />
        <button
          v-if="search"
          type="button"
          class="grid h-7 w-7 place-items-center rounded-full text-muted hover:bg-card hover:text-ink"
          aria-label="Очистить поиск"
          @click="search = ''"
        >
          <i class="fa-solid fa-xmark text-[10px]" aria-hidden="true"></i>
        </button>
      </label>
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

    <div
      v-if="visibleRows.length"
      class="mt-4 grid items-start gap-3 lg:grid-cols-2"
    >
      <article
        v-for="row in visibleRows"
        :key="row.family.id"
        class="card overflow-hidden"
      >
        <button
          type="button"
          class="flex w-full items-start gap-3 p-4 text-left"
          :aria-expanded="expanded === row.family.id"
          @click="toggle(row.family.id)"
        >
          <span
            class="grid h-11 w-11 shrink-0 place-items-center rounded-full text-[12px] font-bold"
            :class="avatarTone(row.family.id)"
          >
            {{ initials(row.family.parent) }}
          </span>
          <span class="min-w-0 flex-1">
            <span class="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span class="text-sm font-bold">{{ row.family.parent }}</span>
              <span class="chip bg-paper text-muted">
                {{ row.family.childIds.length }}
                {{
                  plural(
                    row.family.childIds.length,
                    "ребёнок",
                    "ребёнка",
                    "детей",
                  )
                }}
              </span>
            </span>
            <span class="mt-0.5 block text-[11px] text-muted">{{
              row.family.phone
            }}</span>
            <span class="mt-2 flex flex-wrap gap-1.5">
              <span
                v-for="child in row.children"
                :key="child.id"
                class="chip"
                :class="
                  child.remaining > 0
                    ? 'bg-forest-soft text-forest'
                    : 'bg-clay-soft text-clay'
                "
              >
                {{ child.name }}, {{ child.age }} · {{ child.remaining }} зан.
              </span>
            </span>
          </span>
          <i
            :class="
              expanded === row.family.id
                ? 'fa-solid fa-chevron-up'
                : 'fa-solid fa-chevron-down'
            "
            class="mt-1 text-[10px] text-muted"
            aria-hidden="true"
          ></i>
        </button>

        <div
          v-if="expanded === row.family.id"
          class="border-t border-line/70 bg-paper/40 p-4"
        >
          <ul class="space-y-2">
            <li
              v-for="child in row.children"
              :key="child.id"
              class="flex items-center gap-2.5 rounded-xl border border-line/70 bg-card p-2.5"
            >
              <span
                class="grid h-9 w-9 shrink-0 place-items-center rounded-full text-[10px] font-bold"
                :class="avatarTone(child.id)"
              >
                {{ initials(child.name) }}
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-xs font-bold"
                  >{{ child.name }}, {{ child.age }} лет</span
                >
                <span class="block text-[10px] text-muted">{{
                  child.status
                }}</span>
              </span>
              <a
                :href="`/yasna/day?date=${DEMO_TODAY}`"
                class="btn-quiet !min-h-[34px] border border-line !px-2.5 text-[10px]"
              >
                Записать
              </a>
            </li>
          </ul>

          <div class="mt-3 flex gap-2">
            <a
              :href="`tel:${telHref(row.family.phone)}`"
              class="btn-ghost flex-1 !min-h-[40px] text-xs"
            >
              <i class="fa-solid fa-phone text-[10px]" aria-hidden="true"></i>
              Позвонить
            </a>
            <a
              :href="`/yasna/subscriptions?tab=rules`"
              class="btn-quiet !min-h-[40px] border border-line text-xs"
            >
              Абонементы
            </a>
          </div>
          <button
            type="button"
            class="btn-quiet mt-2 w-full border border-line !min-h-[40px] text-xs"
            @click="openParentLink(row.family)"
          >
            <i class="fa-solid fa-link text-[10px]" aria-hidden="true"></i>
            Ссылка для родителя
          </button>
        </div>
      </article>
    </div>

    <div v-else class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-magnifying-glass" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">Никого не нашли</p>
      <p class="mt-1 text-xs text-muted">
        Измените запрос или добавьте новую семью.
      </p>
      <button type="button" class="btn-primary mt-4" @click="openAdd">
        Добавить семью
      </button>
    </div>

    <BottomSheet
      :open="addOpen"
      title="Новая семья"
      subtitle="Родитель и один или несколько детей"
      @close="addOpen = false"
    >
      <div class="space-y-3">
        <div>
          <label class="label" for="parent-name">Родитель</label>
          <input
            id="parent-name"
            v-model="addForm.parent"
            class="field"
            placeholder="Имя и фамилия"
            autocomplete="name"
          />
        </div>
        <div>
          <label class="label" for="parent-phone">Телефон</label>
          <input
            id="parent-phone"
            v-model="addForm.phone"
            class="field"
            placeholder="+7 900 000-00-00"
            inputmode="tel"
            autocomplete="tel"
          />
        </div>

        <div class="rounded-2xl border border-line/80 bg-paper/50 p-3">
          <div class="flex items-center justify-between">
            <span class="text-xs font-bold text-muted">Дети</span>
            <button
              type="button"
              class="text-[11px] font-bold text-honey"
              @click="addChildField"
            >
              <i class="fa-solid fa-plus text-[9px]" aria-hidden="true"></i>
              ещё ребёнок
            </button>
          </div>
          <div class="mt-2 space-y-2">
            <div
              v-for="(kid, index) in addForm.children"
              :key="index"
              class="flex items-end gap-2"
            >
              <div class="min-w-0 flex-1">
                <label class="label" :for="`kid-name-${index}`"
                  >Имя ребёнка</label
                >
                <input
                  :id="`kid-name-${index}`"
                  v-model="kid.name"
                  class="field"
                  placeholder="Имя"
                />
              </div>
              <div class="w-[74px] shrink-0">
                <label class="label" :for="`kid-age-${index}`">Возраст</label>
                <input
                  :id="`kid-age-${index}`"
                  v-model="kid.age"
                  class="field"
                  placeholder="9"
                  inputmode="numeric"
                  maxlength="2"
                />
              </div>
              <button
                v-if="addForm.children.length > 1"
                type="button"
                class="mb-0.5 grid h-[42px] w-9 shrink-0 place-items-center rounded-xl border border-line bg-card text-muted hover:text-clay"
                aria-label="Убрать ребёнка"
                @click="removeChildField(index)"
              >
                <i class="fa-solid fa-xmark text-[10px]" aria-hidden="true"></i>
              </button>
            </div>
          </div>
        </div>

        <p
          v-if="addError"
          class="rounded-xl bg-clay-soft px-3 py-2 text-xs font-semibold text-clay"
          role="alert"
        >
          {{ addError }}
        </p>

        <button type="button" class="btn-primary w-full" @click="saveFamily">
          <i class="fa-solid fa-plus text-xs" aria-hidden="true"></i>
          Добавить семью
        </button>
        <p class="text-center text-[10px] leading-relaxed text-muted">
          Родитель получит персональную ссылку на кабинет — без почты и пароля.
        </p>
      </div>
    </BottomSheet>

    <BottomSheet
      :open="linkOpen"
      title="Ссылка для родителя"
      :subtitle="linkFamily ? linkFamily.parent : ''"
      @close="linkOpen = false"
    >
      <div class="space-y-3">
        <p class="text-xs leading-relaxed text-muted">
          Родитель открывает ссылку и сразу попадает в кабинет своей семьи — без
          почты и пароля. Ссылку можно передать в мессенджере.
        </p>
        <label class="label" for="parent-link">Персональная ссылка</label>
        <input
          id="parent-link"
          :value="linkUrl"
          readonly
          class="field text-[11px]"
          @focus="($event.target as HTMLInputElement).select()"
        />
        <button type="button" class="btn-primary w-full" @click="copyLink">
          <i class="fa-solid fa-copy text-xs" aria-hidden="true"></i>
          Скопировать ссылку
        </button>
        <p
          class="rounded-xl bg-honey-soft/55 px-3 py-2 text-[10px] leading-relaxed"
        >
          Ссылка действует, пока мастер её не заменит. Если она попала не в те
          руки — выпустите новую, и старая перестанет работать.
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
  initials,
  plural,
  type Child,
  type Family,
  type Subscription,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const { state, childById, subscriptionFor, addFamily } = useDemo();
const { message: toastMessage, show: notify } = useToast();

const search = ref("");
const activeFilter = ref<"all" | "attention" | "empty">("all");
const expanded = ref("");

interface FamilyRow {
  family: Family;
  children: Array<Child & { remaining: number; status: string }>;
}

const rows = computed<FamilyRow[]>(() => {
  const result: FamilyRow[] = [];
  for (const family of state.families) {
    const children: FamilyRow["children"] = [];
    for (const id of family.childIds) {
      const child = childById(id);
      if (!child) continue;
      const sub: Subscription | undefined = subscriptionFor(id);
      const remaining = sub?.remaining ?? 0;
      let status = "Без абонемента";
      if (sub && sub.total > 0) {
        if (sub.validUntil < DEMO_TODAY) status = "Абонемент истёк";
        else if (remaining <= 0) status = "Абонемент закончился";
        else if (remaining <= 1) status = "Заканчивается абонемент";
        else status = `Осталось ${remaining} из ${sub.total} занятий`;
      }
      children.push({ ...child, remaining, status });
    }
    result.push({ family, children });
  }
  return result;
});

const emptyCount = computed(
  () =>
    rows.value.filter((row) =>
      row.children.some((child) => child.remaining <= 0),
    ).length,
);

const visibleRows = computed(() => {
  const query = search.value.trim().toLowerCase();
  return rows.value.filter((row) => {
    const matchesQuery =
      !query ||
      row.family.parent.toLowerCase().includes(query) ||
      row.family.phone.toLowerCase().includes(query) ||
      row.children.some((child) => child.name.toLowerCase().includes(query));

    const matchesFilter =
      activeFilter.value === "all" ||
      (activeFilter.value === "attention" &&
        row.children.some((child) => child.remaining <= 1)) ||
      (activeFilter.value === "empty" &&
        row.children.some((child) => child.remaining <= 0));

    return matchesQuery && matchesFilter;
  });
});

const filters = [
  { key: "all", label: "Все семьи" },
  { key: "attention", label: "Заканчивается" },
  { key: "empty", label: "Без занятий" },
] as const;

function toggle(id: string) {
  expanded.value = expanded.value === id ? "" : id;
}

function telHref(phone: string): string {
  return phone.replace(/[^\d+]/g, "");
}

const linkOpen = ref(false);
const linkUrl = ref("");
const linkFamily = ref<Family | null>(null);

function openParentLink(family: Family) {
  const origin =
    typeof window !== "undefined"
      ? window.location.origin
      : "https://serbaulia5.chatium.ru";
  linkFamily.value = family;
  linkUrl.value = `${origin}/yasna/parent#token=${family.accessToken}`;
  linkOpen.value = true;
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(linkUrl.value);
    notify("Ссылка скопирована");
    linkOpen.value = false;
  } catch {
    notify("Не удалось скопировать — выделите ссылку вручную");
  }
}

const addOpen = ref(false);
const addError = ref("");
const addForm = reactive({
  parent: "",
  phone: "",
  children: [{ name: "", age: "" }] as Array<{ name: string; age: string }>,
});

function openAdd() {
  addForm.parent = "";
  addForm.phone = "";
  addForm.children = [{ name: "", age: "" }];
  addError.value = "";
  addOpen.value = true;
}

function addChildField() {
  if (addForm.children.length >= 4) return;
  addForm.children.push({ name: "", age: "" });
}

function removeChildField(index: number) {
  addForm.children.splice(index, 1);
}

function saveFamily() {
  addError.value = "";
  const parent = addForm.parent.trim();
  const phone = addForm.phone.trim();

  if (!parent) {
    addError.value = "Укажите имя родителя.";
    return;
  }
  if (phone.replace(/\D/g, "").length < 10) {
    addError.value = "Укажите телефон — не меньше 10 цифр.";
    return;
  }

  const kids: Array<{ name: string; age: number }> = [];
  for (const kid of addForm.children) {
    const name = kid.name.trim();
    const age = Number(kid.age);
    if (!name) {
      addError.value = "Укажите имя ребёнка.";
      return;
    }
    if (!Number.isFinite(age) || age < 3 || age > 17) {
      addError.value = "Возраст ребёнка — от 3 до 17 лет.";
      return;
    }
    kids.push({ name, age });
  }

  if (!kids.length) {
    addError.value = "Добавьте хотя бы одного ребёнка.";
    return;
  }

  const family = addFamily(parent, phone, kids);
  addOpen.value = false;
  expanded.value = family.id;
  notify("Семья добавлена в базу");
}
</script>
