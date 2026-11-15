<template>
  <AppShell active="subscriptions">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="eyebrow">Оплаты и остатки</p>
        <h1 class="page-title mt-1.5">Абонементы</h1>
        <p class="mt-1.5 text-sm text-muted">
          Занятия и деньги считаются отдельно — здесь только занятия.
        </p>
      </div>

    </div>

    <div
      class="mt-5 inline-flex rounded-full border border-line bg-card p-1"
      role="tablist"
    >
      <button
        type="button"
        role="tab"
        :aria-selected="tab === 'list'"
        class="rounded-full px-4 py-2 text-xs font-bold transition"
        :class="
          tab === 'list' ? 'bg-forest text-card' : 'text-muted hover:text-ink'
        "
        @click="tab = 'list'"
      >
        Абонементы
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="tab === 'rules'"
        class="rounded-full px-4 py-2 text-xs font-bold transition"
        :class="
          tab === 'rules' ? 'bg-forest text-card' : 'text-muted hover:text-ink'
        "
        @click="tab = 'rules'"
      >
        Тарифы и правила
      </button>
    </div>

    <template v-if="tab === 'list'">
      <div class="mt-4 grid grid-cols-3 gap-2">
        <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
            Активных
          </p>
          <p class="mt-0.5 font-display text-2xl font-semibold">
            {{ activeCount }}
          </p>
        </div>
        <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
            Осталось занятий
          </p>
          <p class="mt-0.5 font-display text-2xl font-semibold">
            {{ totalRemaining }}
          </p>
        </div>
        <div class="rounded-2xl border border-line/80 bg-card px-3 py-2.5">
          <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
            Нужно продлить
          </p>
          <p class="mt-0.5 font-display text-2xl font-semibold text-clay">
            {{ attentionCount }}
          </p>
        </div>
      </div>

      <div class="mt-4 flex flex-wrap gap-1.5">
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

      <label class="label mt-3">Сортировка<select v-model="sortBy" class="field"><option value="name">По ФИО</option><option value="status">По состоянию абонемента</option></select></label>
      <div v-if="visibleRows.length" class="mt-4 grid gap-3">
        <article
          v-for="row in visibleRows"
          :key="row.child.id"
          class="card p-4"
        >
          <div class="flex items-start gap-3">
            <span
              class="grid h-10 w-10 shrink-0 place-items-center rounded-full text-[11px] font-bold"
              :class="avatarTone(row.child.id)"
            >
              {{ initials(row.child.name) }}
            </span>
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <h2 class="text-sm font-bold">{{ row.child.name }}</h2>
                <span class="chip" :class="stateMeta[row.state].chip">
                  {{ stateMeta[row.state].label }}
                </span>
              </div>
              <p class="mt-0.5 truncate text-[11px] text-muted">
                {{ row.child.age }} лет · {{ row.family?.parent }}
              </p>
            </div>
            <button
              type="button"
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full text-muted transition hover:bg-paper hover:text-ink"
              aria-label="История абонемента"
              @click="openDetail(row)"
            >
              <i
                class="fa-solid fa-chevron-right text-[10px]"
                aria-hidden="true"
              ></i>
            </button>
          </div>

          <div v-if="row.total" class="mt-3.5">
            <div class="flex items-baseline justify-between">
              <span class="text-[11px] font-semibold text-muted"
                >Доступно для записи</span
              >
              <span class="text-sm font-bold">
                {{ row.remaining }}
                <span class="text-[11px] font-semibold text-muted"
                  >из {{ row.total }}</span
                >
              </span>
            </div>
            <p class="text-xs text-muted mt-1">Зарезервировано: {{row.reserved}} · Использовано: {{row.used}}</p>
            <div class="mt-1.5 h-2 overflow-hidden rounded-full bg-paper">
              <div
                class="h-full rounded-full transition-all"
                :class="stateMeta[row.state].bar"
                :style="{ width: `${progress(row.remaining, row.total)}%` }"
              ></div>
            </div>
            <div
              class="mt-2 flex items-center justify-between text-[10px] font-semibold text-muted"
            >
              <span>действует до {{ formatDay(row.validUntil) }}</span>
              <span>{{ formatMoney(row.price) }}</span>
            </div>
            <p
              v-if="row.family && (row.family.balance ?? 0) !== 0"
              class="mt-1 text-[10px] font-bold"
              :class="
                (row.family.balance ?? 0) > 0 ? 'text-forest' : 'text-clay'
              "
            >
              {{
                row.family.balance > 0
                  ? `Аванс ${formatMoney(row.family.balance)}`
                  : `Долг ${formatMoney(-row.family.balance)}`
              }}
            </p>
          </div>

          <div
            v-else
            class="mt-3.5 rounded-xl bg-paper/80 px-3 py-2.5 text-[11px] leading-relaxed text-muted"
          >
            <template v-if="row.singleCredits">
              <b class="text-ink">{{ row.singleCredits }}</b>
              {{
                plural(
                  row.singleCredits,
                  "разовое посещение",
                  "разовых посещения",
                  "разовых посещений",
                )
              }}
              — можно записаться без абонемента.
            </template>
            <template v-else>

              Абонемент ещё не оформлен. Оформите первый — и занятия начнут
              списываться автоматически.
            </template>
          </div>

          <p
            v-if="row.total && row.singleCredits"
            class="mt-1 text-[10px] font-bold text-honey"
          >
            + {{ row.singleCredits }}
            {{
              plural(
                row.singleCredits,
                "разовое посещение",
                "разовых посещения",
                "разовых посещений",
              )
            }}
          </p>

          <div class="mt-3 flex gap-2">
            <button
              type="button"
              class="flex-1"
              :class="
                row.total
                  ? 'btn-ghost !min-h-[40px] text-xs'
                  : 'btn-primary !min-h-[40px] text-xs'
              "
              @click="openRenew(row)"
            >
              {{ row.total ? "Продлить" : "Оформить" }}
            </button>
            <button
              v-if="row.total"
              type="button"
              class="btn-quiet !min-h-[40px] border border-line !px-3 text-[11px]"
              @click="openDetail(row)"
            >
              История
            </button>
          </div>
        </article>
      </div>

      <div v-else class="card mt-4 px-5 py-10 text-center">
        <span
          class="mx-auto grid h-12 w-12 place-items-center rounded-full bg-paper text-lg text-muted"
        >
          <i class="fa-solid fa-filter-circle-xmark" aria-hidden="true"></i>
        </span>
        <p class="mt-3 font-display text-lg font-semibold">
          По фильтру ничего нет
        </p>
        <button
          type="button"
          class="mt-2 text-xs font-bold text-honey"
          @click="activeFilter = 'all'"
        >
          Показать всех
        </button>
      </div>
    </template>

    <template v-else>
      <TariffCreator/><GroupEditor/>
      <section class="card mt-4 p-4 sm:p-5" aria-labelledby="tariffs-title">
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="eyebrow">Настройка мастером</p>
            <h2 id="tariffs-title" class="section-title mt-1">Тарифы</h2>
          </div>
          <span class="chip bg-paper text-muted">Цены мастерской</span>
        </div>
        <ul class="mt-3 space-y-2">
          <li
            v-for="tariff in state.settings.tariffs"
            :key="tariff.id"
            class="flex items-center gap-3 rounded-2xl border border-line/80 bg-paper/50 p-3"
          >
            <span
              class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-card text-honey"
            >
              <i class="fa-solid fa-ticket" aria-hidden="true"></i>
            </span>
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-bold">{{ tariff.title }}</span>
              <span class="mt-0.5 block text-[11px] text-muted">
                {{ tariff.lessons }}
                {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }} ·
                {{ tariff.days }} дней
              </span>
            </span>
            <span class="shrink-0 text-sm font-bold">{{
              formatMoney(tariff.price)
            }}</span>
            <button
              type="button"
              class="grid h-8 w-8 shrink-0 place-items-center rounded-full border border-line bg-card text-muted transition hover:text-ink"
              aria-label="Изменить тариф"
              @click="openTariff(tariff)"
            >
              <i class="fa-solid fa-pen text-[10px]" aria-hidden="true"></i>
            </button>
          </li>
        </ul>
      </section>

      <section class="card mt-4 p-4 sm:p-5" aria-labelledby="rules-title">
        <label class="label"
          >Группа<select v-model="ruleGroup" class="input">
            <option v-for="g in groups()" :value="g.id" :key="g.id">
              {{ g.title }}
            </option>
          </select></label
        >
        <p class="text-xs">
          Настройки применяются сразу к следующим отменам, переносам и отметкам. Прошлые операции не пересчитываются.
        </p>
        <h2 id="rules-title" class="section-title mt-1">Правила списания</h2>
        <p class="mt-1 text-[11px] leading-relaxed text-muted">
          Эти правила применяются, когда вы отмечаете посещение в журнале.
        </p>

        <div class="mt-4 divide-y divide-line/70">
          <div class="flex items-start justify-between gap-4 py-3">
            <div>
              <p class="text-sm font-bold">
                Списывать пропуск без предупреждения
              </p>
              <p class="mt-0.5 text-[11px] leading-relaxed text-muted">
                Если родитель не отменил занятие заранее, оно считается
                использованным.
              </p>
            </div>
            <button
              type="button"
              role="switch"
              :aria-checked="state.settings.rules.chargeNoShow"
              class="relative mt-0.5 h-6 w-11 shrink-0 rounded-full transition"
              :class="
                state.settings.rules.chargeNoShow ? 'bg-forest' : 'bg-line'
              "
              @click="toggleRule('chargeNoShow')"
            >
              <span
                class="absolute top-0.5 h-5 w-5 rounded-full bg-card shadow-sm transition-all"
                :class="
                  state.settings.rules.chargeNoShow ? 'left-[22px]' : 'left-0.5'
                "
              ></span>
            </button>
          </div>

          <div class="py-3">
            <p class="text-sm font-bold">Бесплатная отмена</p>
            <p class="mt-0.5 text-[11px] leading-relaxed text-muted">
              За сколько часов до занятия отмена не списывает занятие.
            </p>
