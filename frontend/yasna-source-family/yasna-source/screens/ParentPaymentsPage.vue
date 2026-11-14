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
    active="payments"
    @logout="logout"
  >
    <div>
      <p class="eyebrow">Платежи и чеки</p>
      <h1 class="page-title mt-1.5">История оплат</h1>
      <p class="mt-1.5 text-sm text-muted">
        Оплаченные занятия и абонементы. Чек открывается, если мастер его
        прикрепил.
      </p>
    </div>

    <div class="mt-4 grid grid-cols-3 gap-2">
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p
          class="text-[10px] font-bold uppercase leading-tight tracking-wide text-muted"
        >
          Оплачено
        </p>
        <p
          class="mt-1 font-display text-lg font-semibold leading-tight sm:text-2xl"
        >
          {{ formatMoney(totalPaid) }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p
          class="text-[10px] font-bold uppercase leading-tight tracking-wide text-muted"
        >
          Платежей
        </p>
        <p
          class="mt-1 font-display text-lg font-semibold leading-tight sm:text-2xl"
        >
          {{ invoices.length }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p
          class="text-[10px] font-bold uppercase leading-tight tracking-wide text-muted"
        >
          С чеком
        </p>
        <p
          class="mt-1 font-display text-lg font-semibold leading-tight text-forest sm:text-2xl"
        >
          {{ receiptCount }}
        </p>
      </div>
    </div>

    <div v-if="invoices.length" class="mt-4 space-y-3">
      <article v-for="invoice in invoices" :key="invoice.id" class="card p-4">
        <div class="flex items-start gap-3">
          <span
            class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-honey-soft text-honey"
          >
            <i class="fa-solid fa-receipt" aria-hidden="true"></i>
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-start justify-between gap-2">
              <div class="min-w-0">
                <h2 class="text-sm font-bold leading-snug">
                  {{ invoice.title }}
                </h2>
                <p class="mt-0.5 text-[11px] text-muted">
                  {{ childName(invoice.childId) }} · оплачено
                  {{ formatDay(invoice.paidAt ?? invoice.createdAt) }}
                </p>
              </div>
              <p class="shrink-0 font-display text-lg font-semibold">
                {{ formatMoney(invoice.amount) }}
              </p>
            </div>
            <p class="mt-0.5 text-[10px] text-muted">
              {{ invoice.lessons }}
              {{ plural(invoice.lessons, "занятие", "занятия", "занятий") }} ·
              {{ paymentFor(invoice)?.method ?? "оплата" }}
            </p>
          </div>
        </div>

        <div
          v-if="paymentFor(invoice)?.receiptHash"
          class="mt-3 rounded-xl bg-paper/75 p-2.5"
        >
          <div class="flex items-center gap-2">
            <i class="fa-solid fa-file-pdf text-clay" aria-hidden="true"></i>
            <span class="min-w-0 flex-1 truncate text-[11px] font-semibold">
              {{ paymentFor(invoice)?.receiptName || "Чек оплаты" }}
            </span>
            <span class="chip bg-forest-soft text-forest">чек</span>
          </div>
          <div class="mt-2 grid grid-cols-2 gap-1.5">
            <a
              :href="receiptUrl(paymentFor(invoice)!)"
              target="_blank"
              rel="noopener"
              class="btn-ghost !min-h-[34px] !px-2 text-[10px]"
            >
              Открыть
            </a>
            <a
              :href="receiptUrl(paymentFor(invoice)!, true)"
              target="_blank"
              rel="noopener"
              class="btn-ghost !min-h-[34px] !px-2 text-[10px]"
            >
              Скачать
            </a>
          </div>
        </div>
        <p
          v-else
          class="mt-3 rounded-xl bg-paper/70 px-3 py-2 text-[10px] leading-relaxed text-muted"
        >
          Оплата прошла. Чек ещё не прикреплён мастером — он появится здесь
          после загрузки.
        </p>
      </article>
    </div>

    <div v-else class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-receipt" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">Оплат пока нет</p>
      <p class="mx-auto mt-1 max-w-xs text-xs leading-relaxed text-muted">
        После оплаты занятия или абонемента платёж появится здесь.
      </p>
      <a href="/yasna/parent" class="btn-primary mt-4">Выбрать занятие</a>
    </div>

    <p class="mt-5 text-center text-[10px] leading-relaxed text-muted/70">
      Чеки хранятся в защищённом хранилище мастерской
    </p>
  </ParentShell>
</template>

<script setup lang="ts">
import { computed } from "vue";
import ParentAccess from "../components/ParentAccess.vue";
import ParentShell from "../components/ParentShell.vue";
import {
  formatDay,
  formatMoney,
  plural,
  type Invoice,
  type Payment,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useParent } from "../shared/use-parent";

const { invoicesForFamily, paymentForInvoice, childById } = useDemo();
const { family, ready, error, loginAs, logout } = useParent();

const invoices = computed<Invoice[]>(() =>
  family.value ? invoicesForFamily(family.value.id) : [],
);
const totalPaid = computed(() =>
  invoices.value.reduce((sum, invoice) => sum + invoice.amount, 0),
);
const receiptCount = computed(
  () =>
    invoices.value.filter((invoice) =>
      Boolean(paymentForInvoice(invoice)?.receiptHash),
    ).length,
);

function childName(id: string): string {
  return childById(id)?.name ?? "Ребёнок";
}

function paymentFor(invoice: Invoice): Payment | undefined {
  return paymentForInvoice(invoice);
}

function receiptUrl(payment: Payment, download = false): string {
  if (!payment.receiptHash) return "#";
  const params = new URLSearchParams({ hash: payment.receiptHash });
  if (download) params.set("download", "1");
  return `/yasna/receipt?${params.toString()}`;
}
</script>
