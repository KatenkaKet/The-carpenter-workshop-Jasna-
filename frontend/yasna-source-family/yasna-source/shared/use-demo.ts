// @shared
// Demo adapter for endpoint.md v1.1: same entities and scenarios, no HTTP backend.
// Method names map to the contract; see the table in .CHATIUM-LLM.md.
import { onMounted, reactive, ref } from "vue";
import {
  DEMO_NOW_MINUTES,
  DEMO_TODAY,
  addDays,
  formatDay,
  createSeed,
  currentMoment,
  currentTimeLabel,
  formatMoney,
  markKey,
  markLabel,
  timeToMinutes,
  type AppNotification,
  type Child,
  type DemoState,
  type Family,
  type Invoice,
  type Lesson,
  type Mark,
  type MarkStatus,
  type ParentBooking,
  type ParentBookingStatus,
  type Payment,
  type Subscription,
  type Tariff,
} from "./demo";

const STORAGE_KEY = "yasna.demo.v7";

let browserState: DemoState | null = null;
let browserHydrated = false;
let storageBound = false;

export type EnrollResult = "ok" | "full" | "duplicate" | "missing";

export type BookingError =
  | "NOT_FOUND"
  | "FORBIDDEN"
  | "LESSON_UNAVAILABLE"
  | "DUPLICATE_BOOKING"
  | "SEAT_LIMIT_EXCEEDED"
  | "CHILD_TIME_CONFLICT"
  | "PAYMENT_REQUIRED";

export type BookingResult =
  | { ok: true; booking: ParentBooking }
  | { ok: false; code: BookingError; message: string };

export type PayResult =
  | { ok: true; invoice: Invoice; payment: Payment }
  | { ok: false; code: BookingError; message: string };