<label class="label mt-2">Часов до начала<input class="field" v-model.number="state.settings.rules.cancelHours" type="number" min="0" max="8760" step="1"></label><p class="text-xs text-muted">0 — бесплатно вплоть до начала занятия.</p>
          </div>

          <div class="flex items-start justify-between gap-4 py-3">
            <div>
              <p class="text-sm font-bold">Разрешить перенос занятия</p>
              <p class="mt-0.5 text-[11px] leading-relaxed text-muted">
                Ребёнок может отработать пропуск в другой день по согласованию.
              </p>
            </div>
            <button
              type="button"
              role="switch"
              :aria-checked="state.settings.rules.allowTransfer"
              class="relative mt-0.5 h-6 w-11 shrink-0 rounded-full transition"
              :class="
                state.settings.rules.allowTransfer ? 'bg-forest' : 'bg-line'
              "
              @click="toggleRule('allowTransfer')"
            >
              <span
                class="absolute top-0.5 h-5 w-5 rounded-full bg-card shadow-sm transition-all"
                :class="
                  state.settings.rules.allowTransfer
                    ? 'left-[22px]'
                    : 'left-0.5'
                "
              ></span>
            </button>
          </div>
        </div>

        <button
          type="button"
          class="btn-primary mt-3 w-full"
          @click="saveRules"
        >
          <i class="fa-solid fa-check text-xs" aria-hidden="true"></i>
          Сохранить настройки
        </button>
      </section>

      <p
        class="mt-4 rounded-2xl bg-forest-soft/70 px-4 py-3 text-[11px] leading-relaxed text-forest"
      >
        <b>Как это работает:</b> абонемент хранит количество занятий и срок
        действия отдельно от денег. Оплата пополняет денежный баланс и оформляет
        абонемент, а списание уменьшает остаток занятий по правилам выше.
      </p>
    </template>

    <BottomSheet
      :open="renewOpen"
      :title="
        renewRow && renewRow.total ? 'Продлить абонемент' : 'Оформить абонемент'
      "
      :subtitle="
        renewRow ? `${renewRow.child.name}, ${renewRow.child.age} лет` : ''
      "
      @close="renewOpen = false"
    >
      <div v-if="renewRow" class="space-y-4"><SubscriptionBalances :child-id="renewRow.child.id"/>
        <div
          class="flex items-center justify-between rounded-xl bg-paper/80 px-3 py-2.5"
        >
          <span class="text-xs font-semibold text-muted">Сейчас осталось</span>
          <span class="text-sm font-bold"
            >{{ renewRow.remaining }} занятий</span
          >
        </div>

        <div>
          <span class="label">Тариф</span>
          <div class="space-y-1.5">
            <button
              v-for="tariff in state.settings.tariffs"
              :key="tariff.id"
              type="button"
              class="flex w-full items-center gap-3 rounded-xl border p-3 text-left transition"
              :class="
                selectedTariffId === tariff.id
                  ? 'border-forest bg-forest-soft/50'
                  : 'border-line bg-card hover:border-forest/30'
              "
              @click="selectedTariffId = tariff.id"
            >
              <span class="min-w-0 flex-1">
                <span class="block text-sm font-bold">{{ tariff.title }}</span>
                <span class="block text-[11px] text-muted">
                  {{ tariff.lessons }}
                  {{ plural(tariff.lessons, "занятие", "занятия", "занятий") }}
                  · {{ tariff.days }} дней
                </span>
              </span>
              <span class="shrink-0 text-sm font-bold">{{
                formatMoney(tariff.price)
              }}</span>
              <i
                v-if="selectedTariffId === tariff.id"
                class="fa-solid fa-circle-check text-forest"
                aria-hidden="true"
              ></i>
            </button>
          </div>
        </div>

