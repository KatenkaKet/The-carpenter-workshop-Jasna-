// @shared
import { onMounted, reactive, ref } from "vue";
import {
  DEMO_TODAY,
  addDays,
  createSeed,
  currentTimeLabel,
  markKey,
  markLabel,
  type Child,
  type DemoState,
  type Family,
  type Lesson,
  type Mark,
  type MarkStatus,
  type Subscription,
  type Tariff,
} from "./demo";

const STORAGE_KEY = "yasna.demo.v2";

export type EnrollResult = "ok" | "full" | "duplicate" | "missing";

export function useDemo() {
  const state = reactive<DemoState>(createSeed());
  const ready = ref(false);

  onMounted(() => {
    try {
      const raw = window.localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const saved = JSON.parse(raw) as DemoState;
        if (saved && saved.version === 2 && Array.isArray(saved.families)) {
          Object.assign(state, saved);
        }
      }
    } catch {
      /* демо-хранилище может быть недоступно */
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

    let charged = previous?.charged ?? false;
    let delta = 0;

    if (subscription) {
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
    persist();
    return true;
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
      };
      childIds.push(childId);
    }
    const family: Family = {
      id: familyId,
      parent,
      phone,
      childIds,
      balance: 0,
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
    for (const item of state.notifications) item.read = true;
    persist();
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
    addFamily,
    updateTariff,
    saveSettings,
    markAllRead,
    resetDemo,
  };
}