export function useDemo() {
  const isBrowser = typeof window !== "undefined";
  const state = reactive<DemoState>(
    isBrowser ? (browserState ??= createSeed()) : createSeed(),
  );
  const ready = ref(false);

  onMounted(() => {
    if (!isBrowser) return;
    if (!browserHydrated) {
      browserHydrated = true;
      try {
        const raw = window.localStorage.getItem(STORAGE_KEY);
        if (raw) {
          const saved = JSON.parse(raw) as DemoState;
          if (saved && saved.version === 7 && Array.isArray(saved.families)) {
            Object.assign(state, saved);
          }
        }
      } catch {
        /* демо-хранилище может быть недоступно */
      }
      if (!storageBound) {
        storageBound = true;
        window.addEventListener("storage", (event: StorageEvent) => {
          if (event.key !== STORAGE_KEY || !event.newValue) return;
          try {
            const saved = JSON.parse(event.newValue) as DemoState;
            if (saved && saved.version === 7) Object.assign(state, saved);
          } catch {
            /* ignore malformed state */
          }
        });
      }
    }
    ready.value = true;
  });

  function persist() {
    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      /* приватный режим браузера — просто живём без сохранения */
    }
  }

  function familyById(id: string): Family | undefined {
    return state.families.find((family) => family.id === id);
  }

  function childById(id: string): Child | undefined {
    return state.children.find((child) => child.id === id);
  }

  function subscriptionFor(childId: string): Subscription | undefined {
    return state.subscriptions[childId];
  }

  function familyForChild(child: Child): Family | undefined {
    return familyById(child.familyId);
  }

  function lessonsFor(date: string): Lesson[] {
    return state.lessons
      .filter((item) => item.date === date)
      .slice()
      .sort((a, b) => a.time.localeCompare(b.time));
  }

  function lessonById(id: string): Lesson | undefined {
    return state.lessons.find((item) => item.id === id);
  }

  function markFor(lessonId: string, childId: string): Mark | undefined {
    return state.marks[markKey(lessonId, childId)];
  }

  function addLesson(input: {
    date: string;
    time: string;
    duration: number;
    kind: Lesson["kind"];
    title: string;
    capacity: number;
    childIds: string[];
  }): Lesson {
    const created: Lesson = {
      ...input,
      id: `l_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`,
    };
    state.lessons.push(created);
    persist();
    return created;
  }

  function enroll(lessonId: string, childId: string): EnrollResult {
    const target = lessonById(lessonId);
    if (!target || !childById(childId)) return "missing";
    if (target.childIds.includes(childId)) return "duplicate";
    if (target.childIds.length >= target.capacity) return "full";
    target.childIds.push(childId);
    persist();
    return "ok";
  }

  function setMark(
    lessonId: string,
    childId: string,
    status: MarkStatus,
    grade?: number,
  ): { remaining: number; delta: number } {
    const key = markKey(lessonId, childId);
    const previous = state.marks[key];
    const subscription = state.subscriptions[childId];
    const shouldCharge =
      status === "present" ||
      (status === "absent" && state.settings.rules.chargeNoShow);

    const singleVisit = state.bookings.some(
      (booking) =>
        booking.lessonId === lessonId &&
        booking.childId === childId &&
        booking.paidWith === "single" &&
        (booking.status === "confirmed" || booking.status === "pending"),
    );

    let charged = previous?.charged ?? false;
    let delta = 0;

    if (singleVisit) {
      charged = true;
    } else if (subscription) {
      if (shouldCharge && !charged) {
        subscription.remaining = Math.max(0, subscription.remaining - 1);
        charged = true;
        delta = -1;
      } else if (!shouldCharge && charged) {
        subscription.remaining = Math.min(
          subscription.total,
          subscription.remaining + 1,
        );
        charged = false;
        delta = 1;
      }
    }

    const nextGrade =
      status === "present" ? (grade ?? previous?.grade ?? 0) : undefined;
    const history = (previous?.history ?? []).slice();
    const changed =
      !previous ||
      previous.status !== status ||
      (status === "present" && grade !== undefined && previous.grade !== grade);

    if (changed) {
      history.push({
        at: currentTimeLabel(),
        label: markLabel(status, nextGrade),
      });
    }

    state.marks[key] = {
      status,
      grade: nextGrade || undefined,
      charged,
      history,
    };
    persist();

    return { remaining: subscription?.remaining ?? 0, delta };
  }

  function renew(childId: string, tariff: Tariff): void {
    const subscription = state.subscriptions[childId];
    if (!subscription) return;
    subscription.total += tariff.lessons;
    subscription.remaining += tariff.lessons;
    const base =
      subscription.validUntil > DEMO_TODAY
        ? subscription.validUntil
        : DEMO_TODAY;
    subscription.validUntil = addDays(base, tariff.days);
    subscription.payments.unshift({
      id: `p_${Date.now().toString(36)}`,
      date: DEMO_TODAY,
      amount: tariff.price,
      lessons: tariff.lessons,
      method: "Демо-оплата",
    });
    persist();
  }

  // endpoint.md §9: POST /master/payments/:id/receipts.
  function attachReceipt(
    childId: string,
    paymentId: string,
    hash: string,
    name: string,
  ): boolean {
    const subscription = state.subscriptions[childId];
    const payment = subscription?.payments.find(
      (item) => item.id === paymentId,
    );
    if (!payment) return false;
    payment.receiptHash = hash;
    payment.receiptName = name;
    payment.receiptAt = DEMO_TODAY;
    const child = childById(childId);
    if (child) {
      notifyParent(
        child.familyId,
        "Чек прикреплён",
        `${formatMoney(payment.amount)} · ${name}`,
        "good",
      );
    }
    persist();
    return true;
  }

  // endpoint.md §1: POST /auth/access-link/exchange (demo token lookup).
  function familyByToken(token: string): Family | undefined {
    const normalized = token.trim();
    if (!normalized) return undefined;
    return state.families.find((family) => family.accessToken === normalized);
  }

  function bookingsForFamily(familyId: string): ParentBooking[] {
    return state.bookings
      .filter((booking) => booking.familyId === familyId)
      .slice()
      .sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  }

  function invoicesForFamily(familyId: string): Invoice[] {
    return state.invoices
      .filter((invoice) => invoice.familyId === familyId)
      .slice()
      .sort((a, b) =>
        (b.paidAt ?? b.createdAt).localeCompare(a.paidAt ?? a.createdAt),
      );
  }

  function paymentForInvoice(invoice: Invoice): Payment | undefined {
    if (!invoice.paymentId) return undefined;
    return state.subscriptions[invoice.childId]?.payments.find(
      (payment) => payment.id === invoice.paymentId,
    );
  }

  function notificationsFor(
    audience: "master" | "parent",
    familyId?: string,
  ): AppNotification[] {
    if (audience === "parent") {
      return state.notifications.filter(
        (item) => item.audience === "parent" && item.familyId === familyId,
      );
    }
    return state.notifications.filter((item) => item.audience !== "parent");
  }

  function unreadFor(audience: "master" | "parent", familyId?: string): number {
    return notificationsFor(audience, familyId).filter((item) => !item.read)
      .length;
  }

  function markAllReadFor(
    audience: "master" | "parent",
    familyId?: string,
  ): void {
    for (const item of notificationsFor(audience, familyId)) {
      item.read = true;
    }
    persist();
  }

  function notifyParent(
    familyId: string,
    title: string,
    text: string,
    tone: AppNotification["tone"] = "calm",
  ): void {
    state.notifications.unshift({
      id: `n_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`,
      audience: "parent",
      familyId,
      title,
      text,
      time: "Только что",
      tone,
      read: false,
    });
  }

  function notifyMaster(
    title: string,
    text: string,
    tone: AppNotification["tone"] = "calm",
  ): void {
    state.notifications.unshift({
      id: `n_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`,
      audience: "master",
      title,
      text,
      time: "Только что",
      tone,
      read: false,
    });
  }

  // endpoint.md §4: POST /bookings. Status starts as pending (master approval UX).
  function createBooking(
    familyId: string,
    childId: string,
    lessonId: string,
  ): BookingResult {
    const family = familyById(familyId);
    const child = childById(childId);
    const target = lessonById(lessonId);
    if (!family || !child || !target) {
      return { ok: false, code: "NOT_FOUND", message: "Занятие не найдено." };
    }
    if (child.familyId !== family.id) {
      return {
        ok: false,
        code: "FORBIDDEN",
        message: "Ребёнок из другой семьи.",
      };
    }
    if (
      target.date < DEMO_TODAY ||
      (target.date === DEMO_TODAY &&
        timeToMinutes(target.time) <= DEMO_NOW_MINUTES)
    ) {
      return {
        ok: false,
        code: "LESSON_UNAVAILABLE",
        message: "Это занятие уже началось или прошло.",
      };
    }
    const active = state.bookings.find(
      (booking) =>
        booking.lessonId === lessonId &&
        booking.childId === childId &&
        (booking.status === "pending" || booking.status === "confirmed"),
    );
    if (active || target.childIds.includes(childId)) {
      return {
        ok: false,
        code: "DUPLICATE_BOOKING",
        message: "Ребёнок уже записан на это занятие.",
      };
    }
    if (target.childIds.length >= target.capacity) {
      return {
        ok: false,
        code: "SEAT_LIMIT_EXCEEDED",
        message: "Свободных мест больше нет.",
      };
    }

    const start = timeToMinutes(target.time);
    const end = start + target.duration;
    for (const other of state.lessons) {
      if (other.id === target.id || other.date !== target.date) continue;
      if (!other.childIds.includes(childId)) continue;
      const otherStart = timeToMinutes(other.time);
      const otherEnd = otherStart + other.duration;
      if (start < otherEnd && otherStart < end) {
        return {
          ok: false,
          code: "CHILD_TIME_CONFLICT",
          message: "У ребёнка уже есть занятие в это время.",
        };
      }
    }

    const childSubscription = state.subscriptions[childId];
    let paidWith: "subscription" | "single" | undefined;
    if (
      childSubscription &&
      childSubscription.remaining > 0 &&
      childSubscription.validUntil >= target.date
    ) {
      paidWith = "subscription";
    } else if (
      childSubscription &&
      (childSubscription.singleCredits ?? 0) > 0
    ) {
      childSubscription.singleCredits = Math.max(
        0,
        (childSubscription.singleCredits ?? 0) - 1,
      );
      paidWith = "single";
    } else {
      return {
        ok: false,
        code: "PAYMENT_REQUIRED",
        message: "Сначала купите 1 посещение или абонемент.",
      };
    }

    target.childIds.push(childId);
    const booking: ParentBooking = {
      id: `b_${Date.now().toString(36)}${Math.random().toString(36).slice(2, 4)}`,
      familyId: family.id,
      childId,
      lessonId,
      status: "confirmed",
      createdAt: currentMoment(),
      confirmedAt: currentMoment(),
      paidWith,
    };
    state.bookings.unshift(booking);
    notifyParent(
      family.id,
      "Вы записаны",
      `${child.name} · ${formatDay(target.date)}, ${target.time}. Запись активна.`,
      "good",
    );
    notifyMaster(
      "Новая запись",
      `${family.parent} · ${child.name} · ${formatDay(target.date)}, ${target.time}`,
      "calm",
    );
    persist();
    return { ok: true, booking };
  }

  // endpoint.md §7 POST /invoices + §8 POST /payments/intent.
  // Prototype flow: buy a tariff before booking; payment is shown in history.
  function payTariff(
    childId: string,
    tariffId: string,
    options?: { silentParent?: boolean },
  ): PayResult {
    const child = childById(childId);
    const tariff = state.settings.tariffs.find((item) => item.id === tariffId);
    if (!child || !tariff) {
      return { ok: false, code: "NOT_FOUND", message: "Тариф не найден." };
    }
    const kind = tariff.lessons === 1 ? "single_lesson" : "subscription";
    const invoice: Invoice = {
      id: `inv_${Date.now().toString(36)}`,
      familyId: child.familyId,
      childId: child.id,
      kind,
      title: tariff.title,
      amount: tariff.price,
      lessons: tariff.lessons,
      status: "paid",
      createdAt: currentMoment(),
      paidAt: DEMO_TODAY,
    };
    const payment: Payment = {
      id: `p_${Date.now().toString(36)}`,
      date: DEMO_TODAY,
      amount: tariff.price,
      lessons: tariff.lessons,
      method: "СБП · демо",
      invoiceId: invoice.id,
      kind,
    };

    let subscription = state.subscriptions[child.id];
    if (!subscription) {
      subscription = {
        childId: child.id,
        total: 0,
        remaining: 0,
        validUntil: DEMO_TODAY,
        payments: [],
        singleCredits: 0,
      };
      state.subscriptions[child.id] = subscription;
    }

    if (tariff.lessons > 1) {
      subscription.total += tariff.lessons;
      subscription.remaining += tariff.lessons;
      const base =
        subscription.validUntil > DEMO_TODAY
          ? subscription.validUntil
          : DEMO_TODAY;
      subscription.validUntil = addDays(base, tariff.days);
    } else {
      subscription.singleCredits = (subscription.singleCredits ?? 0) + 1;
    }

    subscription.payments.unshift(payment);
    invoice.paymentId = payment.id;
    state.invoices.unshift(invoice);

    const family = familyById(child.familyId);
    if (!options?.silentParent) {
      notifyParent(
        child.familyId,
        "Оплата прошла",
        `${child.name} · ${tariff.title} · ${formatMoney(tariff.price)}. Теперь можно записаться на занятие.`,
        "good",
      );
    }
    notifyMaster(
      "Оплата от семьи",
      `${formatMoney(tariff.price)} · ${child.name} · ${family?.parent ?? ""}`,
      "good",
    );
    persist();
    return { ok: true, invoice, payment };
  }

  function addFamily(
    parent: string,
    phone: string,
    kids: Array<{ name: string; age: number }>,
  ): Family {
    const familyId = `f_${Date.now().toString(36)}`;
    const childIds: string[] = [];
    for (const kid of kids) {
      const childId = `c_${Date.now().toString(36)}${Math.random()
        .toString(36)
        .slice(2, 4)}`;
      state.children.push({
        id: childId,
        name: kid.name,
        age: kid.age,
        familyId,
      });
      state.subscriptions[childId] = {
        childId,
        total: 0,
        remaining: 0,
        validUntil: DEMO_TODAY,
        payments: [],
        singleCredits: 0,
      };
      childIds.push(childId);
    }
    const family: Family = {
      id: familyId,
      parent,
      phone,
      childIds,
      balance: 0,
      accessToken: `yl_${Math.random().toString(36).slice(2)}${Math.random().toString(36).slice(2)}`,
    };
    state.families.push(family);
    persist();
    return family;
  }

  function updateTariff(id: string, patch: Partial<Tariff>): void {
    const tariff = state.settings.tariffs.find((item) => item.id === id);
    if (!tariff) return;
    Object.assign(tariff, patch);
    persist();
  }

  function saveSettings(): void {
    persist();
  }

  function markAllRead(): void {
    markAllReadFor("master");
  }

  function resetDemo(): void {
    Object.assign(state, createSeed());
    if (typeof window !== "undefined") {
      try {
        window.localStorage.removeItem(STORAGE_KEY);
      } catch {
        /* ignore */
      }
    }
  }

  return {
    state,
    ready,
    familyById,
    familyByToken,
    childById,
    subscriptionFor,
    familyForChild,
    lessonsFor,
    lessonById,
    markFor,
    addLesson,
    enroll,
    setMark,
    renew,
    attachReceipt,
    bookingsForFamily,
    createBooking,
    invoicesForFamily,
    paymentForInvoice,
    payTariff,
    notificationsFor,
    unreadFor,
    markAllReadFor,
    addFamily,
    updateTariff,
    saveSettings,
    markAllRead,
    resetDemo,
  };
}