<label class="label">Способ оплаты<select class="field" v-model="renewMethod"><option value="cash">Наличные</option><option value="transfer">Перевод</option></select></label><label class="label">Чек (необязательно)<input type="file" accept="application/pdf,image/jpeg,image/png" @change="renewFile=($event.target as HTMLInputElement).files?.[0]"></label>
        <button
          v-if="selectedTariff"
          type="button"
          class="btn-primary w-full"
          @click="payAndRenew"
          :disabled="renewBusy"
        >
          <i class="fa-solid fa-lock text-[11px]" aria-hidden="true"></i>
          Сохранить оплату {{ formatMoney(selectedTariff.price) }}
        </button>
        <p class="text-center text-[10px] leading-relaxed text-muted">
          Подтвердите только уже полученную оплату. Оплата будет сохранена в
          базе, а занятия начислены ребёнку.
        </p>
      </div>
    </BottomSheet>

    <BottomSheet
      :open="detailOpen"
      title="Абонемент"
      :subtitle="
        detailRow ? `${detailRow.child.name} · ${detailRow.family?.parent}` : ''
      "
      @close="detailOpen = false"
    >
      <div v-if="detailRow" class="space-y-4">
        <div class="grid grid-cols-2 gap-2">
          <div class="rounded-2xl border border-line/80 bg-paper/60 p-3.5">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
              Остаток занятий
            </p>
            <p class="mt-1 font-display text-2xl font-semibold">
              {{ detailRow.remaining }}
              <span class="font-sans text-xs text-muted"
                >из {{ detailRow.total }}</span
              >
            </p>
            <p class="mt-0.5 text-[10px] text-muted">
              до {{ formatDay(detailRow.validUntil) }}
            </p>
          </div>
          <div class="rounded-2xl border border-line/80 bg-paper/60 p-3.5">
            <p class="text-[10px] font-bold uppercase tracking-wide text-muted">
              Денежный баланс
            </p>
            <p
              class="mt-1 font-display text-2xl font-semibold"
              :class="
                (detailRow.family?.balance ?? 0) < 0
                  ? 'text-clay'
                  : (detailRow.family?.balance ?? 0) > 0
                    ? 'text-forest'
                    : 'text-ink'
              "
            >
              {{ formatMoney(detailRow.family?.balance ?? 0) }}
            </p>
            <p class="mt-0.5 text-[10px] text-muted">
              {{
                (detailRow.family?.balance ?? 0) < 0
                  ? "нужно доплатить"
                  : (detailRow.family?.balance ?? 0) > 0
                    ? "аванс семьи"
                    : "без задолженности"
              }}
            </p>
          </div>
        </div>

        <div>
          <h3 class="text-sm font-bold">История платежей</h3>
          <ul v-if="detailRow.payments.length" class="mt-2 space-y-1.5">
            <li
              v-for="payment in detailRow.payments"
              :key="payment.id"
              class="flex items-center gap-3 rounded-xl border border-line/70 bg-card px-3 py-2.5"
            >
              <span
                class="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-forest-soft text-forest"
              >
                <i class="fa-solid fa-check text-[10px]" aria-hidden="true"></i>
              </span>
              <span class="min-w-0 flex-1">
                <span class="block text-xs font-bold">{{
                  formatMoney(payment.amount)
                }}</span>
                <span class="block text-[10px] text-muted">
                  {{ formatDay(payment.date) }} · {{ payment.lessons }}
                  {{ plural(payment.lessons, "занятие", "занятия", "занятий") }}
                  · {{ payment.method }}
                </span>
              </span>
              <span class="chip bg-forest-soft text-forest">оплачен</span>
            </li>
          </ul>
          <p
            v-else
            class="mt-2 rounded-xl bg-paper/80 px-3 py-3 text-[11px] text-muted"
          >
            Платежей пока нет.
          </p>
        </div>

        <button
          type="button"
          class="btn-primary w-full"
          @click="renewFromDetail"
        >
          Продлить абонемент
        </button>
      </div>
    </BottomSheet>

    <BottomSheet
      :open="tariffOpen"
      title="Изменить тариф"
      subtitle="Цена и срок действуют для новых оплат"
      @close="tariffOpen = false"
    >
      <div class="space-y-3">
        <div>
          <label class="label" for="tariff-title">Название</label>
          <input
            id="tariff-title"
            v-model="tariffForm.title"
            class="field"
            maxlength="50"
          />
        </div>
        <div class="grid grid-cols-2 gap-2">
          <div>
            <label class="label" for="tariff-lessons">Занятий</label>
            <input
              id="tariff-lessons"
              v-model.number="tariffForm.lessons"
              type="number"
              min="1"
              max="99"
              class="field"
            />
          </div>
          <div>
            <label class="label" for="tariff-days">Срок, дней</label>
            <input
              id="tariff-days"
              v-model.number="tariffForm.days"
              type="number"
              min="1"
              max="365"
              class="field"
            />
          </div>
        </div>
        <div>
          <label class="label" for="tariff-price">Цена, ₽</label>
          <input
            id="tariff-price"
            v-model.number="tariffForm.price"
            type="number"
            min="0"
            max="100000"
            class="field"
          />
        </div>
        <p
          v-if="tariffError"
          class="rounded-xl bg-clay-soft px-3 py-2 text-xs font-semibold text-clay"
          role="alert"
        >
          {{ tariffError }}
        </p>
        <button type="button" class="btn-primary w-full" @click="saveTariff">
          Сохранить тариф
        </button>
      </div>
    </BottomSheet>

    <AppToast :message="toastMessage" />
  </AppShell>
