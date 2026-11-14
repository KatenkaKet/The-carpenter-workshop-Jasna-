// @shared

export type LessonKind = "group" | "individual";
export type MarkStatus = "present" | "absent" | "cancel";

export interface Child {
  id: string;
  name: string;
  age: number;
  familyId: string;
}

export interface Family {
  id: string;
  parent: string;
  phone: string;
  childIds: string[];
  balance: number;
}

export interface Lesson {
  id: string;
  date: string;
  time: string;
  duration: number;
  kind: LessonKind;
  title: string;
  capacity: number;
  childIds: string[];
}

export interface Payment {
  id: string;
  date: string;
  amount: number;
  lessons: number;
  method: string;
  receiptHash?: string;
  receiptName?: string;
  receiptAt?: string;
}

export interface Subscription {
  childId: string;
  total: number;
  remaining: number;
  validUntil: string;
  payments: Payment[];
}

export interface MarkChange {
  at: string;
  label: string;
}

export interface Mark {
  status: MarkStatus;
  grade?: number;
  charged: boolean;
  history: MarkChange[];
}

export interface Tariff {
  id: string;
  title: string;
  lessons: number;
  price: number;
  days: number;
}

export interface Rules {
  chargeNoShow: boolean;
  cancelHours: number;
  allowTransfer: boolean;
  maxTransfers: number;
}

export interface AppNotification {
  id: string;
  title: string;
  text: string;
  time: string;
  tone: "warm" | "good" | "calm";
  read: boolean;
}

export interface DemoSettings {
  tariffs: Tariff[];
  rules: Rules;
}

export interface DemoState {
  version: number;
  families: Family[];
  children: Child[];
  lessons: Lesson[];
  subscriptions: Record<string, Subscription>;
  marks: Record<string, Mark>;
  settings: DemoSettings;
  notifications: AppNotification[];
}

export const DEMO_TODAY = "2026-11-14";
export const DEMO_NOW_MINUTES = 11 * 60 + 25;

export const MONTHS_NOMINATIVE = [
  "Январь",
  "Февраль",
  "Март",
  "Апрель",
  "Май",
  "Июнь",
  "Июль",
  "Август",
  "Сентябрь",
  "Октябрь",
  "Ноябрь",
  "Декабрь",
];

export const MONTHS_GENITIVE = [
  "января",
  "февраля",
  "марта",
  "апреля",
  "мая",
  "июня",
  "июля",
  "августа",
  "сентября",
  "октября",
  "ноября",
  "декабря",
];

export const WEEKDAYS_SHORT = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"];
export const WEEKDAYS_FULL = [
  "понедельник",
  "вторник",
  "среда",
  "четверг",
  "пятница",
  "суббота",
  "воскресенье",
];

export function dateParts(iso: string): {
  year: number;
  month: number;
  day: number;
} {
  const [year = "2026", month = "11", day = "14"] = iso.split("-");
  return {
    year: Number(year),
    month: Math.max(0, Math.min(11, Number(month) - 1)),
    day: Number(day),
  };
}

export function isoFrom(year: number, month: number, day: number): string {
  const monthPart = String(month + 1).padStart(2, "0");
  const dayPart = String(day).padStart(2, "0");
  return `${year}-${monthPart}-${dayPart}`;
}

export function formatDay(iso: string): string {
  const { month, day } = dateParts(iso);
  return `${day} ${MONTHS_GENITIVE[month] ?? ""}`;
}

export function formatMonth(year: number, month: number): string {
  return `${MONTHS_NOMINATIVE[month] ?? ""} ${year}`;
}

export function weekdayIndex(iso: string): number {
  const { year, month, day } = dateParts(iso);
  return (new Date(Date.UTC(year, month, day)).getUTCDay() + 6) % 7;
}

export function formatWeekday(iso: string): string {
  return WEEKDAYS_FULL[weekdayIndex(iso)] ?? "";
}

export function formatDateLong(iso: string): string {
  return `${formatWeekday(iso)}, ${formatDay(iso)}`;
}

export function addDays(iso: string, offset: number): string {
  const { year, month, day } = dateParts(iso);
  const next = new Date(Date.UTC(year, month, day + offset));
  return next.toISOString().slice(0, 10);
}

