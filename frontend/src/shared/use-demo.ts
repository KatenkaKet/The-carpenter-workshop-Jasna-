// API-backed compatibility layer for the original Vue screens. No browser database.
import { reactive, ref, watch } from "vue";
import { all, api, key, user } from "./api";
import {
  DEMO_TODAY,
  markKey,
  type DemoState,
  type Family,
  type Lesson,
  type MarkStatus,
  type Tariff,
} from "./demo";
const state = reactive<DemoState>({
  version: 1,
  families: [],
  children: [],
  lessons: [],
  subscriptions: {},
  marks: {},
  settings: {
    tariffs: [],
    rules: {
      chargeNoShow: false,
      cancelHours: 24,
      allowTransfer: true,
      maxTransfers: 0,
    },
  },
  notifications: [],
  bookings: [],
  invoices: [],
});
export const serverSubscriptions=ref<any[]>([]), serverPayments=ref<any[]>([]);
export const issuedLinks=reactive<Record<string,{id:string,url:string}>>({});
const policyVersion=ref(0);
const ready = ref(false);
const historyLessons = ref<Lesson[]>([]);
let rawBookings: any[] = [],
  rawPlans: any[] = [],
  rawRules: any[] = [],
  rawGroups: any[] = [];
export const ruleGroup = ref("");
const clock = (value: string) =>
  new Date(new Date(value).getTime() + 5 * 3600000).toISOString();