</template>

<script setup lang="ts">
import TariffCreator from "../components/TariffCreator.vue";
import GroupEditor from "../components/GroupEditor.vue";
import SubscriptionBalances from "../components/SubscriptionBalances.vue";
import { computed, reactive, ref } from "vue";
import AppShell from "../components/AppShell.vue";
import AppToast from "../components/AppToast.vue";
import BottomSheet from "../components/BottomSheet.vue";
import {
  DEMO_TODAY,
  addDays,
  avatarTone,
  formatDay,
  formatMoney,
  initials,
  plural,
  type Child,
  type Family,
  type Payment,
  type Tariff,
} from "../shared/demo";
import { useDemo } from "../shared/use-demo";
import { useToast } from "../shared/use-toast";

const props = withDefaults(defineProps<{ tab?: string }>(), { tab: "list" });

const {
  state,
  familyForChild,
  subscriptionFor,
  renew,
  updateTariff,
  saveSettings,
  ruleGroup,
  groups,serverSubscriptions,
} = useDemo();
const { message: toastMessage, show: notify } = useToast();

const tab = ref<"list" | "rules">(props.tab === "rules" ? "rules" : "list");
const activeFilter = ref<"all" | "attention" | "empty">("all");

type SubState = "active" | "low" | "ended" | "expired" | "empty" | "suspended";

