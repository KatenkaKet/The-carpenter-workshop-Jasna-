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
  accessToken: string;
}

export interface Lesson {
  groupId?: string;
  isMakeup?: boolean;
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
  receiptUrl?: string;
  receiptName?: string;
  receiptAt?: string;
  invoiceId?: string;
  kind?: "subscription" | "single_lesson";
}

export interface Subscription {
  reserved: number;
  used: number;
  childId: string;
  total: number;
  remaining: number;
  validUntil: string;
  payments: Payment[];
  singleCredits: number;
}

export interface MarkChange {
  at: string;
  label: string;
}

export interface Mark {
  status: MarkStatus;
  charged: boolean;
  history: MarkChange[];
}

export interface Tariff {
  groupId?: string;
  groupName?: string;
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
  audience?: "master" | "parent";
  familyId?: string;
}

export type ParentBookingStatus =
  "pending" | "confirmed" | "rejected" | "cancelled";

export interface ParentBooking {
  id: string;
  familyId: string;
  childId: string;
  lessonId: string;
  status: ParentBookingStatus;
  createdAt: string;
  confirmedAt?: string;
  invoiceId?: string;
  paidWith?: "subscription" | "single";
  note?: string;
}

export interface Invoice {
  id: string;
  familyId: string;
  childId: string;
  bookingId?: string;
  kind: "single_lesson" | "subscription";
  title: string;
  amount: number;
  lessons: number;
  status: "open" | "paid" | "cancelled";
  createdAt: string;
  paidAt?: string;
  paymentId?: string;
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
  bookings: ParentBooking[];
  invoices: Invoice[];
}

export const DEMO_TODAY = new Date(Date.now() + 5 * 3600000)
  .toISOString()
  .slice(0, 10);
export const DEMO_NOW_MINUTES =
  ((new Date().getUTCHours() + 5) % 24) * 60 + new Date().getUTCMinutes();

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
  const [year = "2026", month = "11", day = "14"] = iso.slice(0, 10).split("-");
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

export function markLabel(status: MarkStatus): string {
  if (status === "present") {
    return "Посетил";
  }
  if (status === "absent") return "Пропустил";
  return "Отмена";
}

export function formatMoment(value: string): string {
  const date = value.slice(0, 10);
  const time = value.length >= 16 ? value.slice(11, 16) : "";
  return time ? `${formatDay(date)}, ${time}` : formatDay(date);
}

export function currentMoment(): string {
  return `${DEMO_TODAY}T${currentTimeLabel()}`;
}

export function currentTimeLabel(): string {
  if (typeof Date === "undefined") return "11:25";
  const now = new Date();
  return `${String(now.getHours()).padStart(2, "0")}:${String(
    now.getMinutes(),
  ).padStart(2, "0")}`;
}
