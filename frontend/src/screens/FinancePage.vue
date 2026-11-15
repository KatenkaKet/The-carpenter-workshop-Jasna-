<template>
  <AppShell active="finance">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Оплаты и чеки</p>
        <h1 class="page-title mt-1.5">Финансы</h1>
        <p class="mt-1.5 text-sm text-muted">
          Сколько заплатил каждый клиент и чеки по каждой оплате.
        </p>
      </div>
      <a href="/yasna/subscriptions" class="btn-ghost text-xs">
        <i class="fa-solid fa-ticket" aria-hidden="true"></i>
        Абонементы
      </a>
    </div>

    <div class="mt-5 grid grid-cols-3 gap-2">
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p
          class="text-[10px] font-bold uppercase leading-tight tracking-wide text-muted"
        >
          Оплачено всего
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
          За месяц
        </p>
        <p
          class="mt-1 font-display text-lg font-semibold leading-tight sm:text-2xl"
        >
          {{ formatMoney(monthPaid) }}
        </p>
      </div>
      <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
        <p
          class="text-[10px] font-bold uppercase leading-tight tracking-wide text-muted"
        >
          Без чека
        </p>
        <p
          class="mt-1 font-display text-lg font-semibold leading-tight text-clay sm:text-2xl"
        >
          {{ missingReceipts }}
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
          placeholder="Имя или телефон"
          aria-label="Поиск по финансам"
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

    <div class="mt-3 flex flex-wrap items-center justify-between gap-2">
      <div class="flex flex-wrap gap-1.5">
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
      <p
        class="mt-0.5 basis-full text-right text-[10px] font-semibold text-muted sm:mt-0 sm:basis-auto"
      >
        {{ visibleRows.length }} из {{ rows.length }} клиентов
      </p>
    </div>

    <div v-if="visibleRows.length" class="mt-4 grid gap-3 xl:grid-cols-2">
      <article v-for="row in visibleRows" :key="row.family.id" class="card p-4">
        <div class="flex items-start gap-3">
          <span
            class="grid h-11 w-11 shrink-0 place-items-center rounded-full text-[12px] font-bold"
            :class="avatarTone(row.family.id)"
          >
            {{ initials(row.family.parent) }}
          </span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
              <h2 class="text-sm font-bold">{{ row.family.parent }}</h2>
              <span class="chip bg-paper text-muted">
                {{ row.children.length }}
                {{ plural(row.children.length, "ребёнок", "ребёнка", "детей") }}
              </span>
            </div>
            <p class="mt-0.5 text-[11px] text-muted">{{ row.family.phone }}</p>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <span
                v-for="child in row.children"
                :key="child.id"
                class="chip"
                :class="
                  child.paid > 0
                    ? 'bg-forest-soft text-forest'
                    : 'bg-paper text-muted'
                "
              >
                {{ child.name }}, {{ child.age }} ·
                {{ formatMoney(child.paid) }}
              </span>
            </div>
          </div>
          <div class="hidden shrink-0 text-right sm:block">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
              Оплачено
            </p>
            <p class="mt-0.5 font-display text-2xl font-semibold">
              {{ formatMoney(row.total) }}
            </p>
            <p v-if="row.lastDate" class="mt-0.5 text-[10px] text-muted">
              последняя {{ formatDay(row.lastDate) }}
            </p>
          </div>
        </div>

        <div
          class="mt-3 flex items-end justify-between gap-3 border-t border-line/70 pt-3"
        >
          <div class="min-w-0">
            <p class="font-display text-xl font-semibold sm:hidden">
              {{ formatMoney(row.total) }}
            </p>
            <p class="mt-0.5 text-[10px] font-semibold text-muted">
              {{ row.paymentCount }}
              {{ plural(row.paymentCount, "оплата", "оплаты", "оплат") }} ·
              {{ row.receiptCount }} с чеком
            </p>
            <p
              v-if="row.receiptCount < row.paymentCount"
              class="mt-0.5 text-[10px] font-bold text-honey"
            >
              не хватает чеков: {{ row.paymentCount - row.receiptCount }}
            </p>
          </div>
          <button
            v-if="row.paymentCount"
            type="button"
            class="btn-primary shrink-0 !min-h-[40px] !px-3 text-xs"
            @click="openFamily(row)"
          >
            <i
              class="fa-solid fa-cloud-arrow-up text-[11px]"
              aria-hidden="true"
            ></i>
            {{ actionLabel(row) }}
          </button>
          <span v-else class="chip bg-paper text-muted">оплат пока нет</span>
        </div>
      </article>
    </div>

    <div v-else class="card mt-4 px-5 py-10 text-center">
      <span
        class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
      >
        <i class="fa-solid fa-magnifying-glass" aria-hidden="true"></i>
      </span>
      <p class="mt-3 font-display text-lg font-semibold">Ничего не нашли</p>
      <p class="mt-1 text-xs text-muted">
        Измените запрос или сбросьте фильтр.
      </p>
      <button type="button" class="btn-primary mt-4" @click="resetFilters">
        Показать всех
      </button>
    </div>

    <p class="mt-5 text-center text-[11px] leading-relaxed text-muted/70">
      Чеки хранятся в защищённом файловом хранилище · доступны из карточки
      клиента
    </p>

    <BottomSheet
      :open="sheetOpen"
      title="Чеки оплат"
      :subtitle="activeFamily ? activeFamily.family.parent : ''"
      @close="sheetOpen = false"
    >
      <div v-if="activeFamily" class="space-y-4">
        <div class="grid grid-cols-2 gap-2">
          <div class="rounded-2xl border border-line/80 bg-paper/60 p-3.5">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
              Оплачено
            </p>
            <p class="mt-1 font-display text-xl font-semibold">
              {{ formatMoney(activeFamily.total) }}
            </p>
          </div>
          <div class="rounded-2xl border border-line/80 bg-paper/60 p-3.5">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
              Чеки
            </p>
            <p class="mt-1 font-display text-xl font-semibold">
              {{ activeFamily.receiptCount }} из {{ activeFamily.paymentCount }}
            </p>
          </div>
        </div>

        <p
          v-if="uploadError"
          class="rounded-xl bg-clay-soft px-3 py-2 text-xs font-semibold text-clay"
          role="alert"
        >
          {{ uploadError }}
        </p>

        <div class="space-y-2">
          <h3 class="text-sm font-bold">Оплаты клиента</h3>
          <div
            v-for="item in activeFamily.payments"
            :key="item.key"
            class="rounded-2xl border border-line/80 bg-card p-3"
          >
            <div class="flex items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="text-sm font-bold">
                  {{ formatMoney(item.payment.amount) }}
                </p>
                <p class="mt-0.5 text-[11px] text-muted">
                  {{ formatDay(item.payment.date) }} · {{ item.child.name }},
                  {{ item.child.age }}
                </p>
                <p class="mt-0.5 text-[10px] text-muted">
                  {{ item.payment.method }} ·
                  {{ item.payment.lessons }}
                  {{
                    plural(
                      item.payment.lessons,
                      "занятие",
                      "занятия",
                      "занятий",
                    )
                  }}
                </p>
              </div>
              <span
                class="chip shrink-0"
                :class="
                  item.payment.receiptHash
                    ? 'bg-forest-soft text-forest'
                    : 'bg-honey-soft text-honey'
                "
              >
                {{ item.payment.receiptHash ? "чек" : "без чека" }}
              </span>
            </div>

            <div
              v-if="item.payment.receiptHash"
              class="mt-2 rounded-xl bg-paper/70 p-2.5"
            >
              <div class="flex items-center gap-2">
                <i
                  class="fa-solid fa-file-pdf text-clay"
                  aria-hidden="true"
                ></i>
                <span class="min-w-0 flex-1 truncate text-[11px] font-semibold">
                  {{ item.payment.receiptName || "Чек оплаты" }}
                </span>
              </div>
              <div class="mt-2 grid grid-cols-3 gap-1.5">
                <a
                  :href="receiptUrl(item.payment)"
                  target="_blank"
                  rel="noopener"
                  class="btn-ghost !min-h-[34px] !px-2 text-[10px]"
                >
                  Открыть
                </a>
                <a
                  :href="receiptUrl(item.payment, true)"
                  target="_blank"
                  rel="noopener"
                  class="btn-ghost !min-h-[34px] !px-2 text-[10px]"
                >
                  Скачать
                </a>
                <button
                  type="button"
                  class="btn-quiet !min-h-[34px] border border-line !px-2 text-[10px]"
                  :disabled="uploadingKey !== null"
                  @click="chooseFile(item)"
                >
                  Заменить
                </button>
              </div>
            </div>

            <button
              v-else
              type="button"
              class="btn-ghost mt-2 w-full !min-h-[40px] text-xs"
              :disabled="uploadingKey !== null"
              @click="chooseFile(item)"
            >
              <i
                :class="
                  uploadingKey === item.key
                    ? 'fa-solid fa-spinner fa-spin'
                    : 'fa-solid fa-cloud-arrow-up'
                "
                aria-hidden="true"
              ></i>
              {{
                uploadingKey === item.key ? "Загружаем чек…" : "Загрузить чек"
              }}
            </button>
          </div>
        </div>

        <p
          class="rounded-xl bg-paper/80 px-3 py-2.5 text-[10px] leading-relaxed text-muted"
        >
          PDF, JPG или PNG до 10 МБ. Чек прикрепляется к конкретной оплате, его
          можно открыть и скачать. Файл хранится в защищённом хранилище.
        </p>
      </div>
    </BottomSheet>

    <input
      ref="fileInput"
      type="file"
      class="hidden"
      accept="application/pdf,image/jpeg,image/png"
      @change="onFileChange"
    />

    <AppToast :message="toastMessage" />
    <InvoicesPanel />
  <PaymentCorrections/>