interface SubRow {
  child: Child;
  family: Family | undefined;
  total: number;
  remaining: number;
  singleCredits: number;
  reserved: number;
  used: number;
  validUntil: string;
  payments: Payment[];
  price: number;
  state: SubState;
}

const rows = computed<SubRow[]>(() =>
  state.children.map((child) => {
    const sub = subscriptionFor(child.id);
    const total = sub?.total ?? 0;
    const remaining = sub?.remaining ?? 0;
    const validUntil = sub?.validUntil ?? DEMO_TODAY;
    const singleCredits = sub?.singleCredits ?? 0;
    let subState: SubState = "active";
    if (total === 0) subState = "empty";
    else if (validUntil < DEMO_TODAY) subState = "expired";
    else if (serverSubscriptions.value.filter(s=>s.child_id===child.id).every(s=>s.status==='suspended')) subState="suspended";
    else if (remaining <= 0 && !(sub?.reserved)) subState = "ended";
    else if (remaining <= 1 || validUntil<=addDays(DEMO_TODAY,3)) subState = "low";
    return {
      child,
      family: familyForChild(child),
      total,
      remaining,
      singleCredits,
      reserved:sub?.reserved||0,used:sub?.used||0,
      validUntil,
      payments: sub?.payments ?? [],
      price: sub?.payments.at(-1)?.amount||0,
      state: subState,
    };
  }),
);

