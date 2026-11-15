import { ref } from "vue";
export const user = ref<any>(null),
  apiError = ref(""),
  busy = ref(false);
let csrf = "";
const messages: Record<string, string> = {
  NO_AVAILABLE_LESSONS: "Нет доступных занятий подходящего абонемента. Оплатите абонемент или явно выберите разовое занятие.",
  PROVIDER_REFUND_REQUIRED: "Онлайн-платёж требует возврата через платёжный сервис.",
  PAYMENT_VOIDED: "Оплата аннулирована.",
  RATE_LIMIT: "Слишком много запросов. Подождите минуту и повторите действие.",
  UNAUTHORIZED: "Войдите в кабинет.",
  INVALID_CREDENTIALS: "Проверьте логин и пароль.",
  SESSION_CONTEXT_CONFLICT:
    "Открыты две сессии. Нажмите «Выйти» и войдите снова.",
  SEAT_LIMIT_EXCEEDED: "Свободных мест нет.",
  DUPLICATE_BOOKING: "Ребёнок уже записан.",
  LESSON_NOT_STARTED: "Занятие ещё не началось.",
  MASTER_TIME_CONFLICT: "У мастера уже есть занятие в это время.",
  PAYMENT_NOT_CONFIGURED: "Онлайн-оплата ещё не настроена мастером.",
  VERSION_CONFLICT: "Данные изменились. Обновите страницу.",
  RULE_IN_USE:
    "Правило уже используется. Создайте новую версию на будущую дату.",
  CSRF: "Сессия устарела. Войдите заново.",
};
export async function api(
  path: string,
  method = "GET",
  body?: any,
  key?: string,
): Promise<any> {
  const headers: Record<string, string> = {};
  if (method !== "GET") headers["X-CSRF-Token"] = csrf;
  if (body && !(body instanceof FormData))
    headers["Content-Type"] = "application/json";
  if (key) headers["Idempotency-Key"] = key;
  let response: Response;
  try {
    response = await fetch("/api/v1" + path, {
      method,
      credentials: "include",
      headers,
      body:
        body instanceof FormData
          ? body
          : body === undefined
            ? undefined
            : JSON.stringify(body),
    });
  } catch {
    throw new Error("Нет связи с сервером. Попробуйте ещё раз.");
  }
  const data =
    response.status === 204
      ? null
      : await response.json().catch(() => {
          throw new Error(
            "Сервер временно недоступен. Попробуйте обновить данные.",
          );
        });
  if (!response.ok) {
    if (response.status === 401) {
      user.value = null;
      csrf = "";
    }
    throw new Error(
      messages[data?.error?.code] ||
        data?.error?.message ||
        `Ошибка ${response.status}`,
    );
  }
  if (data?.csrf_token) csrf = data.csrf_token;
  return data;
}
export async function all(path: string): Promise<any[]> {
  const rows: any[] = [];
  for (let offset = 0; ; offset += 200) {
    const d = await api(
      `${path}${path.includes("?") ? "&" : "?"}limit=200&offset=${offset}`,
    );
    rows.push(...d.items);
    if (rows.length >= d.total || !d.items.length) return rows;
  }
}
export async function session() {
  user.value = await api("/auth/me");
  return user.value;
}
export async function logout() {
  await api("/auth/logout", "POST");
  user.value = null;
  csrf = "";
  location.href = "/yasna";
}
export const key = () => crypto.randomUUID();
export function report(error: unknown) {
  apiError.value =
    error instanceof Error ? error.message : "Не удалось выполнить действие.";
}