</AppShell>
</template>

<script setup lang="ts">
import PaymentCorrections from "../components/PaymentCorrections.vue";
import InvoicesPanel from "../components/InvoicesPanel.vue";
import { computed, ref } from "vue";

import AppShell from "../components/AppShell.vue";
import AppToast from "../components/AppToast.vue";
import BottomSheet from "../components/BottomSheet.vue";
import {
  DEMO_TODAY,
  avatarTone,
  formatDay,
  formatMoney,
  initials,
  plural,
  type Child,
  type Family,
  type Payment,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const { state, childById, subscriptionFor, attachReceipt } = useDemo();
const { message: toastMessage, show: notify } = useToast();

interface FinancePayment {
  key: string;
  child: Child;
  payment: Payment;
}

interface FinanceChild extends Child {
  paid: number;
}

interface FinanceRow {
  family: Family;
  children: FinanceChild[];
  payments: FinancePayment[];
  total: number;
  paymentCount: number;
  receiptCount: number;
  lastDate: string;
}

const rows = computed<FinanceRow[]>(() => {
  const result: FinanceRow[] = [];
  for (const family of state.families) {
    const payments: FinancePayment[] = [];
    const children: FinanceChild[] = [];

    for (const id of family.childIds) {
      const child = childById(id);
      if (!child) continue;
      let paid = 0;
      for (const payment of subscriptionFor(id)?.payments ?? []) {
        paid += payment.amount;
        payments.push({ key: `${id}:${payment.id}`, child, payment });
      }
      children.push({ ...child, paid });
    }

    payments.sort((a, b) => b.payment.date.localeCompare(a.payment.date));
    result.push({
      family,
      children,
      payments,
      total: payments.reduce((sum, item) => sum + item.payment.amount, 0),
      paymentCount: payments.length,
      receiptCount: payments.filter((item) => Boolean(item.payment.receiptHash))
        .length,
      lastDate: payments[0]?.payment.date ?? "",
    });
  }
  return result.sort((a, b) => b.total - a.total);
});

const allPayments = computed(() => rows.value.flatMap((row) => row.payments));
const totalPaid = computed(() =>
  allPayments.value.reduce((sum, item) => sum + item.payment.amount, 0),
);
const monthPaid = computed(() =>
  allPayments.value
    .filter((item) => item.payment.date.startsWith(DEMO_TODAY.slice(0, 7)))
    .reduce((sum, item) => sum + item.payment.amount, 0),
);
const missingReceipts = computed(
  () => allPayments.value.filter((item) => !item.payment.receiptHash).length,
);

const search = ref("");
const activeFilter = ref<"all" | "missing" | "with">("all");

const filters = [
  { key: "all", label: "Все клиенты" },
  { key: "missing", label: "Без чека" },
  { key: "with", label: "С чеком" },
] as const;

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
      (activeFilter.value === "missing" &&
        row.receiptCount < row.paymentCount) ||
      (activeFilter.value === "with" && row.receiptCount > 0);

    return matchesQuery && matchesFilter;
  });
});