const sortBy=ref("name");
const visibleRows = computed(() => {
  const sorted=[...rows.value].sort((a,b)=>{const rank={active:0,low:1,ended:2,expired:2,suspended:2,empty:3};return (sortBy.value==="status"?rank[a.state]-rank[b.state]:0)||a.child.name.localeCompare(b.child.name,"ru")});
  if (activeFilter.value === "attention") {
    return sorted.filter(
      (row) => row.state !== "active" && row.state !== "empty",
    );
  }
  if (activeFilter.value === "empty") {
    return sorted.filter((row) => row.state === "empty");
  }
  return sorted;
});

const activeCount = computed(
  () =>
    rows.value.filter(
      (row) => (row.remaining+row.reserved) > 0 && row.state!=="suspended" && row.validUntil >= DEMO_TODAY,
    ).length,
);
const totalRemaining = computed(() =>
  rows.value.reduce((sum, row) => sum + row.remaining, 0),
);
const attentionCount = computed(
  () =>
    rows.value.filter(
      (row) =>
        row.state === "low" || row.state === "ended" || row.state === "expired",
    ).length,
);

const filters = [
  { key: "all", label: "Все" },
  { key: "attention", label: "Заканчиваются" },
  { key: "empty", label: "Без абонемента" },
] as const;