export function monthCells(
  year: number,
  month: number,
): Array<{ day: number; iso: string } | null> {
  const first = new Date(Date.UTC(year, month, 1));
  const offset = (first.getUTCDay() + 6) % 7;
  const total = new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
  const cells: Array<{ day: number; iso: string } | null> = [];
  for (let i = 0; i < offset; i += 1) cells.push(null);
  for (let day = 1; day <= total; day += 1) {
    cells.push({ day, iso: isoFrom(year, month, day) });
  }
  while (cells.length % 7 !== 0) cells.push(null);
  return cells;
}

export function timeToMinutes(time: string): number {
  const [hours = "0", minutes = "0"] = time.split(":");
  return Number(hours) * 60 + Number(minutes);
}

export type LessonStatus = "done" | "now" | "soon" | "planned";

export function lessonStatus(lesson: Lesson): LessonStatus {
  if (lesson.date < DEMO_TODAY) return "done";
  if (lesson.date > DEMO_TODAY) return "planned";
  const start = timeToMinutes(lesson.time);
  const end = start + lesson.duration;
  if (DEMO_NOW_MINUTES >= end) return "done";
  if (DEMO_NOW_MINUTES >= start) return "now";
  if (start - DEMO_NOW_MINUTES <= 90) return "soon";
  return "planned";
}

export function formatMoney(value: number): string {
  return `${String(Math.round(value)).replace(/\B(?=(\d{3})+(?!\d))/g, "\u00A0")} ₽`;
}

export function plural(
  value: number,
  one: string,
  few: string,
  many: string,
): string {
  const mod10 = value % 10;
  const mod100 = value % 100;
  if (mod10 === 1 && mod100 !== 11) return one;
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few;
  return many;
}

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join("");
}

const AVATAR_TONES = [
  "bg-honey-soft text-honey",
  "bg-forest-soft text-forest",
  "bg-clay-soft text-clay",
  "bg-[#EAE5F2] text-lilac",
];

export function avatarTone(seed: string): string {
  let sum = 0;
  for (let i = 0; i < seed.length; i += 1) sum += seed.charCodeAt(i);
  return AVATAR_TONES[sum % AVATAR_TONES.length] ?? AVATAR_TONES[0] ?? "";
}

export function markKey(lessonId: string, childId: string): string {
  return `${lessonId}:${childId}`;
}

export function markLabel(status: MarkStatus, grade?: number): string {
  if (status === "present") {
    return grade ? `Посетил · оценка ${grade}` : "Посетил";
  }
  if (status === "absent") return "Пропустил";
  return "Отмена";
}

export function currentTimeLabel(): string {
  if (typeof Date === "undefined") return "11:25";
  const now = new Date();
  return `${String(now.getHours()).padStart(2, "0")}:${String(
    now.getMinutes(),
  ).padStart(2, "0")}`;
}

function lesson(
  id: string,
  date: string,
  time: string,
  kind: LessonKind,
  title: string,
  capacity: number,
  childIds: string[],
  duration = 90,
): Lesson {
  return { id, date, time, duration, kind, title, capacity, childIds };
}

function subscription(
  childId: string,
  total: number,
  remaining: number,
  validUntil: string,
  payments: Payment[] = [],
): Subscription {
  return { childId, total, remaining, validUntil, payments };
}

