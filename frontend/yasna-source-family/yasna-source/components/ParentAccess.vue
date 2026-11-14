<template>
  <div class="flex min-h-screen flex-col">
    <header
      class="mx-auto flex w-full max-w-5xl items-center justify-between px-4 py-5 sm:px-6"
    >
      <BrandMark />
      <a
        href="/yasna"
        class="text-[11px] font-bold text-muted transition hover:text-ink"
      >
        Кабинет мастера
        <i
          class="fa-solid fa-arrow-up-right-from-square ml-1 text-[9px]"
          aria-hidden="true"
        ></i>
      </a>
    </header>

    <main
      class="mx-auto grid w-full max-w-5xl flex-1 items-start gap-5 px-4 pb-12 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:items-center"
    >
      <section class="card p-5 sm:p-7">
        <p class="eyebrow">Кабинет родителя</p>
        <h1 class="page-title mt-2">Ссылка от мастерской</h1>
        <p class="mt-2 max-w-md text-sm leading-relaxed text-muted">
          Мастерская присылает готовую ссылку на кабинет. Просто откройте её —
          пароль, код доступа и регистрация не нужны.
        </p>

        <p
          v-if="error"
          class="mt-4 rounded-xl bg-clay-soft px-3 py-2.5 text-xs font-semibold leading-relaxed text-clay"
          role="alert"
        >
          {{ error }}
        </p>

        <div class="mt-5 space-y-2.5">
          <div class="flex gap-3 rounded-2xl bg-paper/60 p-3">
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-card text-[11px] font-bold text-forest"
            >
              1
            </span>
            <p class="text-xs leading-relaxed">
              <b>Мастерская отправляет ссылку</b><br />
              <span class="text-muted"
                >в мессенджер или сообщением — как удобно вам.</span
              >
            </p>
          </div>
          <div class="flex gap-3 rounded-2xl bg-paper/60 p-3">
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-card text-[11px] font-bold text-forest"
            >
              2
            </span>
            <p class="text-xs leading-relaxed">
              <b>Вы открываете её на телефоне</b><br />
              <span class="text-muted"
                >кабинет открывается сразу, без ввода данных.</span
              >
            </p>
          </div>
          <div class="flex gap-3 rounded-2xl bg-paper/60 p-3">
            <span
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-card text-[11px] font-bold text-forest"
            >
              3
            </span>
            <p class="text-xs leading-relaxed">
              <b>Оплачиваете и записываетесь</b><br />
              <span class="text-muted"
                >расписание, тарифы, чеки и история — внутри.</span
              >
            </p>
          </div>
        </div>

        <p
          class="mt-4 flex items-start gap-2 rounded-xl bg-honey-soft/55 px-3 py-2.5 text-[11px] leading-relaxed text-ink"
        >
          <i
            class="fa-solid fa-lock mt-0.5 text-[10px] text-honey"
            aria-hidden="true"
          ></i>
          Если ссылка не открылась или потерялась — попросите мастерскую
          прислать новую.
        </p>

        <details class="mt-5 rounded-2xl border border-line/80 bg-paper/50 p-3">
          <summary class="cursor-pointer text-xs font-bold text-muted">
            Демо-доступ для проверки
          </summary>
          <div class="mt-2 space-y-1.5">
            <button
              v-for="family in demoFamilies"
              :key="family.id"
              type="button"
              class="flex w-full items-center gap-2.5 rounded-xl border border-line bg-card p-2.5 text-left transition hover:border-forest/30"
              @click="$emit('demo', family.id)"
            >
              <span
                class="grid h-8 w-8 shrink-0 place-items-center rounded-full text-[10px] font-bold"
                :class="avatarTone(family.id)"
              >
                {{ initials(family.parent) }}
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-xs font-bold">{{ family.parent }}</span>
                <span class="block text-[10px] text-muted">
                  {{ family.childIds.length }}
                  {{
                    plural(
                      family.childIds.length,
                      "ребёнок",
                      "ребёнка",
                      "детей",
                    )
                  }}
                </span>
              </span>
              <i
                class="fa-solid fa-chevron-right text-[9px] text-muted/50"
                aria-hidden="true"
              ></i>
            </button>
          </div>
          <p class="mt-2 text-[10px] leading-relaxed text-muted">
            В демо-версии можно открыть кабинет без ссылки. В рабочей системе
            этот блок отключается.
          </p>
        </details>
      </section>

      <aside class="space-y-3">
        <div
          class="rounded-[24px] border border-forest/20 bg-gradient-to-br from-forest-soft to-card p-5"
        >
          <span
            class="grid h-11 w-11 place-items-center rounded-2xl bg-card text-forest"
          >
            <i class="fa-solid fa-calendar-days" aria-hidden="true"></i>
          </span>
          <h2 class="section-title mt-3">Что внутри</h2>
          <ul class="mt-2 space-y-2 text-xs leading-relaxed text-muted">
            <li class="flex gap-2">
              <i
                class="fa-solid fa-check mt-0.5 text-[10px] text-forest"
                aria-hidden="true"
              ></i>
              Расписание и свободные места на ближайшие дни
            </li>
            <li class="flex gap-2">
              <i
                class="fa-solid fa-check mt-0.5 text-[10px] text-forest"
                aria-hidden="true"
              ></i>
              Тарифы: разовое посещение или абонемент
            </li>
            <li class="flex gap-2">
              <i
                class="fa-solid fa-check mt-0.5 text-[10px] text-forest"
                aria-hidden="true"
              ></i>
              Запись ребёнка, история посещений и оплат, чеки
            </li>
          </ul>
        </div>
        <div class="rounded-[24px] border border-honey/25 bg-honey-soft/55 p-4">
          <p class="text-xs font-bold">Одна ссылка — вся семья</p>
          <p class="mt-1 text-[11px] leading-relaxed text-muted">
            В кабинете видны все дети семьи. Переключаться между ними можно при
            записи и в истории.
          </p>
        </div>
      </aside>
    </main>

    <footer class="pb-6 text-center text-[10px] text-muted/70">
      Мастерская «Ясна» · Сургут · демонстрационная версия
    </footer>
  </div>
</template>

<script setup lang="ts">
import BrandMark from "./BrandMark.vue";
import { avatarTone, initials, plural } from "../shared/demo";
import { useDemo } from "../shared/use-demo";

defineProps<{ error?: string }>();
defineEmits<{ demo: [familyId: string] }>();

const { state } = useDemo();
const demoFamilies = state.families.slice(0, 3);
</script>