export async function refresh() {
  // Keep mounted forms alive during refresh so awaited actions can close them.
  const master = user.value?.role === "master";
  if (!user.value) return;
  const families = master
    ? await all("/master/families")
    : [await api("/families/me")];
  const details = master
    ? await Promise.all(families.map((f) => api("/master/families/" + f.id)))
    : families;
  const children = details.flatMap((f) => f.children);
  rawGroups = await all(master ? "/master/groups" : "/public/groups");
  rawPlans = await all(master ? "/master/plans" : "/public/plans");
  const lessons = master
    ? await all("/master/lessons")
    : await all(
        "/public/schedule?from=" +
          new Date(Date.now() - 30 * 86400000).toISOString().slice(0, 10) +
          "&to=" +
          new Date(Date.now() + 60 * 86400000).toISOString().slice(0, 10),
      );
  rawBookings=await all(master?'/master/bookings':'/families/me/bookings');
  if (!master)
    for (const b of rawBookings)
      if (!lessons.some((l) => (l.id || l.lesson_id) === b.lesson_id))
        lessons.push({
          id: b.lesson_id,
          starts_at: b.starts_at,
          ends_at: b.ends_at,
          group_title: b.group_title,
          capacity: 1,
          seats_taken: 1,
          status: "completed",
        });
  const subs=await all(master?'/master/subscriptions':'/families/me/subscriptions');
  serverSubscriptions.value=subs;
  const invoices = await all(
    master ? "/master/invoices" : "/families/me/invoices",
  );
  const payments = await all(
    master ? "/master/payments" : "/families/me/payments",
  );
  serverPayments.value=payments;
  const notices = await all("/notifications");
  if (master) {
    rawRules = (await all("/master/attendance-rules")).map((r) => ({
      ...r,
      valid_from: r.valid_from.slice(0, 10),
      valid_to: r.valid_to?.slice(0, 10) || null,
    }));
    if (!ruleGroup.value) ruleGroup.value = rawGroups[0]?.id || "";
    await selectRule();
  }
  state.families = details.map((f) => ({
    id: f.id,
    parent: f.parent.full_name,
    phone: f.parent.phone || "",
    childIds: f.children.map((c: any) => c.id),
    balance: f.financial_summary.balance_minor / 100,
    accessToken: "",
  }));
  state.children = children.map((c) => ({
    id: c.id,
    name: c.full_name,
    age: c.birth_date
      ? Math.floor(
          (Date.now() - new Date(c.birth_date).getTime()) / (365.25 * 86400000),
        )
      : 0,
    familyId:
      c.family_id ||
      details.find((f) => f.children.some((x: any) => x.id === c.id))?.id,
  }));
  historyLessons.value = lessons.map((l) => {
    const id = l.id || l.lesson_id;
    const own = rawBookings
      .filter((b) => b.lesson_id === id && b.status === "active")
      .map((b) => b.child_id);
    const seats = Number(l.seats_taken || 0);
    return {
      id,
      date: clock(l.starts_at).slice(0, 10),
      time: clock(l.starts_at).slice(11, 16),
      duration: (Date.parse(l.ends_at) - Date.parse(l.starts_at)) / 60000,
      kind: l.is_individual ? "individual" : "group",
      title: l.group_title,
      groupId: l.group_id,
      isMakeup: l.is_makeup,
      capacity: l.capacity,
      childIds: master
        ? own
        : [
            ...own,
            ...Array.from(
              { length: Math.max(0, seats - own.length) },
              (_, i) => "occupied-" + i,
            ),
          ],
    } as Lesson;
  });
  state.lessons = historyLessons.value.filter(
    (l) =>
      lessons.find((x) => (x.id || x.lesson_id) === l.id)?.status !==
      "cancelled",
  );
  state.settings.tariffs = rawPlans
    .filter((p) => p.is_active !== false)
    .map((p) => ({
      id: p.id,
      title: p.title,
      groupId: p.group_id,
      groupName: rawGroups.find((g) => g.id === p.group_id)?.title,
      lessons: p.lessons_total,
      price: p.price_minor / 100,
      days: p.validity_days,
    }));
  state.subscriptions = {};
  for (const c of state.children) {
    const relevant = subs.filter(
      (s) => s.child_id === c.id,
    );
    state.subscriptions[c.id] = {
      childId: c.id,
      total: relevant.reduce((n, s) => n + s.lessons_total, 0),
      remaining: relevant.reduce((n,s)=>n+Number(s.lessons_available),0),
      reserved: relevant.reduce((n,s)=>n+s.lessons_reserved,0),
      used: relevant.reduce((n,s)=>n+s.lessons_used,0),
      validUntil:
        relevant
          .map((s) => s.valid_until.slice(0, 10))
          .sort()
          .at(-1) || DEMO_TODAY,
      singleCredits: 0,
      payments: payments
        .filter((p) => p.child_id === c.id && p.status === "succeeded")
        .map((p) => ({
          id: p.id,
          date: clock(p.paid_at).slice(0, 10),
          amount: p.amount_minor / 100,
          lessons:
            invoices.find((i) => i.id === p.invoice_id)?.plan_snapshot
              ?.lessons_total || 1,
          method: (
            {
              cash: "Наличные",
              test: "Тестовая оплата",
              transfer: "Перевод",
              online: "ЮKassa",
              sbp: "СБП",
            } as any
          )[p.method],
          invoiceId: p.invoice_id,
          receiptHash: p.receipt?.id,
          receiptUrl:
            p.receipt?.kind === "link"
              ? p.receipt.external_link
              : p.receipt?.download_url,
          receiptName: p.receipt ? "Чек" : undefined,
        })),
    };
  }
  state.bookings = rawBookings.map((b) => ({
    id: b.id,
    familyId: b.family_id || user.value.family_id,
    childId: b.child_id,
    lessonId: b.lesson_id,
    status: b.status === "active" ? "confirmed" : "cancelled",
    fundingKind:b.funding_kind,
    createdAt: b.created_at,
  }));
  state.marks = {};
  for (const b of rawBookings) {
    const a = b.attendance;
    if (a && a.status !== "pending")
      state.marks[markKey(b.lesson_id, b.child_id)] = {
        status: (
          { attended: "present", missed: "absent", cancelled: "cancel" } as any
        )[a.status],
        charged: a.charged,
        history: [{ at: a.marked_at, label: a.comment || "Отметка мастера" }],
      };
  }
  state.invoices = invoices.map((i) => ({
    id: i.id,
    familyId: i.family_id,
    childId: i.child_id,
    kind: i.kind,
    title: i.plan_snapshot?.title || "Разовое занятие",
    amount: i.amount_minor / 100,
    lessons: i.plan_snapshot?.lessons_total || 1,
    status: i.status,
    createdAt: clock(i.created_at).slice(0, 10),
    paidAt: payments.find(
      (p) => p.invoice_id === i.id && p.status === "succeeded",
    )?.paid_at
      ? clock(
          payments.find(
            (p) => p.invoice_id === i.id && p.status === "succeeded",
          ).paid_at,
        ).slice(0, 10)
      : undefined,
    paymentId: payments.find(
      (p) => p.invoice_id === i.id && p.status === "succeeded",
    )?.id,
  }));
  state.notifications = notices.map((n) => ({
    id: n.id,
    title:
      (
        {
          "lesson.changed": "Занятие изменено",
          "lesson.cancelled": "Занятие отменено",
          "booking.cancelled": "Запись отменена",
          "booking.transferred": "Запись перенесена",
          "payment.required": "Необходима оплата",
          "payment.succeeded": "Оплата получена",
          "booking.confirmed": "Запись подтверждена",
          "receipt.attached": "Прикреплён чек",
          "subscription.ending": "Абонемент заканчивается",
        } as any
      )[n.type] || "Уведомление мастерской",
    text: n.payload?.message || "",
    time: new Date(n.created_at).toLocaleString("ru-RU"),
    tone: "calm",
    read: !!n.read_at,
    audience: master ? "master" : "parent",
    familyId: user.value.family_id,
  }));
  ready.value = true;
}
async function selectRule(){if(user.value?.role!=="master")return;const group=ruleGroup.value;try{const r=await api('/master/booking-policy');if(group!==ruleGroup.value)return;policyVersion.value=r.version;Object.assign(state.settings.rules,{chargeNoShow:r.missed_charge_lesson,cancelHours:r.free_cancel_before_hours,allowTransfer:r.transfer_allowed});}catch(e){policyVersion.value=0;throw e}}
watch(ruleGroup,selectRule);
const renewAttempts=new Map<string,{key:string,paymentId?:string,receiptDone?:boolean}>();
export function useDemo() {
  const familyById = (id: string) => state.families.find((f) => f.id === id);
  const childById = (id: string) => state.children.find((c) => c.id === id);
  const lessonById = (id: string) => state.lessons.find((l) => l.id === id);
  async function createBooking(_family: string, child: string, lesson: string, funding="subscription") {
    try {
      const b = await api(
        "/bookings",
        "POST",
        { child_id: child, lesson_id: lesson, funding_kind: funding },
        key(),
      );
      await refresh();
      return { ok: true as const, booking: b };
    } catch (e) {
      return {
        ok: false as const,
        code: "NOT_FOUND",
        message: (e as Error).message,
      };
    }
  }
  async function enroll(lesson: string, child: string) {
    const l=lessonById(lesson);const has=serverSubscriptions.value.some(s=>s.child_id===child&&(!s.group_id||s.group_id===l?.groupId));
    const r = await createBooking("", child, lesson,has?"subscription":"awaiting_payment");
    if (!r.ok) throw new Error(r.message);
    return "ok";
  }
  async function addLesson(input: any) {
    if(!/^([01]\d|2[0-3]):[0-5]\d$/.test(input.time)||!Number.isInteger(input.duration)||input.duration<1||input.duration>720)throw new Error("Укажите время HH:MM и длительность от 1 до 720 минут.");
    let g = rawGroups.find(
      (g) =>
        g.title === input.title &&
        g.is_individual === (input.kind === "individual"),
    );
    if (!g) {
      g = await api("/master/groups", "POST", {
        title: input.title,
        capacity: input.capacity,
        is_individual: input.kind === "individual",
      });
      await api("/master/attendance-rules", "POST", {
        group_id: g.id,
        valid_from: DEMO_TODAY,
        attended_charge_lesson: true,
        missed_charge_lesson: state.settings.rules.chargeNoShow,
        parent_cancel_charge_lesson: true,
        free_cancel_before_hours: state.settings.rules.cancelHours,
        parent_cancel_allowed: true,
        cancel_deadline_hours: 0,
        transfer_allowed: state.settings.rules.allowTransfer,
        transfer_deadline_hours: 0,
        transfer_charge_lesson: false,
        single_lesson_price_minor: Math.round(input.singlePrice * 100),
        ending_notice_days: 3,
        ending_notice_lessons: 1,
      });
    }
    const start = new Date(input.date + "T" + input.time + ":00+05:00");
    const l = await api("/master/lessons", "POST", {
      group_id: g.id,
      starts_at: start.toISOString(),
      ends_at: new Date(start.getTime() + input.duration * 60000).toISOString(),
      capacity: input.capacity,
      is_makeup: !!input.isMakeup,
    });
    for (const child of input.childIds)
      await api(
        "/bookings",
        "POST",
        { lesson_id: l.id, child_id: child },
        key(),
      );
    await refresh();
    return lessonById(l.id)!;
  }
  async function setMark(
    lesson: string,
    child: string,
    status: MarkStatus,
  ) {
    const b = rawBookings.find(
      (b) =>
        b.lesson_id === lesson && b.child_id === child && b.status === "active",
    );
    if (!b) throw new Error("Запись не найдена");
    const before = state.subscriptions[child]?.remaining || 0;
    await api(`/master/lessons/${lesson}/attendance/${b.id}`, "PUT", {
      status: { present: "attended", absent: "missed", cancel: "cancelled" }[
        status
      ],
      ...(status === "cancel" ? { cancel_reason: "master" } : {}),
      expected_version: b.attendance.version,
      comment: "Исправление отметки в кабинете мастера",
    });
    await refresh();
    const remaining = state.subscriptions[child]?.remaining || 0;
    return { remaining, delta: remaining - before };
  }
  async function renew(child:string,tariff:Tariff,method="cash",file?:File){const id=child+":"+tariff.id+":"+method;const attempt=renewAttempts.get(id)||{key:key()};renewAttempts.set(id,attempt);if(!attempt.paymentId){const p=await api('/master/subscription-purchases','POST',{child_id:child,plan_id:tariff.id,valid_from:DEMO_TODAY,method},attempt.key);attempt.paymentId=p.id;}
    if(file&&!attempt.receiptDone){try{const f=new FormData();f.append('file',file);await api('/master/payments/'+attempt.paymentId+'/receipts','POST',f);attempt.receiptDone=true;}catch(e){await refresh();throw new Error('Оплата сохранена. Чек не загружен: '+(e as Error).message+' Повторите загрузку или добавьте чек в Финансах.');}}
    await refresh();renewAttempts.delete(id);
  }
  async function addFamily(parent: string, phone: string, kids: any[]) {
    const f = await api("/master/families", "POST", {
      parent_full_name: parent,
      phone,
      children: kids.map((k) => ({
        full_name: k.name,
        ...(k.birthDate ? { birth_date: k.birthDate } : {}),
      })),
    });
    await refresh();
    return familyById(f.id)!;
  }
  async function issueLink(family: Family) {
    const l = await api(`/master/families/${family.id}/access-links`, "POST", {
      expires_at: new Date(Date.now() + 30 * 86400000).toISOString(),
    });
    const url=location.origin+"/access#token="+l.url.split("token=")[1];issuedLinks[family.id]={id:l.id,url};return url;
  }
  async function updateTariff(id: string, p: any) {
    const old = rawPlans.find((x) => x.id === id);
    await api("/master/plans/" + id, "PATCH", {
      expected_version: old.version,
      title: p.title,
      lessons_total: p.lessons,
      price_minor: Math.round(p.price * 100),
      validity_days: p.days,
    });
    await refresh();
  }
  async function createTariff(p:any){await api('/master/plans','POST',{title:p.title,lessons_total:p.lessons,price_minor:Math.round(p.price*100),validity_days:p.days});await refresh();}
  async function saveSettings(){await api('/master/booking-policy','PUT',{expected_version:policyVersion.value,missed_charge_lesson:state.settings.rules.chargeNoShow,free_cancel_before_hours:state.settings.rules.cancelHours,transfer_allowed:state.settings.rules.allowTransfer});await selectRule();}
  async function attachReceipt(_child: string, payment: string, file: File) {
    const form = new FormData();
    form.append("file", file);
    const old = state.subscriptions[_child]?.payments.find(
      (p) => p.id === payment,
    )?.receiptHash;
    let method = "POST";
    if (old) {
      const receipt = await api("/receipts/" + old);
      form.append("expected_version", String(receipt.version));
      form.append("comment", "Замена чека мастером");
      method = "PUT";
    }
    await api(`/master/payments/${payment}/receipts`, method, form);
    await refresh();
    return true;
  }
  async function markAllReadFor() {
    await api("/notifications/read-all", "POST");
    await refresh();
  }
  return {
    state,
    serverSubscriptions,serverPayments,issuedLinks,createTariff,
    historyLessons,
    ready,
    refresh,
    ruleGroup,
    groups: () => rawGroups,
    familyById,
    childById,
    subscriptionFor: (id: string) => state.subscriptions[id],
    familyForChild: (c: any) => familyById(c.familyId),
    lessonsFor: (date: string) =>
      state.lessons
        .filter((l) => l.date === date)
        .sort((a, b) => a.time.localeCompare(b.time)),
    lessonById,
    markFor: (lesson: string, child: string) =>
      state.marks[markKey(lesson, child)],
    addLesson,
    enroll,
    setMark,
    renew,
    attachReceipt,
    addFamily,
    issueLink,
    updateTariff,
    saveSettings,
    markAllReadFor,
    markAllRead: markAllReadFor,
    resetDemo: refresh,
    createBooking,
    bookingsForFamily: (id: string) =>
      state.bookings.filter((b) => b.familyId === id),
    invoicesForFamily: (id: string) =>
      state.invoices.filter((i) => i.familyId === id),
    paymentForInvoice: (i: any) =>
      state.subscriptions[i.childId]?.payments.find(
        (p) => p.invoiceId === i.id,
      ),
    notificationsFor: (_a?: string, _f?: string) => state.notifications,
    unreadFor: (_a?: string, _f?: string) =>
      state.notifications.filter((n) => !n.read).length,
  };
}
