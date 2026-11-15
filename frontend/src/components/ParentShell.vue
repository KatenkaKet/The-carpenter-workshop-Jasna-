<template>
  <div class="min-h-screen">
    <header
      class="sticky top-0 z-30 border-b border-line/70 bg-paper/90 backdrop-blur"
    >
      <div
        class="mx-auto flex w-full max-w-[1000px] items-center gap-3 px-4 py-3 sm:px-6"
      >
        <a href="/yasna/parent" aria-label="Кабинет семьи">
          <BrandMark />
        </a>
        <div class="hidden min-w-0 border-l border-line/80 pl-3 sm:block">
          <p class="text-[10px] font-bold uppercase tracking-wider text-honey">
            Кабинет семьи
          </p>
          <p class="truncate text-sm font-bold">{{ family.parent }}</p>
        </div>

        <div class="ml-auto flex items-center gap-1">
          <button
            type="button"
            class="relative grid h-10 w-10 place-items-center rounded-full text-ink transition hover:bg-card"
            :aria-expanded="panel"
            aria-label="Уведомления"
            @click="panel = !panel"
          >
            <i class="fa-solid fa-bell" aria-hidden="true"></i>
            <span
              v-if="unread"
              class="absolute right-1 top-1 grid h-4 min-w-4 place-items-center rounded-full bg-clay px-1 text-[9px] font-bold text-white ring-2 ring-paper"
            >
              {{ unread }}
            </span>
          </button>
          <span
            class="grid h-9 w-9 place-items-center rounded-full bg-forest text-[11px] font-bold text-card"
            :title="family.parent"
          >
            {{ initials(family.parent) }}
          </span>

        </div>
      </div>

      <div class="hidden border-t border-line/60 sm:block">
        <nav
          class="mx-auto flex w-full max-w-[1000px] gap-1 px-4 sm:px-6"
          aria-label="Разделы кабинета"
        >
          <a
            v-for="item in nav"
            :key="item.key"
            :href="item.href"
            class="flex items-center gap-2 border-b-2 px-3 py-2.5 text-xs font-bold transition"
            :class="
              item.key === active
                ? 'border-honey text-ink'
                : 'border-transparent text-muted hover:text-ink'
            "
            :aria-current="item.key === active ? 'page' : undefined"
          >
            <i :class="item.icon" class="text-[13px]" aria-hidden="true"></i>
            {{ item.label }}
          </a>
        </nav>
      </div>
    </header>

    <div
      v-if="panel"
      class="fixed inset-0 z-40"
      aria-hidden="true"
      @click="panel = false"
    ></div>
    <div
      v-if="panel"
      class="fixed right-3 top-[64px] z-50 w-[min(92vw,360px)] rounded-[22px] border border-line bg-card p-4 shadow-lift"
      role="dialog"
      aria-label="Уведомления"
    >
      <div class="flex items-center justify-between">
        <p class="font-display text-lg font-semibold">Уведомления</p>
        <button
          type="button"
          class="text-[11px] font-bold text-honey"
          @click="readAll"
        >
          Прочитать все
        </button>
      </div>
      <ul v-if="notifications.length" class="mt-3 space-y-2">
        <li
          v-for="item in notifications"
          :key="item.id"
          class="flex gap-2.5 rounded-xl p-2.5"
          :class="item.read ? 'bg-transparent' : 'bg-paper/80'"
        >
          <span
            class="mt-0.5 grid h-8 w-8 shrink-0 place-items-center rounded-full text-xs"
            :class="toneClass[item.tone]"
          >
            <i :class="toneIcon[item.tone]" aria-hidden="true"></i>
          </span>
          <span class="min-w-0">
            <span class="block text-xs font-bold">{{ item.title }}</span>
            <span class="mt-0.5 block text-[11px] leading-snug text-muted">{{
              item.text
            }}</span>
            <span class="mt-1 block text-[10px] text-muted/70">{{
              item.time
            }}</span>
          </span>
        </li>
      </ul>
      <p
        v-else
        class="mt-4 rounded-xl bg-paper/80 px-3 py-4 text-center text-xs text-muted"
      >
        Пока нет уведомлений
      </p>
    </div>

    <main
      class="mx-auto w-full max-w-[1000px] px-4 pb-28 pt-5 sm:px-6 sm:pb-10"
    >
      <slot />
    </main>

    <nav
      class="safe-bottom fixed inset-x-0 bottom-0 z-30 border-t border-line/80 bg-card/95 px-3 pt-1.5 backdrop-blur sm:hidden"
      aria-label="Разделы кабинета"
    >
      <div class="mx-auto grid max-w-md grid-cols-4 gap-1">
        <a
          v-for="item in nav"
          :key="item.key"
          :href="item.href"
          class="flex flex-col items-center gap-1 rounded-xl px-2 py-1.5 text-[10px] font-bold transition"
          :class="
            item.key === active ? 'bg-forest-soft text-forest' : 'text-muted'
          "
          :aria-current="item.key === active ? 'page' : undefined"
        >
          <i :class="item.icon" class="text-[17px]" aria-hidden="true"></i>
          {{ item.short }}
        </a>
      </div>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import BrandMark from "./BrandMark.vue";
import { initials, type Family } from "../shared/demo";
import { useDemo } from "../shared/use-demo";

const props = defineProps<{
  family: Family;
  active: "schedule" | "tariffs" | "history" | "payments";
}>();

defineEmits<{ logout: [] }>();

const { notificationsFor, unreadFor, markAllReadFor } = useDemo();
const panel = ref(false);

const notifications = computed(() =>
  notificationsFor("parent", props.family.id),
);
const unread = computed(() => unreadFor("parent", props.family.id));

const nav = [
  {
    key: "schedule",
    label: "Расписание",
    short: "Расписание",
    href: "/yasna/parent",
    icon: "fa-solid fa-calendar-days",
  },
  {
    key: "tariffs",
    label: "Тарифы",
    short: "Тарифы",
    href: "/yasna/parent-tariffs",
    icon: "fa-solid fa-ticket",
  },
  {
    key: "history",
    label: "История посещений",
    short: "История",
    href: "/yasna/parent-history",
    icon: "fa-solid fa-clock-rotate-left",
  },
  {
    key: "payments",
    label: "История оплат",
    short: "Оплаты",
    href: "/yasna/parent-payments",
    icon: "fa-solid fa-receipt",
  },
] as const;

const toneClass: Record<string, string> = {
  warm: "bg-honey-soft text-honey",
  good: "bg-forest-soft text-forest",
  calm: "bg-paper text-muted",
};

const toneIcon: Record<string, string> = {
  warm: "fa-solid fa-triangle-exclamation",
  good: "fa-solid fa-circle-check",
  calm: "fa-solid fa-bell",
};

async function readAll() {
  await markAllReadFor();
  panel.value = false;
}
</script>