export function createSeed(): DemoState {
  const families: Family[] = [
    {
      id: "f1",
      parent: "Ольга Смирнова",
      phone: "+7 922 145-67-89",
      childIds: ["c1", "c2"],
      balance: 0,
    },
    {
      id: "f2",
      parent: "Артём Кузнецов",
      phone: "+7 922 310-44-12",
      childIds: ["c3"],
      balance: 0,
    },
    {
      id: "f3",
      parent: "Марина Петрова",
      phone: "+7 950 220-18-03",
      childIds: ["c4", "c5"],
      balance: 1200,
    },
    {
      id: "f4",
      parent: "Дарья Иванова",
      phone: "+7 912 887-32-55",
      childIds: ["c6"],
      balance: -900,
    },
    {
      id: "f5",
      parent: "Дмитрий Соколов",
      phone: "+7 922 664-90-41",
      childIds: ["c7"],
      balance: 0,
    },
    {
      id: "f6",
      parent: "Анна Морозова",
      phone: "+7 932 401-77-26",
      childIds: ["c8"],
      balance: 0,
    },
    {
      id: "f7",
      parent: "Сергей Волков",
      phone: "+7 950 733-05-19",
      childIds: ["c9"],
      balance: 0,
    },
    {
      id: "f8",
      parent: "Ирина Зайцева",
      phone: "+7 922 508-62-34",
      childIds: ["c10"],
      balance: -600,
    },
  ];

  const children: Child[] = [
    { id: "c1", name: "Мирон", age: 9, familyId: "f1" },
    { id: "c2", name: "Варя", age: 6, familyId: "f1" },
    { id: "c3", name: "Лев", age: 11, familyId: "f2" },
    { id: "c4", name: "София", age: 8, familyId: "f3" },
    { id: "c5", name: "Тимур", age: 10, familyId: "f3" },
    { id: "c6", name: "Ева", age: 7, familyId: "f4" },
    { id: "c7", name: "Марк", age: 9, familyId: "f5" },
    { id: "c8", name: "Алиса", age: 6, familyId: "f6" },
    { id: "c9", name: "Даниил", age: 12, familyId: "f7" },
    { id: "c10", name: "Полина", age: 8, familyId: "f8" },
  ];

  const subscriptions: Record<string, Subscription> = {
    c1: subscription("c1", 4, 3, "2026-12-14", [
      {
        id: "p1",
        date: "2026-10-17",
        amount: 3200,
        lessons: 4,
        method: "Перевод",
        receiptHash: "msk_LlUQoujnnX_p.pdf",
        receiptName: "Чек_14-001.pdf",
        receiptAt: "2026-10-17",
      },
    ]),
    c2: subscription("c2", 4, 0, "2026-11-20", [
      {
        id: "p2",
        date: "2026-10-20",
        amount: 3200,
        lessons: 4,
        method: "Наличные",
      },
    ]),
    c3: subscription("c3", 4, 2, "2026-12-01", [
      {
        id: "p3",
        date: "2026-10-21",
        amount: 3200,
        lessons: 4,
        method: "Перевод",
      },
    ]),
    c4: subscription("c4", 8, 3, "2026-12-20", [
      {
        id: "p4",
        date: "2026-10-25",
        amount: 5800,
        lessons: 8,
        method: "СБП",
      },
    ]),
    c5: subscription("c5", 8, 5, "2027-01-15", [
      {
        id: "p5",
        date: "2026-10-27",
        amount: 5800,
        lessons: 8,
        method: "СБП",
      },
    ]),
    c6: subscription("c6", 4, 1, "2026-11-28", [
      {
        id: "p6",
        date: "2026-10-28",
        amount: 3200,
        lessons: 4,
        method: "Наличные",
      },
    ]),
    c7: subscription("c7", 8, 4, "2026-12-10", [
      {
        id: "p7",
        date: "2026-10-30",
        amount: 5800,
        lessons: 8,
        method: "Перевод",
      },
    ]),
    c8: subscription("c8", 4, 2, "2026-12-05", [
      {
        id: "p8",
        date: "2026-11-01",
        amount: 3200,
        lessons: 4,
        method: "СБП",
      },
    ]),
    c9: subscription("c9", 8, 6, "2026-12-25", [
      {
        id: "p9",
        date: "2026-11-03",
        amount: 5800,
        lessons: 8,
        method: "Перевод",
      },
    ]),
    c10: subscription("c10", 4, 0, "2026-11-10", [
      {
        id: "p10",
        date: "2026-11-05",
        amount: 3200,
        lessons: 4,
        method: "Наличные",
      },
    ]),
  };

  const lessons: Lesson[] = [
    lesson(
      "l_0310",
      "2026-11-03",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c4", "c6"],
    ),
    lesson("l_0516", "2026-11-05", "16:00", "group", "Шкатулка · 8–12 лет", 4, [
      "c5",
      "c9",
    ]),
    lesson(
      "l_0710",
      "2026-11-07",
      "10:00",
      "group",
      "Первые поделки · 5–7 лет",
      4,
      ["c2", "c8"],
    ),
    lesson(
      "l_1013",
      "2026-11-10",
      "12:30",
      "individual",
      "Индивидуальное занятие",
      1,
      ["c3"],
    ),
    lesson(
      "l_1210",
      "2026-11-12",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c4", "c7"],
    ),
    lesson(
      "l_1410",
      "2026-11-14",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c4", "c6", "c7", "c10"],
    ),
    lesson(
      "l_1413",
      "2026-11-14",
      "12:30",
      "individual",
      "Разделочная доска",
      1,
      ["c3"],
    ),
    lesson("l_1416", "2026-11-14", "16:00", "group", "Шкатулка · 8–12 лет", 4, [
      "c5",
      "c9",
      "c8",
    ]),
    lesson(
      "l_1710",
      "2026-11-17",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c4", "c6"],
    ),
    lesson(
      "l_1716",
      "2026-11-17",
      "16:00",
      "individual",
      "Индивидуальное занятие",
      1,
      ["c3"],
    ),
    lesson(
      "l_1910",
      "2026-11-19",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c5", "c7", "c10"],
    ),
    lesson("l_2112", "2026-11-21", "12:00", "group", "Шкатулка · 8–12 лет", 4, [
      "c8",
      "c2",
    ]),
    lesson(
      "l_2410",
      "2026-11-24",
      "10:00",
      "group",
      "Первые поделки · 5–7 лет",
      4,
      ["c4", "c6"],
    ),
    lesson(
      "l_2416",
      "2026-11-24",
      "16:00",
      "individual",
      "Индивидуальное занятие",
      1,
      ["c9"],
    ),
    lesson(
      "l_2610",
      "2026-11-26",
      "10:00",
      "group",
      "Столярная мастерская · 7–10 лет",
      6,
      ["c1", "c5"],
    ),
    lesson("l_2617", "2026-11-26", "17:30", "group", "Шкатулка · 8–12 лет", 4, [
      "c8",
      "c10",
    ]),
    lesson(
      "l_2812",
      "2026-11-28",
      "12:00",
      "group",
      "Первые поделки · 5–7 лет",
      4,
      ["c2", "c6", "c8"],
    ),
  ];

  const marks: Record<string, Mark> = {
    [markKey("l_1410", "c1")]: {
      status: "present",
      grade: 5,
      charged: true,
      history: [{ at: "10:04", label: "Посетил · оценка 5" }],
    },
    [markKey("l_1410", "c4")]: {
      status: "present",
      grade: 4,
      charged: true,
      history: [{ at: "10:05", label: "Посетил · оценка 4" }],
    },
    [markKey("l_1410", "c6")]: {
      status: "absent",
      charged: false,
      history: [
        { at: "10:06", label: "Посетил · оценка 5" },
        { at: "10:41", label: "Пропустил" },
      ],
    },
    [markKey("l_1410", "c7")]: {
      status: "present",
      grade: 5,
      charged: true,
      history: [{ at: "10:07", label: "Посетил · оценка 5" }],
    },
    [markKey("l_1410", "c10")]: {
      status: "absent",
      charged: false,
      history: [{ at: "10:08", label: "Пропустил" }],
    },
  };

  return {
    version: 2,
    families,
    children,
    lessons,
    subscriptions,
    marks,
    settings: {
      tariffs: [
        {
          id: "t1",
          title: "Пробное занятие",
          lessons: 1,
          price: 900,
          days: 14,
        },
        {
          id: "t2",
          title: "Абонемент «Четыре»",
          lessons: 4,
          price: 3200,
          days: 60,
        },
        {
          id: "t3",
          title: "Абонемент «Восемь»",
          lessons: 8,
          price: 5800,
          days: 90,
        },
      ],
      rules: {
        chargeNoShow: false,
        cancelHours: 12,
        allowTransfer: true,
        maxTransfers: 1,
      },
    },
    notifications: [
      {
        id: "n1",
        title: "Закончился абонемент",
        text: "У Вари Смирновой осталось 0 занятий",
        time: "12 минут назад",
        tone: "warm",
        read: false,
      },
      {
        id: "n2",
        title: "Оплата получена",
        text: "3 200 ₽ · Мирон Смирнов, 4 занятия",
        time: "1 час назад",
        tone: "good",
        read: false,
      },
      {
        id: "n3",
        title: "Новая запись",
        text: "Тимур записан на 17 ноября, 10:00",
        time: "Вчера",
        tone: "calm",
        read: true,
      },
    ],
  };
}