const stateMeta: Record<
  SubState,
  { label: string; chip: string; bar: string }
> = {
  suspended:{label:"Ожидает оплаты",chip:"bg-clay-soft text-clay",bar:"bg-clay"},
  active: {
    label: "Активен",
    chip: "bg-forest-soft text-forest",
    bar: "bg-forest",
  },
  low: {
    label: "Заканчивается",
    chip: "bg-honey-soft text-honey",
    bar: "bg-honey",
  },
  ended: {
    label: "Закончился",
    chip: "bg-clay-soft text-clay",
    bar: "bg-clay",
  },
  expired: { label: "Истёк", chip: "bg-clay-soft text-clay", bar: "bg-clay" },
  empty: {
    label: "Без абонемента",
    chip: "bg-paper text-muted",
    bar: "bg-line",
  },
};

function progress(remaining: number, total: number): number {
  if (!total) return 0;
  return Math.max(0, Math.min(100, Math.round((remaining / total) * 100)));
}

const renewOpen = ref(false);
const renewBusy = ref(false);
const renewMethod=ref("cash"),renewFile=ref<File>();
const renewRow = ref<SubRow | null>(null);
const selectedTariffId = ref("t2");

const selectedTariff = computed(() =>
  state.settings.tariffs.find((tariff) => tariff.id === selectedTariffId.value),
);

function openRenew(row: SubRow) {
  renewRow.value = row;renewFile.value=undefined;
  selectedTariffId.value = state.settings.tariffs[0]?.id ?? "";
  renewOpen.value = true;
}

async function payAndRenew() {
  const tariff = selectedTariff.value;
  const row = renewRow.value;
  if (!tariff || !row) return;
  if (renewBusy.value) return;
  renewBusy.value = true;
  try {
    await renew(row.child.id, tariff,renewMethod.value,renewFile.value);
  } finally {
    renewBusy.value = false;
  }
  renewOpen.value = false;
  notify(`Абонемент продлён · оплата сохранена ${formatMoney(tariff.price)}`);
}

const detailOpen = ref(false);
const detailRow = ref<SubRow | null>(null);

function openDetail(row: SubRow) {
  detailRow.value = row;
  detailOpen.value = true;
}

function renewFromDetail() {
  if (!detailRow.value) return;
  const row = detailRow.value;
  detailOpen.value = false;
  openRenew(row);
}

const tariffOpen = ref(false);
const tariffError = ref("");
const tariffForm = reactive({
  id: "",
  title: "",
  lessons: 4,
  price: 3200,
  days: 60,
});

function openTariff(tariff: Tariff) {
  tariffForm.id = tariff.id;
  tariffForm.title = tariff.title;
  tariffForm.lessons = tariff.lessons;
  tariffForm.price = tariff.price;
  tariffForm.days = tariff.days;
  tariffError.value = "";
  tariffOpen.value = true;
}

async function saveTariff() {
  tariffError.value = "";
  if (!tariffForm.title.trim()) {
    tariffError.value = "Укажите название тарифа.";
    return;
  }
  if (tariffForm.lessons < 1 || tariffForm.days < 1 || tariffForm.price < 0) {
    tariffError.value = "Проверьте количество занятий, срок и цену.";
    return;
  }
  await updateTariff(tariffForm.id, {
    title: tariffForm.title.trim(),
    lessons: tariffForm.lessons,
    price: tariffForm.price,
    days: tariffForm.days,
  });
  tariffOpen.value = false;
  notify("Тариф обновлён");
}

async function toggleRule(key: "chargeNoShow" | "allowTransfer") {
  state.settings.rules[key] = !state.settings.rules[key];
}

async function setCancelHours(hours: number) {
  state.settings.rules.cancelHours = hours;
}

async function setMaxTransfers(count: number) {
  state.settings.rules.maxTransfers = count;
}

async function saveRules() {
  await saveSettings();
  notify("Настройки сохранены и применены");
}
</script>
