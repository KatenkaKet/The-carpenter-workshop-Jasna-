# 📚 REST API — Система записи и учёта занятий «Ясна»

## 📑 Содержание

1. [Общие соглашения](#0-общие-соглашения)
2. [Доступ и сессии](#1-доступ-и-сессии)
3. [Публичное расписание](#2-публичное-расписание)
4. [Семьи и дети](#3-семьи-и-дети)
5. [Группы, занятия и запись](#4-группы-занятия-и-запись)
6. [Посещаемость и правила](#5-посещаемость-и-правила)
7. [Тарифы и абонементы](#6-тарифы-и-абонементы)
8. [Счета, денежный баланс и задолженность](#7-счета-денежный-баланс-и-задолженность)
9. [Платежи и ЮKassa](#8-платежи-и-yukassa)
10. [Чеки](#9-чеки)
11. [Уведомления и аудит](#10-уведомления-и-аудит)
12. [CSV и эксплуатация](#11-csv-и-эксплуатация)
13. [Безопасность и приёмка](#12-безопасность-и-проверки-приёмки)
14. [Дополнительный этап (не MVP)](#13-дополнительный-этап-не-обязательный-mvp)

---

## 0. Общие соглашения

### 👥 Роли и доступ

| Роль | Описание |
|------|----------|
| `guest` | Неавторизованный пользователь |
| `parent` | Родитель (семья определяется из серверной сессии) |
| `master` | Мастер (администратор мастерской) |

- `/public/*` — доступно всем
- `/master/*` — только для `master`
- Для `child_id`, `booking_id`, `subscription_id`, `invoice_id`, `payment_id`, `receipt_id` **всегда** проверяется принадлежность
- ⚠️ **UUID не является средством авторизации**

### 🔐 Коды ошибок доступа

| Код | Значение |
|-----|----------|
| `401` | Нет действующей сессии |
| `403` | Неверная роль / CSRF |
| `404` | Чужой объект (без раскрытия факта существования) |

### 🆔 Форматы данных

- **ID:** UUID
- **Даты:** `YYYY-MM-DD`
- **Время:** RFC 3339 с зоной, хранение в UTC
- **Бизнес-зона:** `Asia/Yekaterinburg`
- **Диапазон `from`/`to` в списках:** даты включительно в этой зоне; реализуется как `[начало from, начало следующего дня после to)`

### 💰 Деньги

- `amount_minor` и прочие `*_minor` — **целые копейки** (`int64`)
- `currency` = `"RUB"`
- ⚠️ **Не использовать `float` для расчётов**
- Пример: `400000` = 4000 рублей

### 📋 Списки

| Параметр | Диапазон | По умолчанию |
|----------|----------|--------------|
| `limit` | 1…200 | 50 |
| `offset` | ≥ 0 | 0 |

- Стабильная сортировка: `created_at, id` (если не указано иное)
- Ответ: `{ items, total, limit, offset }`
- Неизвестные поля запроса отклоняются с `422`
- `PATCH` — только перечисленные поля

### 🚫 Скрытые поля

Никогда не возвращаются: `token_hash`, `password_hash`, ключи провайдера.

### ✅ HTTP-коды успеха

| Метод | Код |
|-------|-----|
| `GET` / `PATCH` / `PUT` | `200` |
| Создание | `201` |
| Удаление / отзыв | `204` |

### ❌ Формат ошибок

```json
{
  "error": {
    "code": "SEAT_LIMIT_EXCEEDED",
    "message": "Нет свободных мест",
    "details": { "capacity": 2 }
  },
  "request_id": "..."
}
```

| Код | Назначение |
|-----|------------|
| `400` | Некорректный JSON |
| `401` | Проблема с сессией |
| `403` | Роль / CSRF |
| `404` | Объект не найден |
| `409` | Конфликт |
| `413` | Размер файла |
| `422` | Ошибка валидации |
| `429` | Лимит (`Retry-After`) |
| `503` | Временный сбой |

> 📌 Каждый ответ содержит `X-Request-Id`. Все успешные изменения сохраняются в аудит.

### 🔁 Идемпотентность

**`Idempotency-Key` обязателен** для `POST`:
- денежные операции,
- счета, корректировки,
- создание записи,
- перенос занятия.

Ключ сохраняется атомарно с операцией вместе с пользователем, маршрутом, хешем тела и результатом.

- Повтор с **тем же телом** → исходный ответ
- Повтор с **другим телом** → `409 IDEMPOTENCY_CONFLICT`
- Проверка доступа выполняется и при повторе
- Ключ запроса к провайдеру — отдельный, стабильно связан с попыткой оплаты

> 📌 `Idempotency-Key` не требуется и не поддерживается для `PUT /master/lessons/:lessonId/attendance/:bookingId`. Идемпотентность здесь достигается через `expected_version` и правило «повтор с текущей версией и тем же состоянием ничего не списывает». Если клиент присылает `Idempotency-Key` на этот маршрут, сервер возвращает `422`.

### 🗂 Версионирование

- Изменяемые объекты содержат `version`
- `version` — целое число, начинается с `1`, увеличивается на `1` при каждом успешном изменении сущности
- Для `PATCH` и `PUT` требуется `expected_version`
- Несовпадение → `409 VERSION_CONFLICT`
- Успешный `PATCH`/`PUT` возвращает объект с новой `version`
- Повтор `PUT` с текущей версией и тем же состоянием — ничего не списывает
- **История и финансовые движения неизменяемы**

### 📐 Проектные допущения (не готовые функции)

- Мастер создаёт семью и передаёт ссылку вручную; самостоятельная регистрация новых семей не включена
- Существующая семья записывает детей самостоятельно
- Для мастера — отдельный логин/пароль, созданный при установке
- Абонемент принадлежит **одному ребёнку и одной группе**, срок фиксируется в счёте
- Продажи — полными счетами
- Частичные оплаты, распределение переплаты, автоматические возвраты — **следующий этап**
- Переплата учитывается, но не активирует второй абонемент
- Новая покупка — отдельный абонемент, а не изменение оплаченного старого

---

## 1. Доступ и сессии

### 🔑 Модель входа

**Родитель** входит по персональной ссылке, без обязательных email и пароля.

Мастер выдаёт ссылку вида:
```
https://<host>/access#token=<secret>
```

- `secret`: 32 случайных байта из `crypto/rand` Go, `base64url`
- В БД хранится **только SHA-256 хеш**
- Ссылка повторно используемая до `expires_at` или отзыва
- Это осознанная модель **постоянного доступа**, а не одноразовая ссылка на email
- Срок задаёт мастер

> ⚠️ **Безопасность фронтенда:**
> - Фронтенд читает fragment, сразу удаляет его через `history.replaceState` и отправляет `secret` в `POST` body
> - На странице входа **нет сторонних скриптов/аналитики**
> - **Не писать** токен в логи, аудит, URL запросов или `localStorage`

---

### `POST /auth/access-link/exchange`

**Доступ:** `guest`

**Body:**
```json
{ "token": "string" }
```

**Ответ 200:**
```json
{
  "role": "parent",
  "family_id": "uuid",
  "csrf_token": "..."
}
```
+ `Set-Cookie`

Логика:
- Проверить срок / отзыв
- Создать серверную сессию с `access_link_id` и `family_id`
- Неверная, истёкшая или отозванная ссылка → `401 INVALID_ACCESS_LINK` (единый ответ, не раскрывает причину)

---

### `POST /auth/master/login`

**Доступ:** `guest`

**Body:**
```json
{ "login": "string", "password": "string" }
```

**Ответ 200:**
```json
{
  "role": "master",
  "csrf_token": "..."
}
```
+ отдельная cookie

- Учётная запись provisioned при установке; публичной регистрации мастера нет
- Пароль хранить как **Argon2id-хеш с солью**, не как открытый пароль в конфигурации
- `401 INVALID_CREDENTIALS`; лимит попыток по IP и логину
- Вход родителя **не может** выдавать роль `master`
- 2FA может быть добавлена отдельно

---

### `GET /auth/me`

**Доступ:** `parent` | `master`

**Ответ 200:**
```json
{
  "id": "uuid",
  "role": "parent|master",
  "full_name": "...",
  "family_id": "uuid|null",
  "csrf_token": "..."
}
```
> `family_id = null` для мастера

⚠️ Если присутствуют обе cookie → `409 SESSION_CONTEXT_CONFLICT`; интерфейс предлагает выйти или открыть отдельный профиль браузера.

---

### `POST /auth/logout`

**Доступ:** `guest` | `parent` | `master`

Отозвать предъявленные сессии и очистить обе cookie. Повтор → `204`.

---

### `GET /master/families/:id/access-links`

**Доступ:** `master`

Список ссылок семьи: `{ id, created_at, expires_at, revoked_at }`. Секрет не возвращается.

---

### `POST /master/families/:id/access-links`

**Доступ:** `master`

**Body:**
```json
{ "expires_at": "datetime" }
```

**Ответ 201:**
```json
{ "id": "uuid", "url": "...", "expires_at": "..." }
```
> 🔒 Секрет — **только в этом ответе**

- Новый выпуск **атомарно отзывает** старые ссылки семьи и все их сессии
- Срок должен быть в будущем
- Утерянную ссылку нельзя восстановить — только заменить

---

### `DELETE /master/families/:id/access-links/:linkId`

**Доступ:** `master`

`204`: отзыв ссылки и всех её сессий в одной транзакции. Повтор → `204`.

---

### 🍪 Сессии и cookie

**Сессии:** случайные opaque-токены, в БД — хеши + `role`, `user_id`, `family_id`, `access_link_id`, `expires_at`, `revoked_at`. JWT не нужен.

| Cookie | Срок |
|--------|------|
| `yasna_parent_session` | ≤ 30 дней, не дольше ссылки |
| `yasna_master_session` | ≤ 12 часов |

**Атрибуты cookie:** `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, без `Domain`.

- Проверять отзыв на каждом запросе, включая скачивание чеков
- Изменяющие запросы с cookie требуют `Origin` из разрешённого списка и `X-CSRF-Token`
- `login`/`exchange` проверяют `Origin`, принимают только JSON и ограничены по частоте
- Webhook не использует cookie/CSRF (проверка описана в разделе 8)

---

## 2. Публичное расписание

### `GET /public/groups`

**Query:** `is_individual?`, `limit?`, `offset?`

Только активные группы.

**items:**
```json
{
  "id": "uuid",
  "title": "...",
  "description": "...",
  "capacity": 10,
  "is_individual": false
}
```

---

### `GET /public/groups/:id`

**Ответ 200:** `{ id, title, description, capacity, is_individual }`

Неактивная группа → `404`.

---

### `GET /public/schedule`

**Query:** `from`, `to` (обязательны), `group_id?`, `limit?`, `offset?`

- Диапазон ≤ 93 дней
- Сортировка: `starts_at, id`

**items:**
```json
{
  "lesson_id": "uuid",
  "group_id": "uuid",
  "group_title": "...",
  "starts_at": "RFC3339",
  "ends_at": "RFC3339",
  "status": "scheduled",
  "capacity": 10,
  "seats_taken": 7,
  "seats_available": 3,
  "is_individual": false
}
```

> 🚫 Никаких имён, детей, записей и контактов в публичном ответе.
> ⚠️ `422 INVALID_RANGE`, если `from > to` или диапазон превышает 93 дня. `400` для некорректного формата даты.

---

### `GET /lessons/:id/availability`

**Доступ:** `parent` | `master`

**Ответ 200:**
```json
{
  "lesson_id": "uuid",
  "status": "scheduled",
  "capacity": 10,
  "seats_taken": 7,
  "seats_available": 3,
  "version": 1
}
```

> 📌 Доступность — справочная; окончательная проверка происходит при записи.

---

### `GET /public/plans`

**Query:** `group_id?`, `limit?`, `offset?`

Только активные тарифы.

**items:**
```json
{
  "id": "uuid",
  "title": "...",
  "group_id": "uuid",
  "lessons_total": 8,
  "validity_days": 30,
  "price_minor": 400000,
  "currency": "RUB",
  "version": 1
}
```

---

### `GET /public/booking-rules`

**Query:** `group_id?`, `date` (обязательно)

**Ответ 200:** действующие условия отмены, переноса и списания для показа до записи.
```json
{
  "group_id": "uuid|null",
  "date": "YYYY-MM-DD",
  "free_cancel_before_hours": 24,
  "cancel_deadline_hours": 24,
  "parent_cancel_allowed": true,
  "parent_cancel_charge_lesson": true,
  "transfer_allowed": true,
  "transfer_deadline_hours": 24,
  "transfer_charge_lesson": false,
  "single_lesson_price_minor": 80000,
  "currency": "RUB",
  "rule_version": 3
}
```
> `group_id = null` — применено правило по умолчанию.
> `rule_version` — идентификатор `attendance_rules.id`, применённого к этой паре `(group_id, date)`.
> Если правила не настроены — `409 RULES_NOT_CONFIGURED`.

---

## 3. Семьи и дети

### `GET /families/me`

**Доступ:** `parent`

**Ответ 200:**
```json
{
  "id": "uuid",
  "parent": {
    "id": "uuid",
    "full_name": "...",
    "phone": "...",
    "email": "..."
  },
  "children": [
    { "id": "uuid", "full_name": "...", "birth_date": "YYYY-MM-DD" }
  ],
  "financial_summary": {
    "balance_minor": 0,
    "debt_minor": 0,
    "currency": "RUB"
  }
}
```

> 📌 `phone`/`email` необязательны и **не определяют доступ**.
> 📌 `parent.id` — это `users.id`, не `families.id`. `family_id` совпадает с `id`.

---

### `GET /master/families`

**Доступ:** `master`

**Query:** `search?`, `limit?`, `offset?`

**items:** `{ id, parent_full_name, phone, email, children_count, version }`

---

### `GET /master/families/:id`

**Доступ:** `master`

**Ответ 200:** семья с детьми, `financial_summary` и `version`, без секретов доступа.

---

### `POST /master/families`

**Доступ:** `master`

**Body:**
```json
{
  "parent_full_name": "...",
  "phone": "...",
  "email": "...",
  "children": [
    { "full_name": "...", "birth_date": "YYYY-MM-DD" }
  ]
}
```

- Минимум один ребёнок
- Всё создаётся атомарно
- Email не обязателен и не уникальный ключ

**Ответ 201:** объект семьи с `version`. Выпуск ссылки — отдельная операция (раздел 1).

> 📌 Создаётся пользователь-родитель (`users.role = 'parent'`) без пароля и без `login`. Доступ к кабинету — только через ссылку доступа.

---

### `PATCH /master/families/:id`

**Доступ:** `master`

**Body:**
```json
{
  "expected_version": 1,
  "parent_full_name": "...",
  "phone": "...",
  "email": "..."
}
```

---

### `POST /master/families/:id/children`

**Доступ:** `master`

**Body:** `{ "full_name": "...", "birth_date": "YYYY-MM-DD" }`

**Ответ 201:** объект ребёнка.

---

### `GET /master/children`

**Доступ:** `master`

**Query:** `search?`, `family_id?`, `limit?`, `offset?`

**items:** `{ id, family_id, full_name, birth_date }`

---

### `GET /master/children/:id`

**Доступ:** `master`

**Ответ 200:** `{ id, family_id, full_name, birth_date, version }`

---

### `PATCH /master/children/:id`

**Доступ:** `master`

**Body:**
```json
{
  "expected_version": 1,
  "full_name": "...",
  "birth_date": "YYYY-MM-DD",
  "is_active": true
}
```

> 📌 Архивация сохраняет историю; при будущих активных записях → `409` (сначала отменить/перенести их).

---

## 4. Группы, занятия и запись

### `GET /master/groups`

**Доступ:** `master`

**Query:** `is_active?`, `search?`, `limit?`, `offset?`

**items:** объекты групп с `version`.

---

### `POST /master/groups`

**Доступ:** `master`

**Body:**
```json
{
  "title": "...",
  "capacity": 10,
  "is_individual": false,
  "description": "..."
}
```

- `capacity > 0`
- Индивидуальное занятие → `capacity = 1`

---

### `PATCH /master/groups/:id`

**Доступ:** `master`

**Body:**
```json
{
  "expected_version": 1,
  "title": "...",
  "capacity": 10,
  "description": "...",
  "is_active": true
}
```

> 📌 `capacity` группы — шаблон для новых занятий; существующие меняются явно.
> ⚠️ Архивация при будущих `scheduled` занятиях → `409`; сначала отменить их.

---

### `GET /master/lessons`

**Доступ:** `master`

**Query:** `from?`, `to?`, `group_id?`, `status?`, `limit?`, `offset?`

Сортировка: `starts_at, id`.

---

### `GET /master/lessons/:id`

**Доступ:** `master`

**Ответ 200:**
```json
{
  "id": "uuid",
  "group_id": "uuid",
  "starts_at": "RFC3339",
  "ends_at": "RFC3339",
  "capacity": 10,
  "seats_taken": 7,
  "status": "scheduled",
  "notes": "...",
  "version": 1
}
```

---

### `POST /master/lessons`

**Доступ:** `master`

**Body:**
```json
{
  "group_id": "uuid",
  "starts_at": "RFC3339",
  "ends_at": "RFC3339",
  "capacity": 10,
  "notes": "..."
}
```

- Будущее время, `ends_at > starts_at`
- `capacity` по умолчанию из группы
- Индивидуальное занятие → `capacity = 1`
- Для одного мастера **пересечения `scheduled`-занятий запрещены** даже между разными группами; проверка защищена от конкурентного создания

---

### `PATCH /master/lessons/:id`

**Доступ:** `master`

**Body:**
```json
{
  "expected_version": 1,
  "starts_at": "RFC3339",
  "ends_at": "RFC3339",
  "capacity": 10,
  "status": "scheduled|completed|cancelled",
  "cancel_reason": "...",
  "notes": "..."
}
```

**Правила переходов `status`:**

| Из → В | Разрешено |
|--------|-----------|
| `scheduled` → `completed` | ✅ после `ends_at` и обработки всех `pending`-отметок |
| `scheduled` → `cancelled` | ✅ |
| `completed` → `scheduled`/`cancelled` | ❌ |
| `cancelled` → `scheduled`/`completed` | ❌ |

> 📌 Исторические отметки `completed` можно исправлять через раздел 5.

**Ограничения:**
- Уменьшение `capacity` ниже активных записей → `409`
- Изменение времени повторяет проверки создания и проверяет пересечения у записанных детей; иначе `409`
- **Перенос времени** сохраняет записи и уведомляет семьи
- **При отмене мастером:** `cancel_reason` обязателен; все записи отменяются, все списания этого занятия компенсируются (включая прежние отмены родителями), создаются уведомления и аудит
- Оплаченные разовые счета при компенсации дают **кредит семьи**, не автоматический возврат

---

### `POST /bookings`

**Доступ:** `parent` | `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{ "lesson_id": "uuid", "child_id": "uuid" }
```

**Ответ 201:**
```json
{
  "id": "uuid",
  "lesson_id": "uuid",
  "child_id": "uuid",
  "status": "active",
  "version": 1
}
```

**Условия:**
- Занятие `scheduled` и ещё не началось
- Ребёнок активен и принадлежит семье
- Транзакция блокирует строку ребёнка и строку занятия
- Проверяются: доступ, вместимость, пересечение времени с активными записями этого ребёнка
- `UNIQUE(lesson_id, child_id) WHERE status='active'` защищает от дублей, но позволяет заново записаться после отмены с новым `booking_id`

> 🔒 Все отмены, переносы, изменения `capacity` используют тот же протокол блокировок.
> Единый порядок блокировок (дети по ID, занятия по ID) и повтор транзакции при deadlock.

**Возможные ошибки `409`:**
- `SEAT_LIMIT_EXCEEDED`
- `DUPLICATE_BOOKING`
- `CHILD_TIME_CONFLICT`
- `LESSON_UNAVAILABLE`

> 📌 Отсутствие абонемента **не запрещает** запись; деньги учитываются по разделам 5–8.

---

### `DELETE /bookings/:id`

**Доступ:** `parent` | `master`

`204` — мягкая отмена. Повтор отмены своей записи → `204`.

- Сохранить `cancelled_at`, `cancelled_by` и причину
- Проверить deadline по правилам
- Отмена **после начала** родителем → `409`; мастер исправляет историю через посещаемость
- Отмена атомарно создаёт `attendance = cancelled` с причиной `parent`/`master` и выполняет расчёт списания по правилам
- ⚠️ Не откладывать платную отмену на ручную отметку

> 📌 До начала занятия мастер может использовать `DELETE /bookings/:id` с `cancel_reason = 'master'`. После начала занятия мастер исправляет историю через `PUT /master/lessons/:lessonId/attendance/:bookingId` со `status = 'cancelled'` и `cancel_reason = 'master'`.

---

### `POST /bookings/:id/transfer`

**Доступ:** `parent` | `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{ "target_lesson_id": "uuid" }
```

**Ответ 201:**
```json
{
  "old_booking_id": "uuid",
  "new_booking_id": "uuid",
  "charge_applied": 0
}
```

**Условия:**
- До начала исходного занятия
- В той же группе
- Условия и deadline — из правил
- Блокировать ребёнка и оба занятия, проверить целевое место и конфликты
- Затем в одной транзакции: отменить старую запись и создать новую
- При ошибке — ничего не менять

> 📌 Плата/списание за перенос вычисляется по `transfer_charge_lesson`, а не повторно по правилу обычной отмены. Уведомить семью. Старую запись сохранить в истории.

---

### `GET /families/me/bookings`

**Доступ:** `parent`

**Query:** `upcoming?`, `from?`, `to?`, `child_id?`, `status?`, `limit?`, `offset?`

**items:**
```json
{
  "id": "uuid",
  "child_id": "uuid",
  "lesson_id": "uuid",
  "group_title": "...",
  "starts_at": "RFC3339",
  "ends_at": "RFC3339",
  "status": "active",
  "attendance": "pending",
  "version": 1
}
```

---

### `GET /families/me/attendance`

**Доступ:** `parent`

**Query:** `child_id?`, `from?`, `to?`, `limit?`, `offset?`

**items:** семейная история, включая отменённые записи.
```json
{
  "booking_id": "uuid",
  "lesson_id": "uuid",
  "starts_at": "RFC3339",
  "status": "attended",
  "cancel_reason": null,
  "charged": true,
  "subscription_id": "uuid",
  "invoice_id": "uuid",
  "marked_at": "RFC3339"
}
```

---

### `GET /master/lessons/:lessonId/bookings`

**Доступ:** `master`

**Query:** `status?`, `limit?`, `offset?`

**items:**
```json
{
  "id": "uuid",
  "lesson_id": "uuid",
  "child_id": "uuid",
  "child_full_name": "...",
  "family_id": "uuid",
  "family_parent_full_name": "...",
  "status": "active",
  "attendance": {
    "status": "pending",
    "cancel_reason": null,
    "marked_at": null
  },
  "subscription": {
    "id": "uuid|null",
    "lessons_remaining": 3
  },
  "version": 1
}
```

---

## 5. Посещаемость и правила

**Статусы `attendance`:**

| Статус | Описание |
|--------|----------|
| `pending` | Ожидает отметки (не означает занятого места) |
| `attended` | Посещено |
| `missed` | Пропущено |
| `cancelled` | Отменено |

**`cancel_reason`:** `parent` | `master` | `transfer`

- Отметки `attended`/`missed` доступны после `starts_at`
- Одна `attendance` на `booking_id`

---

### `PUT /master/lessons/:lessonId/attendance/:bookingId`

**Доступ:** `master`

**Body:**
```json
{
  "status": "attended",
  "cancel_reason": null,
  "comment": "...",
  "expected_version": 1
}
```

> 📌 При исправлении существующей отметки `comment` **обязателен**. `booking` принадлежит `lesson`.

**Ответ 200:**
```json
{
  "attendance_id": "uuid",
  "status": "attended",
  "charged": true,
  "subscription_id": "uuid",
  "lessons_remaining": 3,
  "invoice_id": "uuid",
  "version": 2
}
```

**Алгоритм в транзакции:**
1. Блокировать запись, отметку и затронутые абонементы/счета
2. Рассчитать новый эффект
3. Сравнить с **фактическим старым движением** (не с текущим правилом старого статуса)
4. Если эффект не изменился — новых списаний нет
5. Сначала компенсировать прежний эффект при необходимости, затем применить новый

**Правила списания:**
- Возврат занятия — **всегда в исходный абонемент**, даже истёкший; срок не продлевается
- Списывать из подходящего по группе и дате занятия абонемента с остатком > 0, сначала с ближайшим `valid_until`, при равенстве — по ID
- Остаток не может стать < 0
- Нет подходящего абонемента → создать один разовый счёт по цене из сохранённого правила; посещение всё равно отмечается, возникает задолженность
- Уже оплаченный разовый счёт данной записи покрывает посещение **без списания ещё и из абонемента**
- Нельзя одновременно иметь действующий разовый счёт и списание занятия для одного эффекта посещаемости
- Купленный позже абонемент **не погашает** старый долг автоматически
- При отмене списания по разовому счёту — компенсирующая денежная запись; если счёт оплачен, появляется кредит семьи
- При новом платном состоянии оплаченной записи используется её прежняя оплата через **обратную компенсацию**, не новый платёж

> ⚠️ `pending` не означает занятого места: восстановление записи требует `POST /bookings`.
> Историческая правка посещаемости **не меняет** `capacity` и статус бронирования.

---

### `GET /master/attendance/:bookingId/history`

**Доступ:** `master`

**Query:** `limit?`, `offset?`

**items:**
```json
{
  "version": 2,
  "before": { "status": "pending" },
  "after": { "status": "attended" },
  "comment": "...",
  "actor": "uuid",
  "created_at": "RFC3339",
  "rule_version": 1,
  "subscription_ledger_entry_ids": ["uuid"],
  "money_ledger_entry_ids": []
}
```

---

### `GET /master/attendance-rules`

**Доступ:** `master`

**Query:** `group_id?`, `date?`, `limit?`, `offset?`

**items:** объекты версий правил.
```json
{
  "id": "uuid",
  "group_id": "uuid|null",
  "valid_from": "YYYY-MM-DD",
  "valid_to": "YYYY-MM-DD|null",
  "attended_charge_lesson": true,
  "missed_charge_lesson": true,
  "parent_cancel_charge_lesson": true,
  "free_cancel_before_hours": 24,
  "parent_cancel_allowed": true,
  "cancel_deadline_hours": 24,
  "transfer_allowed": true,
  "transfer_deadline_hours": 24,
  "transfer_charge_lesson": false,
  "single_lesson_price_minor": 80000,
  "ending_notice_days": 3,
  "ending_notice_lessons": 2,
  "version": 1,
  "applied_at": "RFC3339|null",
  "created_at": "RFC3339"
}
```

---

### `GET /master/attendance-rules/:id`

**Доступ:** `master`

**Ответ 200:** такой же объект, как в списке.

---

### `POST /master/attendance-rules`

**Доступ:** `master`

**Body:**
```json
{
  "group_id": "uuid|null",
  "valid_from": "date",
  "valid_to": "date|null",
  "attended_charge_lesson": true,
  "missed_charge_lesson": true,
  "parent_cancel_charge_lesson": true,
  "free_cancel_before_hours": 24,
  "parent_cancel_allowed": true,
  "cancel_deadline_hours": 24,
  "transfer_allowed": true,
  "transfer_deadline_hours": 24,
  "transfer_charge_lesson": false,
  "single_lesson_price_minor": 80000,
  "ending_notice_days": 3,
  "ending_notice_lessons": 2
}
```

- Часы/пороги ≥ 0, цена > 0
- Для одного `scope` не может быть пересекающихся периодов, включая scope по умолчанию (`group_id = null`)
- Правило группы приоритетнее общего

---

### `PATCH /master/attendance-rules/:id`

**Доступ:** `master`

**Body:** `{ expected_version, ...поля создания }`

> ⚠️ Только пока версия не применялась. Применимая версия **неизменяема**; новые условия создаются с будущей даты, границы периодов закрываются атомарно, без изменения сохранённых снимков.

---

### `DELETE /master/attendance-rules/:id`

**Доступ:** `master`

- `204` для неприменённой версии
- Применённая → `409 RULE_IN_USE`

> 📌 Для изменения действующей политики создавать **преемника**, а не удалять историю.

---

### 🧭 Логика применения правил

- Выбор версии — по **дате занятия**; при записи сохраняется снимок условий
- Нет подходящих правил → `409 RULES_NOT_CONFIGURED`; не применять скрытые значения
- Перенос времени занятия не меняет принятые условия, но обновляет `starts_at`; срок отмены считается от **актуального времени занятия** и фактического `cancelled_at`, никогда от момента, когда мастер поставил отметку
- Если `hours_before ≥ free_cancel_before_hours` → отмена родителя бесплатна; иначе действует `parent_cancel_charge_lesson`
- Отмена мастером — **всегда без списания**
- Для новых записей применяются новые версии; исправление истории использует снимок
- Денежные штрафы сверх стоимости занятия **не входят в MVP**

---

## 6. Тарифы и абонементы

### `GET /master/plans`

**Доступ:** `master`

**items:**
```json
{
  "id": "uuid",
  "group_id": "uuid",
  "title": "...",
  "lessons_total": 8,
  "validity_days": 30,
  "price_minor": 400000,
  "currency": "RUB",
  "is_active": true,
  "version": 1
}
```

---

### `POST /master/plans`

**Доступ:** `master`

**Body:**
```json
{
  "title": "...",
  "group_id": "uuid",
  "lessons_total": 8,
  "validity_days": 30,
  "price_minor": 400000
}
```

- Число занятий, дней и цена > 0
- `currency = "RUB"`

---

### `PATCH /master/plans/:id`

**Доступ:** `master`

**Body:**
```json
{
  "expected_version": 1,
  "title": "...",
  "lessons_total": 8,
  "validity_days": 30,
  "price_minor": 400000,
  "is_active": true
}
```

> 📌 Изменения создают новую версию; уже выставленные счета сохраняют снимок тарифа.

---

### `GET /families/me/subscriptions`

**Доступ:** `parent`

**Query:** `child_id?`, `status?`, `limit?`, `offset?`

**items:** объект абонемента (схема аналогична `GET /master/children/:id/subscriptions`).

---

### `GET /master/children/:id/subscriptions`

**Доступ:** `master`

**Query:** `status?`, `limit?`, `offset?`

**Объект:**
```json
{
  "id": "uuid",
  "child_id": "uuid",
  "group_id": "uuid",
  "invoice_id": "uuid",
  "lessons_total": 8,
  "lessons_remaining": 5,
  "adjustments_total": 0,
  "valid_from": "date",
  "valid_until": "date",
  "status": "active",
  "version": 1
}
```

**Статусы:**

| Статус | Описание |
|--------|----------|
| `scheduled` | Ожидает активации |
| `active` | Действует |
| `expired` | Истёк по сроку |
| `exhausted` | Исчерпан по занятиям |

> 📌 Оплаченные занятия и ручные добавления показываются раздельно. Статус вычисляется на чтении по сроку и остатку.

---

### `GET /subscriptions/:id/ledger`

**Доступ:** `parent` (свой) | `master`

**Query:** `from?`, `to?`, `reason?`, `limit?`, `offset?`

**items:**
```json
{
  "id": "uuid",
  "delta": -1,
  "reason": "attendance_charge",
  "attendance_id": "uuid",
  "payment_id": null,
  "reverses_entry_id": null,
  "comment": "...",
  "created_at": "RFC3339"
}
```

---

### `POST /master/subscriptions/:id/adjustments`

**Доступ:** `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{ "delta": 1, "comment": "Обязательный комментарий" }
```

**Ответ 201:**
```json
{ "ledger_entry_id": "uuid", "lessons_remaining": 6 }
```

- `delta` — ненулевой `int`
- `comment` — обязательный `string`
- Отрицательный итог → `409`

> ⚠️ Только служебная корректировка, **не способ** регистрации продажи/оплаты.

---

### 🎟 Создание абонемента

Абонемент создаётся **автоматически** при полной оплате счёта (раздел 8), один на `invoice_id`.

> 🚫 Отдельного `POST /master/subscriptions` для выдачи оплаченных занятий без оплаты **нет**.

---

### 📒 `subscription_ledger`

Неизменяемые движения:

| Тип | Описание |
|-----|----------|
| `+N` | `purchase` (покупка) |
| `-1` | `attendance_charge` (списание за посещение) |
| `+1` | `attendance_reversal` (возврат) |
| `+/-` | `manual_adjustment` (ручная корректировка) |

- Компенсация ссылается на оригинал
- **Источник истины** — `SUM(delta)`
- Если хранится кеш `lessons_remaining`, обновлять **только одним механизмом** (триггер `INSERT ledger`), не дублировать `UPDATE` в обработчике
- ⚠️ Нельзя заменять транзакционное обновление периодической сверкой

---

## 7. Счета, денежный баланс и задолженность

> 📌 Счёт фиксирует назначение покупки и сумму **ДО оплаты**. Оплата привязана к счёту, счёт — к семье и ребёнку. Тариф, сумма, число занятий и срок — серверный снимок.

---

### `POST /invoices`

**Доступ:** `parent` | `master` · 🔑 **Idempotency-Key**

**Body для абонемента:**
```json
{
  "child_id": "uuid",
  "kind": "subscription",
  "plan_id": "uuid",
  "valid_from": "date"
}
```

**Body для разового занятия:**
```json
{
  "child_id": "uuid",
  "kind": "single_lesson",
  "booking_id": "uuid"
}
```

- `valid_until = valid_from + validity_days - 1` (даты включительно)
- Для разового счёта — только своя активная запись, цена из снимка правила; не создавать, если посещение уже покрыто абонементом
- Дубль открытого/оплаченного разового счёта → `409 INVOICE_EXISTS` с доступным своему пользователю ID

**Ответ 201:**
```json
{
  "id": "uuid",
  "child_id": "uuid",
  "kind": "subscription",
  "amount_minor": 400000,
  "currency": "RUB",
  "status": "open",
  "due_minor": 400000,
  "plan_snapshot": { "..." },
  "booking_id": null,
  "valid_from": "date",
  "valid_until": "date",
  "version": 1
}
```

---

### `GET /invoices/:id`

**Доступ:** `parent` (свой) | `master`

---

### `GET /families/me/invoices`

**Доступ:** `parent`

**Query:** `status?`, `child_id?`, `limit?`, `offset?`

**items:**
```json
{
  "id": "uuid",
  "family_id": "uuid",
  "child_id": "uuid",
  "kind": "subscription|single_lesson",
  "status": "open|paid|cancelled",
  "amount_minor": 400000,
  "due_minor": 400000,
  "currency": "RUB",
  "plan_id": "uuid|null",
  "booking_id": "uuid|null",
  "valid_from": "YYYY-MM-DD|null",
  "valid_until": "YYYY-MM-DD|null",
  "version": 1,
  "created_at": "RFC3339"
}
```

---

### `GET /master/invoices`

**Доступ:** `master`

**Query:** `family_id?`, `child_id?`, `status?`, `limit?`, `offset?`

**items:** аналогично `GET /families/me/invoices`.

---

### `POST /master/invoices/:id/cancel`

**Доступ:** `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{ "reason": "...", "expected_version": 1 }
```

**Ответ 200:** объект счёта.

- Только неоплаченный счёт без незавершённой попытки оплаты; иначе `409`
- Создать компенсирующее движение, **не удалять** счёт
- Оплаченные исправления посещаемости компенсируются через раздел 5; автоматического возврата денег нет

---

### 💵 `money_ledger`

Неизменяемые записи с полями: `family_id`, `invoice_id`, `payment_id?`, `amount_minor` (со знаком), `reason`, `source_id`, `reverses_entry_id?`, `created_at`.

| Событие | Движение |
|---------|----------|
| Выставление счёта | `-` стоимость |
| Подтверждённая оплата | `+` сумма |
| Отмена начисления | `+` стоимость |
| Отмена компенсации | обратная запись |

**Баланс:**
- `balance_minor = SUM(money_ledger.amount_minor)`
  - Положительный = **кредит семьи**
  - Отрицательный = **долг**
- `debt_minor = SUM(max(0, остаток к оплате по каждому счёту))`
- Кредит по одному счёту **не скрывает** долг по другому
- Нет автоматического взаимозачёта

**Пример:**
```
Счёт:     -400000
Оплата:   +400000
──────────────────
Баланс:   0        → абонемент: 4 занятия
Посещение:         → абонемент: 3 занятия, баланс по-прежнему 0
```

---

### `GET /families/me/balance`

**Доступ:** `parent`

**Ответ 200:**
```json
{
  "balance_minor": 0,
  "debt_minor": 0,
  "currency": "RUB"
}
```

---

### `GET /master/families/:id/balance`

**Доступ:** `master`

**Ответ 200:**
```json
{ "balance_minor": 0, "debt_minor": 0, "currency": "RUB" }
```

---

### `GET /families/me/balance/ledger`

**Доступ:** `parent`

**items:**
```json
{
  "id": "uuid",
  "amount_minor": -400000,
  "reason": "invoice_issued",
  "invoice_id": "uuid",
  "payment_id": "uuid|null",
  "reverses_entry_id": "uuid|null",
  "created_at": "RFC3339"
}
```

---

### `GET /master/families/:id/balance/ledger`

**Доступ:** `master`

**Query:** `from?`, `to?`, `limit?`, `offset?`

**items:** денежные движения с причиной и счётом (схема аналогична `GET /families/me/balance/ledger`).

---

## 8. Платежи и ЮKassa

### `POST /payments/intent`

**Доступ:** `parent` (свой) | `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{ "invoice_id": "uuid", "method": "online|sbp" }
```

- Сумма/число занятий от клиента **НЕ принимаются**
- Только открытый неоплаченный счёт; сумма = `due_minor` из БД
- На счёт допускается одна незавершённая попытка
- Повтор возвращает её URL
- Смена метода, пока попытка не завершена → `409 PAYMENT_IN_PROGRESS`
- Сохранить попытку и стабильный ключ провайдера **ДО внешнего запроса**
- После timeout повторить запрос с тем же ключом, не создавать новую попытку

**Ответ 201:**
```json
{
  "payment_id": "uuid",
  "status": "pending",
  "amount_minor": 400000,
  "currency": "RUB",
  "payment_url": "https://..."
}
```

> 📌 ЮKassa — провайдер, СБП — способ оплаты (`payment_method_data.type = sbp`).
> Для СБП использовать `redirect confirmation` и `capture = true`; фронтенд открывает `confirmation_url`.
> ⚠️ Не обещать обязательный `qr_code` в ответе этого API.
> `return_url` задаётся сервером, не произвольным адресом клиента.

---

### `GET /payments/:id`

**Доступ:** `parent` (свой) | `master`

**Ответ 200:**
```json
{
  "id": "uuid",
  "invoice_id": "uuid",
  "child_id": "uuid",
  "amount_minor": 400000,
  "currency": "RUB",
  "method": "online",
  "status": "succeeded",
  "paid_at": "RFC3339",
  "receipt": { "id": "uuid" }
}
```

---

### `GET /families/me/payments`

**Доступ:** `parent`

**Query:** `from?`, `to?`, `child_id?`, `status?`, `limit?`, `offset?`

**items:**
```json
{
  "id": "uuid",
  "invoice_id": "uuid",
  "child_id": "uuid",
  "amount_minor": 400000,
  "currency": "RUB",
  "method": "online",
  "status": "succeeded",
  "paid_at": "RFC3339|null",
  "receipt": { "id": "uuid" }
}
```
> `receipt` равен `null`, если чек ещё не прикреплён.

---

### `GET /master/payments`

**Доступ:** `master`

**Query:** `from?`, `to?`, `family_id?`, `child_id?`, `status?`, `method?`, `limit?`, `offset?`

**items:** аналогично `GET /families/me/payments`.

---

### `POST /master/payments`

**Доступ:** `master` · 🔑 **Idempotency-Key**

**Body:**
```json
{
  "invoice_id": "uuid",
  "method": "cash|transfer",
  "paid_at": "RFC3339",
  "comment": "..."
}
```

- Сумма = полный непогашенный остаток счёта; платёж сразу `succeeded`
- При незавершённой онлайн-попытке → `409`; сначала сверить её результат
- Та же транзакционная процедура зачисления, что и для онлайн-платежа

**Ответ 201:** объект платежа; повтор не создаёт дубликат.

---

### `POST /webhooks/payments/yookassa`

**Доступ:** провайдер, без cookie

Принимать документированный JSON `event/object`.

> ⚠️ **НЕ требовать** выдуманный `X-Provider-Signature`: в выбранном протоколе проверять объект запросом к API ЮKassa с серверными реквизитами; проверку IP использовать как дополнительный барьер.

**Алгоритм обработки:**
1. По `object.id` запросить актуальный платёж
2. Проверить `provider_payment_id`, сумму, `RUB`, магазин/получателя и test-режим
3. `metadata` **не является доказательством**
4. Привязку по `metadata` разрешать только после сверки с сохранённой попыткой, своим магазином, суммой и назначением, **атомарно закрепляя уникальный provider ID** (нужно для гонки, когда webhook приходит раньше ответа на создание платежа)

**В одной транзакции:**
- Заблокировать платёж и счёт
- Зафиксировать актуальный статус
- При первом `succeeded`: добавить `+amount_minor` в `money_ledger`
- Закрыть оплаченный счёт
- Для абонемента: создать `subscriptions` и `+N` в `subscription_ledger` **ОДИН раз**
- Добавить аудит и уведомление в транзакционную очередь доставки (outbox)
- Для разового занятия: счёт покрывает посещение, абонемент не создаётся

**Уникальности:**
- `(provider, provider_payment_id)`
- Денежное зачисление на `payment_id`
- `subscriptions.invoice_id`
- `purchase` на `subscription_id`

> ⚠️ Проверки статуса без блокировки и этих ограничений **недостаточны** при параллельных webhook.

**Поведение при повторах:**
- Повтор/запоздавшее событие **не откатывает** `succeeded` в `pending`/`canceled`
- Если подтверждён второй реальный платёж — сохранить как **переплату** и уведомить мастера, но **не начислять** второй абонемент. Не терять факт поступления денег

**Ответы:**
- `200 { ok: true }` — после успешной фиксации или подтверждённого повтора
- Неизвестное событие — безопасно сохранить диагностический факт и вернуть `200` без начисления
- Битый JSON → `400`; несоответствие суммы/магазина → `400` и сигнал мастеру
- Сбой API/БД или временно неизвестная попытка → `503` для повтора, **не фиктивный 200**

**Статусы прототипа:** `pending → succeeded | canceled`; сетевой timeout оставляет `pending`.

> 🚫 Денежные возвраты — отдельные операции будущего этапа, не смена `succeeded` на `refunded`.
> Не подключать реальные оплаты до реализации сверки возвратов и согласования ограничений.

---

## 9. Чеки

### `POST /master/payments/:id/receipts`

**Доступ:** `master`

**Формат:** `multipart/form-data` — ровно одно из `file` или `external_link`

- Только для `succeeded`
- PDF / JPEG / PNG ≤ 10 MiB; проверять содержимое, а не лишь расширение
- `external_link` — HTTPS из настроенного списка допустимых доменов сервиса чеков
- Сервер **не скачивает** произвольный URL
- Один текущий чек на платёж, дубль → `409`

**Ответ 201:**
```json
{
  "id": "uuid",
  "payment_id": "uuid",
  "kind": "file|link",
  "download_url": "/api/v1/receipts/:id/download",
  "external_link": "https://...",
  "uploaded_at": "RFC3339"
}
```

---

### `PUT /master/payments/:id/receipts`

**Доступ:** `master`

**Формат:** `multipart` — `file` или `external_link`, `comment`, `expected_version`

Замена ошибочного чека с сохранением истории, старого файла и автора изменения.

---

### `GET /receipts/:id`

**Доступ:** `parent` (свой) | `master`

**Ответ 200:** метаданные чека, включая `version` (для `PUT` с `expected_version`); для файла `download_url = /api/v1/receipts/:id/download`.

---

### `GET /receipts/:id/download`

**Доступ:** `parent` (свой) | `master`

**Для файла:**
- `200` байты
- `Content-Type` корректный
- Безопасное имя в `Content-Disposition`
- `Cache-Control: private, no-store`
- `X-Content-Type-Options: nosniff`

**Для ссылки:**
- `409 RECEIPT_IS_EXTERNAL`
- Интерфейс открывает `external_link` отдельно

> 🔒 Файлы хранятся вне публичного каталога. Не выдавать постоянные публичные `file_url`.
> Для скачивания внутри приложения мастер прикрепляет файл; скачивание по внешней ссылке зависит от возможностей сервиса, куда она ведёт.

---

## 10. Уведомления и аудит

### `GET /notifications`

**Доступ:** `parent` | `master`

**Query:** `unread?`, `limit?`, `offset?`

Только текущего пользователя.

**Ответ 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "type": "booking.confirmed",
      "payload": { "lesson_id": "uuid", "child_id": "uuid" },
      "read_at": "RFC3339|null",
      "created_at": "RFC3339"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0,
  "unread_count": 3
}
```
> `unread_count` — по всем уведомлениям пользователя.

---

### `POST /notifications/:id/read`

**Доступ:** `parent` | `master`

`204`; повтор → `204`, чужое → `404`.

---

### `POST /notifications/read-all`

**Доступ:** `parent` | `master`

**Ответ 200:** `{ "updated": 5 }`. Только текущий пользователь.

---

### 📣 Типы уведомлений

| Тип | Описание |
|-----|----------|
| `booking.confirmed` | Запись подтверждена |
| `booking.cancelled` | Запись отменена |
| `booking.transferred` | Запись перенесена |
| `lesson.changed` | Занятие изменено |
| `lesson.cancelled` | Занятие отменено |
| `subscription.ending` | Абонемент скоро истекает |
| `subscription.exhausted` | Абонемент исчерпан |
| `subscription.expired` | Абонемент истёк |
| `payment.required` | Требуется оплата |
| `payment.succeeded` | Оплата прошла |
| `receipt.attached` | Чек прикреплён |

> 📌 Уникальный ключ события/получателя защищает от дублей при повторе фоновой задачи.

---

### `GET /master/audit-log`

**Доступ:** `master`

**Query:** `resource_type?`, `resource_id?`, `actor_user_id?`, `event_type?`, `from?`, `to?`, `limit?`, `offset?`

**items:**
```json
{
  "id": 12345,
  "event_type": "booking.created",
  "actor_user_id": "uuid",
  "actor_role": "parent",
  "resource_type": "booking",
  "resource_id": "uuid",
  "before": { "..." },
  "after": { "..." },
  "reason": "...",
  "request_id": "...",
  "created_at": "RFC3339"
}
```
> 📌 `id` — целое число (`bigserial` в БД).
> 📌 Для фоновых событий `actor_role = system`.

**Что аудировать:** правила, доступ, семьи, записи, посещения, счета, деньги, чеки, корректировки.

> 🚫 Не сохранять пароли, сырые токены, cookie и платёжные секреты.

---

## 11. CSV и эксплуатация

### `GET /master/export/children.csv`

**Доступ:** `master`

**Колонки:** `id`, `full_name`, `birth_date`, `family_id`, `parent_full_name`, `parent_phone`, `parent_email`

---

### `GET /master/export/attendance.csv`

**Доступ:** `master`

**Query:** `from`, `to` (обязательны), `group_id?`

**Колонки:** `booking_id`, `lesson_id`, `starts_at`, `child_id`, `child_full_name`, `status`, `cancel_reason`, `charged`, `subscription_id`, `invoice_id`, `marked_at`

---

### `GET /master/export/payments.csv`

**Доступ:** `master`

**Query:** `from`, `to` (обязательны), `status?`, `method?`

**Колонки:** `id`, `invoice_id`, `child_id`, `family_id`, `amount_minor`, `currency`, `method`, `status`, `paid_at`, `provider_payment_id`, `receipt_id`

---

### `GET /master/export/all.zip`

**Доступ:** `master`

**Query:** `from?`, `to?`

- Если отсутствуют оба — все данные; иначе требуются оба
- Архив из трёх CSV; дети — все, посещения/оплаты — за диапазон
- Параметры `from`/`to` — даты в зоне `Asia/Yekaterinburg`, включительно, как в остальных списках

**Требования к CSV:**
- UTF-8, заголовки колонок
- Корректное экранирование кавычек/переводов строк
- Текстовые ячейки, начинающиеся с `=`, `+`, `-`, `@`, табуляции/CR — **обезвреживать для Excel**
- Числа экспортировать как числа
- `Content-Disposition: attachment`; потоковая отдача

---

### ⚙️ Фоновые задачи внутри бэкенда

> Четвёртый сервис не обязателен.

- Напоминания о сроке/остатке по настройкам, о неоплаченных счетах
- Доставка outbox, повтор после сбоя без дублей
- Сверка `pending` с ЮKassa, повтор создания с прежним ключом при потере ответа
- Контроль инвариантов денежных движений и остатков **без скрытого исправления истории**
- Очистка истёкших сессий (отзыв проверяется сразу, а не только по расписанию)

> 📌 Резервное копирование БД и файлов чеков — задача эксплуатации, не публичный endpoint. Периодичность, хранение вне сервера и проверка восстановления должны быть настроены **перед передачей системы**; команды для текущей заготовки приведены в README.

---

## 12. Безопасность и проверки приёмки

### 🛡 Базовые требования

- HTTPS
- Разрешённые `Origin`
- CSRF
- Лимиты запросов и размера тела, в том числе webhook
- При перегрузке webhook — `429`/`503` с повтором, **не потеря события**
- Не доверять `X-Forwarded-For` от произвольного клиента; список доверенных прокси задаёт сервер
- Секреты — только серверные
- Демо использует вымышленные семьи и test-платежи
- Для ответов кабинета, ссылок доступа, платёжных данных — `Cache-Control: no-store`

### 🩺 Проверка работоспособности (Health-checks)

- `GET /healthz` (доступ: guest). `200 { "status": "ok" }` — процесс жив.
- `GET /readyz` (доступ: guest). `200 { "status": "ready", "db": "ok" }` — БД доступна и миграции применены. `503` с описанием проблемы иначе.
- Оба эндпоинта не требуют cookie и не пишут в аудит. Используются хостингом и внешним мониторингом.

---

### ✅ Сценарии приёмки

| # | Сценарий | Ожидаемый результат |
|---|----------|---------------------|
| 1 | Семья с двумя детьми, группа `capacity = 2`, запись без email | Успех |
| 2 | Счёт на 4 занятия → тестовый `succeeded` | Остаток 4, деньги 0 после оплаты |
| 3 | `attended` → остаток 3 → прикрепление и скачивание чека родителем | Успех |
| 4 | Два конкурентных запроса на последнее место | Один успех, другой `409` |
| 5 | Повтор ребёнка на занятии | `409`; отмена и новая запись возможны при наличии места |
| 6 | Повтор/параллельный webhook | Одно зачисление, один абонемент, остаток всё ещё 3 |
| 7 | `attended` → бесплатная отмена → возврат 1 | Успех |
| 8 | Повтор исправления посещаемости | Не меняет остаток |
| 9 | Смена `attended` → `missed` при обоих платных состояниях | Не возвращает занятие |
| 10 | Поздняя отмена | Использует `cancelled_at`; смена правила не переписывает прошлое |
| 11 | Посещение без абонемента | Фиксируется с разовым долгом; наличная оплата гасит его |
| 12 | Отзыв ссылки | Блокирует и старую ссылку, и все её сессии |
| 13 | Чужие UUID в чтении/записи/скачивании/CSV | Не раскрывают данные |
| 14 | Отмена занятия мастером | Компенсирует списания и уведомляет все затронутые семьи |
| 15 | Выгрузка CSV, уведомления, резервная копия и проверенное восстановление | Успех |

---

## 13. Дополнительный этап (не обязательный MVP)

> ⚠️ **Не выдавать предварительные маршруты за работающие функции.**

### 📋 Лист ожидания (предварительные маршруты)

- `GET /families/me/waitlist`
- `POST /waitlist`
- `DELETE /waitlist/:id`
- `POST /waitlist/:id/accept`

> 📌 Перед реализацией отдельно согласовать срок предложения и резервирование места; `waiting`/`offered` не должны допускать двойную запись или превышение вместимости.

### 🔮 Отдельные задачи будущего

- Отработка пропуска
- Импорт «МойКласс»
- Мессенджеры
- Страница мастерской

---

## 🔗 Источники для интеграции

> Проверены 2026-11-14.

- [Уведомления ЮKassa](https://yookassa.ru/developers/using-api/webhooks)
- [СБП ЮKassa](https://yookassa.ru/developers/payment-acceptance/integration-scenarios/manual-integration/other/sbp)