function actionLabel(row: FinanceRow): string {
  return row.receiptCount < row.paymentCount ? "Загрузить чек" : "Чеки оплат";
}

function resetFilters() {
  search.value = "";
  activeFilter.value = "all";
}

const sheetOpen = ref(false);
const activeFamilyId = ref("");
const activeFamily = computed(() =>
  rows.value.find((row) => row.family.id === activeFamilyId.value),
);

function openFamily(row: FinanceRow) {
  activeFamilyId.value = row.family.id;
  uploadError.value = "";
  sheetOpen.value = true;
}

const fileInput = ref<HTMLInputElement | null>(null);
const pending = ref<{ key: string; childId: string; paymentId: string } | null>(
  null,
);
const uploadingKey = ref<string | null>(null);
const uploadError = ref("");

function chooseFile(item: FinancePayment) {
  pending.value = {
    key: item.key,
    childId: item.child.id,
    paymentId: item.payment.id,
  };
  uploadError.value = "";
  if (fileInput.value) {
    fileInput.value.value = "";
    fileInput.value.click();
  }
}

function validate(file: File): string {
  const allowed = /\.(pdf|jpe?g|png)$/i.test(file.name);
  if (!allowed) return "Подойдёт PDF, JPG или PNG.";
  if (file.size > 10 * 1024 * 1024)
    return "Файл больше 10 МБ. Выберите чек поменьше.";
  return "";
}

async function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  const target = pending.value;
  if (!file || !target) {
    input.value = "";
    return;
  }

  uploadError.value = "";
  const invalid = validate(file);
  if (invalid) {
    uploadError.value = invalid;
    input.value = "";
    pending.value = null;
    return;
  }

  uploadingKey.value = target.key;
  try {
    const saved = await attachReceipt(target.childId, target.paymentId, file);
    if (!saved) throw new Error("Payment not found");
    notify("Чек прикреплён к оплате");
  } catch {
    uploadError.value =
      "Не удалось загрузить чек. Проверьте файл и попробуйте ещё раз.";
  } finally {
    uploadingKey.value = null;
    pending.value = null;
    input.value = "";
  }
}

function receiptUrl(payment: Payment, _download = false): string {
  return (
    payment.receiptUrl ||
    (payment.receiptHash
      ? `/api/v1/receipts/${payment.receiptHash}/download`
      : "#")
  );
}
</script>
