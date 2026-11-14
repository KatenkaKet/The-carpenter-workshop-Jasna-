<template>
  <div class="min-h-screen">
    <aside
      class="fixed inset-y-0 left-0 z-30 hidden w-[252px] flex-col border-r border-line/80 bg-card/80 px-4 py-6 backdrop-blur lg:flex"
    >
      <div class="flex items-center justify-between gap-2 px-2">
        <a href="/yasna" aria-label="Ясна — расписание">
          <BrandMark />
        </a>
        <button
          type="button"
          class="relative grid h-9 w-9 place-items-center rounded-full text-ink transition hover:bg-paper"
          :aria-expanded="panel"
          aria-label="Уведомления"
          @click="panel = !panel"
        >
          <i class="fa-solid fa-bell" aria-hidden="true"></i>
          <span
            v-if="unread"
            class="absolute right-1 top-1 grid h-4 min-w-4 place-items-center rounded-full bg-clay px-1 text-[9px] font-bold text-white ring-2 ring-card"
          >
            {{ unread }}
          </span>
        </button>
      </div>

      <nav class="mt-8 space-y-1" aria-label="Разделы кабинета">
        <a
          v-for="item in nav"
          :key="item.key"
          :href="item.href"
          class="nav-item"
          :class="{ 'nav-item-active': item.key === active }"
          :aria-current="item.key === active ? 'page' : undefined"
        >
          <i
            :class="item.icon"
            class="w-4 text-center text-[15px]"
            aria-hidden="true"
          ></i>
          <span>{{ item.label }}</span>
        </a>
      </nav>

      <div
        class="mt-6 rounded-2xl border border-honey/25 bg-honey-soft/50 p-3.5"
      >
        <p class="flex items-center gap-2 text-xs font-bold text-ink">
          <i class="fa-solid fa-seedling text-honey" aria-hidden="true"></i>
          Демо-режим
        </p>
        <p class="mt-1.5 text-[11px] leading-relaxed text-muted">
          Данные вымышленные и сохраняются только в этом браузере.
        </p>
        <button
          type="button"
          class="mt-2 text-[11px] font-bold text-honey underline decoration-honey/40 underline-offset-2 hover:text-ink"
          @click="reset"
        >
          Сбросить демо-данные
        </button>
      </div>

      <div
        class="mt-auto flex items-center gap-3 rounded-2xl border border-line/80 bg-paper/70 p-3"
      >
        <span
          class="grid h-10 w-10 place-items-center rounded-full bg-forest text-sm font-bold text-card"
        >
          АК
        </span>
        <span class="min-w-0">
          <span class="block truncate text-sm font-bold">Анна Ковалёва</span>
          <span class="block text-[11px] text-muted">Мастер · Ясна</span>
        </span>
      </div>
    </aside>

    <div class="lg:pl-[252px]">
      <header
        class="sticky top-0 z-30 flex items-center justify-between border-b border-line/70 bg-paper/90 px-4 py-3 backdrop-blur sm:px-6 lg:hidden"
      >
        <a href="/yasna" aria-label="Ясна — расписание">
          <BrandMark />
        </a>
        <div class="flex items-center gap-1.5">
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
              class="absolute right-1.5 top-1.5 h-2 w-2 rounded-full bg-clay ring-2 ring-paper"
            ></span>
          </button>
          <span
            class="grid h-9 w-9 place-items-center rounded-full bg-forest text-[11px] font-bold text-card"
            aria-label="Анна Ковалёва, мастер"
          >
            АК
          </span>
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
        class="fixed left-3 right-3 top-[68px] z-50 rounded-[22px] border border-line bg-card p-4 shadow-lift sm:left-auto sm:right-4 sm:w-[340px] lg:left-[268px] lg:right-auto lg:top-[68px]"
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
        <ul class="mt-3 space-y-2">
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
      </div>

      <main
        class="mx-auto w-full max-w-[1080px] px-4 pb-28 pt-5 sm:px-6 lg:pb-12 lg:pt-8"
      >
        <slot />
      </main>
    </div>

    <nav
      class="safe-bottom fixed inset-x-0 bottom-0 z-30 border-t border-line/80 bg-card/95 px-2 pt-1.5 backdrop-blur lg:hidden"
      aria-label="Разделы кабинета"
    >
      <div class="mx-auto grid max-w-md grid-cols-5 gap-0.5">
        <a
          v-for="item in nav"
          :key="item.key"
          :href="item.href"
          class="flex flex-col items-center gap-1 rounded-xl px-0.5 py-1.5 text-[9px] font-bold leading-tight transition"
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
import { useDemo } from "../shared/use-demo";

const props = defineProps<{
  active: "schedule" | "journal" | "subscriptions" | "clients" | "finance";
}>();

const { notificationsFor, unreadFor, markAllReadFor, resetDemo } = useDemo();
const panel = ref(false);

const nav = [
  {
    key: "schedule",
    label: "Расписание",
    short: "Расписание",
    href: "/yasna",
    icon: "fa-solid fa-calendar-days",
  },
  {
    key: "journal",
    label: "Журнал посещаемости",
    short: "Журнал",
    href: "/yasna/journal",
    icon: "fa-solid fa-clipboard-check",
  },
  {
    key: "subscriptions",
    label: "Абонементы",
    short: "Абонементы",
    href: "/yasna/subscriptions",
    icon: "fa-solid fa-ticket",
  },
  {
    key: "clients",
    label: "Клиенты",
    short: "Клиенты",
    href: "/yasna/clients",
    icon: "fa-solid fa-users",
  },
  {
    key: "finance",
    label: "Финансы",
    short: "Финансы",
    href: "/yasna/finance",
    icon: "fa-solid fa-ruble-sign",
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
  calm: "fa-solid fa-calendar-days",
};

const notifications = computed(() => notificationsFor("master"));
const unread = computed(() => unreadFor("master"));

function readAll() {
  markAllReadFor("master");
  panel.value = false;
}

function reset() {
  if (typeof window === "undefined") return;
  if (
    window.confirm(
      "Вернуть демонстрационные данные к началу? Ваши изменения пропадут.",
    )
  ) {
    resetDemo();
    window.location.href = "/yasna";
  }
}
</script>
