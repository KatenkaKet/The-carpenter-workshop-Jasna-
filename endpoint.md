# REST API — Система записи и учёта занятий «Ясна»

## Дополнения: рабочий сценарий, общие тарифы и CSV

- `GET /public/payment-options` → `{test_enabled: boolean}`.
- `POST /payments/test-confirm` → тело `{invoice_id}`, обязательны CSRF и `Idempotency-Key`, роль родитель/мастер. Только `APP_ENV=development` и `PAYMENT_STUB=true`. Возвращает успешный платёж `method=test`, HTTP 201. Повтор того же ключа возвращает исходный результат, новый ключ для оплаченного счёта даёт 409. В выключенном режиме — 404.
- `GET /master/booking-policy` → общие правила и `version` (0 до первого сохранения).
- `PUT /master/booking-policy` → `{expected_version, missed_charge_lesson, free_cancel_before_hours, transfer_allowed}`. Часы — целое 0–8760. Глобальная настройка действует сразу для следующих операций; имеет приоритет над старыми настройками групп. Старые групповые endpoints сохранены для совместимости.
- `POST /master/plans`: `group_id` теперь необязателен. Без него тариф универсален; UI не предлагает группы. Старые оплаченные абонементы сохраняют область действия.
- `POST /bookings`: `funding_kind=subscription|single_lesson|awaiting_payment|legacy`. `awaiting_payment` допускает предварительную запись ребёнка без прежнего подходящего абонемента. После подтверждения покупки доступные занятия распределяются по таким записям хронологически в одной транзакции. `single_lesson` остаётся отдельной оплатой.
- Списки абонементов возвращают `plan_title`, `lessons_reserved`, `lessons_used`, `lessons_available`, `adjustments_total` и статус `suspended` при аннулировании. `lessons_remaining` — неиспользованный остаток, включая резерв.
- `POST /master/subscription-purchases`: `{child_id, plan_id, valid_from, method: cash|transfer, comment?}`, CSRF + Idempotency-Key. Счёт, успешная оплата, абонемент и покрытие записей создаются атомарно.
- `GET /bookings/:id/cancellation`: `allowed`, `free_cancel_until`, `will_charge`, `timezone`, `server_time`, реальные правила. Окончательное списание решается сервером при DELETE.
- `POST /master/payments/:id/void`: `{expected_version, reason}`, CSRF + Idempotency-Key. Поддерживает cash/transfer/test, сохраняет историю и использованные занятия; внешний платёж требует возврата у провайдера.
- `DELETE /master/receipts/:id`: `{expected_version, reason}`, CSRF. Чек закрывается для доступа, история сохраняется.
- `GET /master/export/children.csv`, `attendance.csv`, `payments.csv`, `all.zip`: период теперь необязателен, без него выгружается вся история. Только мастер.

Эти дополнения имеют приоритет над прежними примерами ниже. [Сценарий проверки и ограничения](docs/demo-and-export.md).


Версия: 1.2, дополнения от 2026-11-15. Обязательные маршруты реализованы в Go + Gin.
Инструкции запуска, проверки и ограничения: [backend/README.md](backend/README.md).

Уточнения реализации: изменения сериализуются общей транзакционной блокировкой PostgreSQL;
чеки хранятся в закрытой таблице БД; расчёт с незавершённой попыткой оплаты нельзя
компенсировать до её завершения. Онлайн-платежи разрешены только в тестовом режиме.
Базовый путь: /api/v1. JSON UTF-8; исключения — файлы и CSV.
Исправленная редакция исходного `endpoint.txt`; [причины изменений](docs/endpoint-review.md).

## 0. ОБЩИЕ СОГЛАШЕНИЯ

Роли: guest, parent, master. /public/* доступен всем; /master/* — только master.
Семья родителя определяется из серверной сессии. Для child_id, booking_id,
subscription_id, invoice_id, payment_id, receipt_id всегда проверять принадлежность.
UUID не является средством авторизации. Чужой объект возвращает 404 без раскрытия
его существования; неверная роль — 403; нет действующей сессии — 401.

ID: UUID. Даты: YYYY-MM-DD. Время: RFC 3339 с зоной, хранение в UTC.
Бизнес-зона: Asia/Yekaterinburg. Диапазон from/to в списках — даты включительно
в этой зоне, реализуемые как [начало from, начало следующего дня после to).
Деньги: amount_minor и прочие `*_minor` — целые копейки (int64), currency="RUB".
Не использовать float для расчётов. Например, 400000 = 4000 рублей.

Списки: limit 1..200 (default 50), offset >= 0 (default 0), стабильная сортировка
created_at,id, если не указано другое. Ответ: {items,total,limit,offset}.
Неизвестные поля запроса отклоняются с 422. PATCH — только перечисленные поля.
Скрытые поля, token_hash, password_hash, ключи провайдера никогда не возвращаются.

Успех: GET/PATCH/PUT — 200, создание — 201, удаление/отзыв — 204, если не указано иное.
Ошибки: 400 JSON, 401 сессия, 403 роль/CSRF, 404 объект, 409 конфликт,
413 размер файла, 422 валидация, 429 лимит (Retry-After), 503 временный сбой.
Формат: {"error":{"code":"SEAT_LIMIT_EXCEEDED","message":"Нет свободных мест",
"details":{"capacity":2}},"request_id":"..."}.
Каждый ответ содержит X-Request-Id. Все успешные изменения сохраняют аудит.

Idempotency-Key обязателен для POST с денежными операциями, счетами, корректировками,
созданием записи и переносом. Ключ сохраняется с пользователем, маршрутом, хешем тела
и результатом атомарно с операцией. Повтор с тем же телом возвращает исходный ответ,
с другим телом — 409 IDEMPOTENCY_CONFLICT. Проверка доступа выполняется и при повторе.
Ключ запроса к провайдеру — отдельный, стабильно связанный с попыткой оплаты.

Изменяемые объекты содержат version. Для PATCH и PUT отметки требуется
expected_version; несовпадение — 409 VERSION_CONFLICT. Повтор PUT с текущей версией
и тем же состоянием ничего не списывает. История и финансовые движения неизменяемы.

### Проектные допущения

Не готовые функции и не утверждённые правила мастерской.

- мастер создаёт семью и передаёт ссылку вручную; самостоятельная регистрация новых
  семей не включена, существующая семья записывает детей самостоятельно;
- для мастера предлагается отдельный логин/пароль, созданный при установке;
- абонемент принадлежит одному ребёнку и группе, срок фиксируется в счёте;
- продажи полными счетами; частичные оплаты, распределение переплаты и автоматические
  возвраты — следующий этап. Переплата учитывается, но не активирует второй абонемент;
- новая покупка — отдельный абонемент, а не изменение оплаченного старого.

## 1. ДОСТУП И СЕССИИ

Родитель входит по персональной ссылке, без обязательного email и пароля.
Мастер выдаёт ссылку вида `https://<host>/access#token=<secret>`. Secret: 32 случайных
байта из crypto/rand Go, base64url; в БД хранится только SHA-256 хеш.
Ссылка повторно используемая до expires_at или отзыва. Это осознанная модель
постоянного доступа, а не одноразовая ссылка на email. Срок задаёт мастер.
Фронтенд читает fragment, сразу удаляет его через history.replaceState и отправляет
secret в POST body. На странице входа нет сторонних скриптов/аналитики.
Не писать токен в логи, аудит, URL запросов или localStorage.

### `POST /auth/access-link/exchange`

**Доступ:** guest.

  **Body:** {token: string}. 200: {role:"parent",family_id,csrf_token} + Set-Cookie.
  Проверить срок/отзыв; создать серверную сессию с access_link_id и family_id.
  Неверная, истёкшая или отозванная ссылка: 401 INVALID_ACCESS_LINK (единый ответ).

### `POST /auth/master/login`

**Доступ:** guest.

  **Body:** {login,password}. 200: {role:"master",csrf_token} + отдельная cookie.
  Учётная запись provisioned при установке; публичной регистрации мастера нет.
  Пароль хранить как Argon2id-хеш с солью, не как открытый пароль в конфигурации.
  401 INVALID_CREDENTIALS; лимит попыток по IP и логину. Вход родителя не может
  выдавать роль master. 2FA может быть добавлена отдельно.

### `GET /auth/me`

**Доступ:** parent|master.

  200: {id,role,full_name,family_id,csrf_token}; family_id=null для мастера.
  Если присутствуют обе cookie, вернуть 409 SESSION_CONTEXT_CONFLICT;
  интерфейс предлагает выйти или открыть отдельный профиль браузера.

### `POST /auth/logout`

**Доступ:** guest|parent|master.

  204. Отозвать предъявленные сессии и очистить обе cookie. Повтор — 204.

### `GET /master/families/:id/access-links`

**Доступ:** master.

  Список {id,created_at,expires_at,revoked_at}; секрет не возвращается.

### `POST /master/families/:id/access-links`

**Доступ:** master.

  **Body:** {expires_at:datetime}. 201: {id,url,expires_at}; секрет только в этом ответе.
  Новый выпуск атомарно отзывает старые ссылки семьи и все их сессии.
  Срок должен быть в будущем. Утерянную ссылку нельзя восстановить — только заменить.

### `DELETE /master/families/:id/access-links/:linkId`

**Доступ:** master.

  204: отзыв ссылки и всех её сессий в одной транзакции; повтор — 204.

Сессии: случайные opaque-токены, в БД хеши, role, user_id, family_id,
access_link_id, expires_at, revoked_at. JWT не нужен.
Cookie: yasna_parent_session и yasna_master_session; HttpOnly; Secure;
SameSite=Lax; Path=/; без Domain. Срок родителя <=30 дней и не дольше ссылки;
мастера <=12 часов. Проверять отзыв на каждом запросе, включая скачивание чеков.
Изменяющие запросы с cookie требуют Origin из разрешённого списка и X-CSRF-Token.
Login/exchange проверяют Origin, принимают только JSON и ограничены по частоте;
webhook не использует cookie/CSRF, его проверка описана в разделе 8.

## 2. ПУБЛИЧНОЕ РАСПИСАНИЕ

### `GET /public/groups`

  **Query:** is_individual?, limit?, offset?. Только активные группы.

  **items:** {id,title,description,capacity,is_individual}.

### `GET /public/groups/:id`

  200: {id,title,description,capacity,is_individual}; неактивная группа — 404.

### `GET /public/schedule`

  **Query:** from,to (обязательны), group_id?, limit?, offset?; диапазон <=93 дней.

  **items:** {lesson_id,group_id,group_title,starts_at,ends_at,status,capacity,
  seats_taken,seats_available,is_individual}. Сортировка starts_at,id.
  Никаких имён, детей, записей и контактов в публичном ответе.

### `GET /lessons/:id/availability`

**Доступ:** parent|master.

  200: {lesson_id,status,capacity,seats_taken,seats_available,version}.
  Доступность — справочная; окончательная проверка происходит при записи.

### `GET /public/plans`

  **Query:** group_id?, limit?, offset?. items: {id,title,group_id,lessons_total,
  validity_days,price_minor,currency,version}; только активные тарифы.

### `GET /public/booking-rules`

  **Query:** group_id?, date (обязательно).
  200: действующие условия отмены, переноса и списания для показа до записи.

## 3. СЕМЬИ И ДЕТИ

### `GET /families/me`

**Доступ:** parent.

  200: {id,parent:{id,full_name,phone,email},children:[{id,full_name,birth_date}],
  financial_summary:{balance_minor,debt_minor,currency}}.
  phone/email необязательны и не определяют доступ.

### `GET /master/families`

**Доступ:** master.

  **Query:** search?, limit?, offset?. items: {id,parent_full_name,phone,email,children_count}.

### `GET /master/families/:id`

**Доступ:** master.

  200: семья с детьми и financial_summary, без секретов доступа.

### `POST /master/families`

**Доступ:** master.

  **Body:** {parent_full_name,phone?,email?,children:[{full_name,birth_date?}]}.
  Минимум один ребёнок; всё создаётся атомарно. Email не обязателен и не уникальный ключ.
  201: объект семьи. Выпуск ссылки — отдельная операция раздела 1.

### `PATCH /master/families/:id`

**Доступ:** master.

  **Body:** {expected_version,parent_full_name?,phone?,email?}.

### `POST /master/families/:id/children`

**Доступ:** master.

  **Body:** {full_name,birth_date?}. 201: ребёнок.

### `GET /master/children`

**Доступ:** master.

  **Query:** search?, family_id?, limit?, offset?. items: {id,family_id,full_name,birth_date}.

### `GET /master/children/:id`

**Доступ:** master.

  200: {id,family_id,full_name,birth_date,version}.

### `PATCH /master/children/:id`

**Доступ:** master.

  **Body:** {expected_version,full_name?,birth_date?,is_active?}. Архивация сохраняет историю;
  при будущих активных записях — 409, сначала отменить/перенести их.

## 4. ГРУППЫ, ЗАНЯТИЯ И ЗАПИСЬ

### `GET /master/groups`

**Доступ:** master.

  **Query:** is_active?,search?,limit?,offset?. items: объекты групп с version.

### `POST /master/groups`

**Доступ:** master.

  **Body:** {title,capacity,is_individual?,description?}. capacity>0; индивидуальное =1.

### `PATCH /master/groups/:id`

**Доступ:** master.

  **Body:** {expected_version,title?,capacity?,description?,is_active?}.
  capacity группы — шаблон для новых занятий; существующие меняются явно.
  Архивация при будущих занятиях scheduled — 409; сначала отменить их.

### `GET /master/lessons`

**Доступ:** master.

  **Query:** from?,to?,group_id?,status?,limit?,offset?. Сортировка starts_at,id.

### `GET /master/lessons/:id`

**Доступ:** master.

  200: {id,group_id,starts_at,ends_at,capacity,seats_taken,status,notes,version}.

### `POST /master/lessons`

**Доступ:** master.

  **Body:** {group_id,starts_at,ends_at,capacity?,notes?}.
  Будущее время, ends_at>starts_at; capacity по умолчанию из группы.
  Индивидуальное занятие capacity=1. Для одного мастера пересечения scheduled-занятий
  запрещены даже между разными группами; проверка защищена от конкурентного создания.

### `PATCH /master/lessons/:id`

**Доступ:** master.

  **Body:** {expected_version,starts_at?,ends_at?,capacity?,status?,cancel_reason?,notes?}.
  status: scheduled|completed|cancelled. Переход из completed/cancelled обратно запрещён.
  completed возможен после ends_at и обработки всех pending-отметок активных записей.
  Исторические отметки completed можно исправлять через раздел 5.
  Уменьшение capacity ниже активных записей — 409. Изменение времени повторяет
  проверки создания и проверяет пересечения у записанных детей; иначе 409.
  Перенос времени сохраняет записи и уведомляет семьи. При отмене мастером —
  cancel_reason обязателен; все записи отменяются, все списания этого занятия
  компенсируются, включая прежние отмены родителями, создаются уведомления и аудит.
  Оплаченные разовые счета при компенсации дают кредит семьи, не автоматический возврат.

### `POST /bookings`

**Доступ:** parent|master; Idempotency-Key.

  **Body:** {lesson_id,child_id}. 201: {id,lesson_id,child_id,status:"active",version}.
  Занятие scheduled и ещё не началось; ребёнок активен и принадлежит семье.
  Транзакция блокирует строку ребёнка и строку занятия; проверяет доступ,
  вместимость и пересечение времени с активными записями этого ребёнка.
  UNIQUE(lesson_id,child_id) WHERE status='active' защищает от дублей,
  но позволяет заново записаться после отмены с новым booking_id.
  Все отмены, переносы, изменения capacity используют тот же протокол блокировок.
  Единый порядок блокировок (дети по ID, занятия по ID) и повтор транзакции при deadlock.
  409 SEAT_LIMIT_EXCEEDED|DUPLICATE_BOOKING|CHILD_TIME_CONFLICT|LESSON_UNAVAILABLE.
  Отсутствие абонемента не запрещает запись; деньги учитываются по разделам 5–8.

### `DELETE /bookings/:id`

**Доступ:** parent|master.

  204, мягкая отмена; повтор отмены своей записи — 204.
  Сохранить cancelled_at, cancelled_by и причину. Проверить deadline по правилам.
  Отмена после начала родителем — 409; мастер исправляет историю через посещаемость.
  Отмена атомарно создаёт attendance=cancelled с причиной parent/master и выполняет
  расчёт списания по правилам. Не откладывать платную отмену на ручную отметку.

### `POST /bookings/:id/transfer`

**Доступ:** parent|master; Idempotency-Key.

  **Body:** {target_lesson_id}. 201: {old_booking_id,new_booking_id,charge_applied}.
  До начала исходного занятия, в той же группе; условия и deadline — из правил.
  Блокировать ребёнка и оба занятия, проверить целевое место и конфликты, затем
  в одной транзакции отменить старую запись и создать новую. При ошибке ничего не менять.
  Плата/списание за перенос вычисляется по transfer_charge_lesson, а не повторно
  по правилу обычной отмены. Уведомить семью. Старую запись сохранить в истории.

### `GET /families/me/bookings`

**Доступ:** parent.

  **Query:** upcoming?,from?,to?,child_id?,status?,limit?,offset?.

  **items:** {id,child_id,lesson_id,group_title,starts_at,ends_at,status,attendance,version}.

### `GET /families/me/attendance`

**Доступ:** parent.

  **Query:** child_id?,from?,to?,limit?,offset?.

  **items:** {booking_id,lesson_id,starts_at,status,cancel_reason,charged,subscription_id,
  invoice_id,marked_at}; семейная история, включая отменённые записи.

### `GET /master/lessons/:lessonId/bookings`

**Доступ:** master.

  **Query:** status?,limit?,offset?. items: запись, ребёнок, семья, attendance, остаток.

## 5. ПОСЕЩАЕМОСТЬ И ПРАВИЛА

attendance.status: pending|attended|missed|cancelled; cancel_reason: parent|master|transfer.
Отметки attended/missed доступны после starts_at. Одна attendance на booking_id.

### `PUT /master/lessons/:lessonId/attendance/:bookingId`

**Доступ:** master.

  **Body:** {status,cancel_reason?,comment?,expected_version}.
  При исправлении существующей отметки comment обязателен; booking принадлежит lesson.
  200: {attendance_id,status,charged,subscription_id,lessons_remaining,invoice_id,version}.
  В транзакции блокировать запись, отметку и затронутые абонементы/счета.
  Рассчитать новый эффект, сравнить с ФАКТИЧЕСКИМ старым движением, а не с текущим
  правилом старого статуса. Если эффект не изменился — новых списаний нет.
  Сначала компенсировать прежний эффект при необходимости, затем применить новый.
  Возврат занятия всегда в исходный абонемент, даже истёкший; срок не продлевается.
  Списывать из подходящего по группе и дате занятия абонемента с остатком >0,
  сначала с ближайшим valid_until, при равенстве по ID. Остаток не может стать <0.
  Нет подходящего абонемента: создать один разовый счёт по цене из сохранённого правила;
  посещение всё равно отмечается, возникает задолженность. Уже оплаченный разовый
  счёт данной записи покрывает посещение без списания ещё и из абонемента.
  Нельзя одновременно иметь действующий разовый счёт и списание занятия для одного
  эффекта посещаемости. Купленный позже абонемент не погашает старый долг автоматически.
  При отмене списания по разовому счёту — компенсирующая денежная запись;
  если счёт оплачен, появляется кредит семьи. При новом платном состоянии оплаченной
  записи используется её прежняя оплата через обратную компенсацию, не новый платёж.
  pending не означает занятого места: восстановление записи требует POST /bookings.
  Историческая правка посещаемости не меняет capacity и статус бронирования.

### `GET /master/attendance/:bookingId/history`

**Доступ:** master.

  **Query:** limit?,offset?. items: {version,before,after,comment,actor,created_at,
  rule_version,subscription_ledger_entry_ids,money_ledger_entry_ids}.

### `GET /master/attendance-rules`

**Доступ:** master.

  **Query:** group_id?,date?,limit?,offset?. Объекты версий правил.

### `GET /master/attendance-rules/:id`

**Доступ:** master.

### `POST /master/attendance-rules`

**Доступ:** master.

  **Body:** {group_id:null|uuid,valid_from,valid_to?,attended_charge_lesson,
  missed_charge_lesson,parent_cancel_charge_lesson,free_cancel_before_hours,
  parent_cancel_allowed,cancel_deadline_hours,transfer_allowed,
  transfer_deadline_hours,transfer_charge_lesson,single_lesson_price_minor,
  ending_notice_days,ending_notice_lessons}.
  Часы/пороги >=0, цена>0. Для одного scope не может быть пересекающихся периодов,
  включая scope по умолчанию с group_id=null. Правило группы приоритетнее общего.

### `PATCH /master/attendance-rules/:id`

**Доступ:** master.

  **Body:** {expected_version,...поля создания}. Только пока версия не применялась.
  Применённая версия неизменяема; новые условия создаются с будущей даты,
  границы периодов закрываются атомарно, без изменения сохранённых снимков.

### `DELETE /master/attendance-rules/:id`

**Доступ:** master.

  204 для неприменённой версии; применённая — 409 RULE_IN_USE.
  Для изменения действующей политики создавать преемника, а не удалять историю.

Выбор версии — по дате занятия; при записи сохраняется снимок условий.
Нет подходящих правил — 409 RULES_NOT_CONFIGURED; не применять скрытые значения.
Перенос времени занятия не меняет принятые условия, но обновляет starts_at;
срок отмены считается от актуального времени занятия и фактического cancelled_at,
никогда от момента, когда мастер поставил отметку.
Если hours_before >= free_cancel_before_hours, отмена родителя бесплатна;
иначе действует parent_cancel_charge_lesson. Отмена мастером всегда без списания.
Для новых записей применяются новые версии; исправление истории использует снимок.
Денежные штрафы сверх стоимости занятия не входят в MVP.

## 6. ТАРИФЫ И АБОНЕМЕНТЫ

### `GET /master/plans`

**Доступ:** master.

### `POST /master/plans`

**Доступ:** master.

  **Body:** {title,group_id,lessons_total,validity_days,price_minor}.
  Число занятий, дней и цена >0; currency=RUB.

### `PATCH /master/plans/:id`

**Доступ:** master.

  **Body:** {expected_version,title?,lessons_total?,validity_days?,price_minor?,is_active?}.
  Изменения создают новую версию, уже выставленные счета сохраняют снимок тарифа.

### `GET /families/me/subscriptions`

**Доступ:** parent.

  **Query:** child_id?,status?,limit?,offset?.

### `GET /master/children/:id/subscriptions`

**Доступ:** master.

  **Query:** status?,limit?,offset?.
  Объект: {id,child_id,group_id,invoice_id,lessons_total,lessons_remaining,
  adjustments_total,valid_from,valid_until,status,version}.
  status: scheduled|active|expired|exhausted. Оплаченные занятия и ручные добавления
  показываются раздельно. Статус вычисляется на чтении по сроку и остатку.

### `GET /subscriptions/:id/ledger`

**Доступ:** parent свой|master.

  **Query:** from?,to?,reason?,limit?,offset?.

  **items:** {id,delta,reason,attendance_id,payment_id,reverses_entry_id,comment,created_at}.

### `POST /master/subscriptions/:id/adjustments`

**Доступ:** master; Idempotency-Key.

  **Body:** {delta:int ненулевой,comment:string обязательный}.
  201: {ledger_entry_id,lessons_remaining}. Отрицательный итог — 409.
  Только служебная корректировка, не способ регистрации продажи/оплаты.

Абонемент создаётся АВТОМАТИЧЕСКИ при полной оплате счёта (раздел 8), один на invoice_id.
Отдельного POST /master/subscriptions для выдачи оплаченных занятий без оплаты нет.
subscription_ledger: неизменяемые движения +N purchase, -1 attendance_charge,
+1 attendance_reversal, +/- manual_adjustment. Компенсация ссылается на оригинал.
Источник истины — SUM(delta). Если хранится кеш lessons_remaining, обновлять только
одним механизмом (триггер INSERT ledger), не дублировать UPDATE в обработчике.
Нельзя заменять транзакционное обновление периодической сверкой.

## 7. СЧЕТА, ДЕНЕЖНЫЙ БАЛАНС И ЗАДОЛЖЕННОСТЬ

Счёт фиксирует назначение покупки и сумму ДО оплаты. Оплата привязана к счёту,
счёт — к семье и ребёнку. Тариф, сумма, число занятий и срок — серверный снимок.

### `POST /invoices`

**Доступ:** parent|master; Idempotency-Key.

  **Body subscription:** {child_id,kind:"subscription",plan_id,valid_from}.

  **Body single_lesson:** {child_id,kind:"single_lesson",booking_id}.
  valid_until=valid_from+validity_days-1 (даты включительно).
  Для разового счёта — только своя активная запись, цена из снимка правила;
  не создавать если посещение уже покрыто абонементом. Дубль открытого/оплаченного
  разового счёта возвращает 409 INVOICE_EXISTS с доступным своему пользователю ID.
  201: {id,child_id,kind,amount_minor,currency,status:"open",due_minor,
  plan_snapshot,booking_id,valid_from,valid_until,version}.

### `GET /invoices/:id`

**Доступ:** parent свой|master.

### `GET /families/me/invoices`

**Доступ:** parent.

  **Query:** status?,child_id?,limit?,offset?.

### `GET /master/invoices`

**Доступ:** master.

  **Query:** family_id?,child_id?,status?,limit?,offset?.

### `POST /master/invoices/:id/cancel`

**Доступ:** master; Idempotency-Key.

  **Body:** {reason,expected_version}. 200: счёт.
  Только неоплаченный счёт без незавершённой попытки оплаты; иначе 409.
  Создать компенсирующее движение, не удалять счёт. Оплаченные исправления
  посещаемости компенсируются через раздел 5, автоматического возврата денег нет.

money_ledger: неизменяемые записи с family_id, invoice_id, payment_id?,
amount_minor (со знаком), reason, source_id, reverses_entry_id?, created_at.
Выставление счёта: минус стоимость; подтверждённая оплата: плюс сумма;
отмена начисления: плюс стоимость; отмена компенсации: обратная запись.
balance_minor = SUM(money_ledger.amount_minor): положительный = кредит семьи,
отрицательный = долг. debt_minor = SUM(max(0,остаток к оплате по каждому счёту));
кредит по одному счёту не скрывает долг по другому. Нет автоматического взаимозачёта.
Пример: счёт -400000, оплата +400000 → денежный баланс 0 и абонемент 4 занятия.
Посещение → 3 занятия, денежный баланс по-прежнему 0.

### `GET /families/me/balance`

**Доступ:** parent.

### `GET /master/families/:id/balance`

**Доступ:** master.

  200: {balance_minor,debt_minor,currency:"RUB"}.

### `GET /families/me/balance/ledger`

**Доступ:** parent.

### `GET /master/families/:id/balance/ledger`

**Доступ:** master.

  **Query:** from?,to?,limit?,offset?. items: денежные движения с причиной и счётом.

## 8. ПЛАТЕЖИ И ЮKASSA

### `POST /payments/intent`

**Доступ:** parent свой|master; Idempotency-Key.

  **Body:** {invoice_id,method:"online"|"sbp"}. Сумма/число занятий от клиента НЕ принимаются.
  Только открытый неоплаченный счёт; сумма = due_minor из БД.
  На счёт допускается одна незавершённая попытка. Повтор возвращает её URL;
  смена метода пока попытка не завершена — 409 PAYMENT_IN_PROGRESS.
  Сохранить попытку и стабильный ключ провайдера ДО внешнего запроса.
  После timeout повторить запрос с тем же ключом, не создавать новую попытку.
  201: {payment_id,status:"pending",amount_minor,currency,payment_url}.
  ЮKassa — провайдер, СБП — способ оплаты payment_method_data.type=sbp.
  Для СБП использовать redirect confirmation и capture=true; фронтенд открывает
  confirmation_url. Не обещать обязательный qr_code в ответе этого API.
  return_url задаётся сервером, не произвольным адресом клиента.

### `GET /payments/:id`

**Доступ:** parent свой|master.

  200: {id,invoice_id,child_id,amount_minor,currency,method,status,paid_at,receipt}.

### `GET /families/me/payments`

**Доступ:** parent.

  **Query:** from?,to?,child_id?,status?,limit?,offset?.

### `GET /master/payments`

**Доступ:** master.

  **Query:** from?,to?,family_id?,child_id?,status?,method?,limit?,offset?.

### `POST /master/payments`

**Доступ:** master; Idempotency-Key.

  **Body:** {invoice_id,method:"cash"|"transfer",paid_at?,comment?}.
  Сумма = полный непогашенный остаток счёта; платёж сразу succeeded.
  При незавершённой онлайн-попытке — 409, сначала сверить её результат.
  Та же транзакционная процедура зачисления, что для онлайн-платежа.
  201: объект платежа; повтор не создаёт дубликат.

### `POST /webhooks/payments/yookassa`

**Доступ:** провайдер, без cookie.

  Принимать документированный JSON event/object. НЕ требовать выдуманный
  X-Provider-Signature: в выбранном протоколе проверять объект запросом к API ЮKassa
  с серверными реквизитами; проверку IP использовать как дополнительный барьер.
  По object.id запросить актуальный платёж. Проверить provider_payment_id,
  сумму, RUB, магазин/получателя и test-режим; metadata не является доказательством.
  Привязку по metadata разрешать только после сверки с сохранённой попыткой,
  своим магазином, суммой и назначением, атомарно закрепляя уникальный provider ID.
  Это нужно для гонки, когда webhook приходит раньше ответа на создание платежа.

  В одной транзакции: заблокировать платёж и счёт; зафиксировать актуальный статус;
  при первом succeeded добавить +amount_minor в money_ledger; закрыть оплаченный счёт;
  для абонемента создать subscriptions и +N в subscription_ledger ОДИН раз;
  добавить аудит и уведомление в транзакционную очередь доставки (outbox).
  Для разового занятия счёт покрывает посещение, абонемент не создаётся.
  Уникальности: (provider,provider_payment_id), денежное зачисление на payment_id,
  subscriptions.invoice_id, purchase на subscription_id. Проверки статуса без
  блокировки и этих ограничений недостаточно при параллельных webhook.
  Повтор/запоздавшее событие не откатывает succeeded в pending/canceled.
  Если подтверждён второй реальный платёж, сохранить его как переплату и уведомить
  мастера, но не начислять второй абонемент. Не терять факт поступления денег.

  200 {ok:true} — после успешной фиксации или подтверждённого повтора.
  Неизвестное событие: безопасно сохранить диагностический факт и вернуть 200 без начисления.
  Битый JSON — 400; несоответствие суммы/магазина — 400 и сигнал мастеру.
  Сбой API/БД или временно неизвестная попытка — 503 для повтора, не фиктивный 200.
  Статусы прототипа: pending → succeeded|canceled; сетевой timeout оставляет pending.
  Денежные возвраты — отдельные операции будущего этапа, не смена succeeded на refunded.
  Не подключать реальные оплаты до реализации сверки возвратов и согласования ограничений.

## 9. ЧЕКИ

### `POST /master/payments/:id/receipts`

**Доступ:** master.

  **multipart/form-data:** ровно одно из file или external_link.
  Только для succeeded. PDF/JPEG/PNG <=10 MiB; проверять содержимое, а не лишь расширение.
  external_link — HTTPS из настроенного списка допустимых доменов сервиса чеков.
  Сервер не скачивает произвольный URL. Один текущий чек на платёж, дубль — 409.
  201: {id,payment_id,kind:"file"|"link",download_url,external_link,uploaded_at}.

### `PUT /master/payments/:id/receipts`

**Доступ:** master.

  **multipart:** file или external_link, comment, expected_version.
  Замена ошибочного чека с сохранением истории, старого файла и автора изменения.

### `GET /receipts/:id`

**Доступ:** parent свой|master.

  200: метаданные чека; для файла download_url=/api/v1/receipts/:id/download.

### `GET /receipts/:id/download`

**Доступ:** parent свой|master.

  Для файла: 200 байты, Content-Type, безопасное имя в Content-Disposition,
  Cache-Control: private,no-store, X-Content-Type-Options: nosniff.
  Для ссылки: 409 RECEIPT_IS_EXTERNAL; интерфейс открывает external_link отдельно.
  Файлы хранятся вне публичного каталога. Не выдавать постоянные публичные file_url.
  Для скачивания внутри приложения мастер прикрепляет файл; скачивание по внешней
  ссылке зависит от возможностей сервиса, куда она ведёт.

## 10. УВЕДОМЛЕНИЯ И АУДИТ

### `GET /notifications`

**Доступ:** parent|master.

  **Query:** unread?,limit?,offset?. Только текущего пользователя.
  200: {items,total,limit,offset,unread_count}; unread_count по всем его уведомлениям.

### `POST /notifications/:id/read`

**Доступ:** parent|master.

  204; повтор — 204, чужое — 404.

### `POST /notifications/read-all`

**Доступ:** parent|master.

  200: {updated}. Только текущий пользователь.
Типы: booking.confirmed, booking.cancelled, booking.transferred, lesson.changed,
lesson.cancelled, subscription.ending, subscription.exhausted, subscription.expired,
payment.required, payment.succeeded, receipt.attached.
Уникальный ключ события/получателя защищает от дублей при повторе фоновой задачи.

### `GET /master/audit-log`

**Доступ:** master.

  **Query:** resource_type?,resource_id?,actor_user_id?,event_type?,from?,to?,limit?,offset?.

  **items:** {id,event_type,actor_user_id,actor_role,resource_type,resource_id,before,after,
  reason,request_id,created_at}. Для фоновых событий actor_role=system.
  Аудировать правила, доступ, семьи, записи, посещения, счета, деньги, чеки и корректировки.
  Не сохранять пароли, сырые токены, cookie и платёжные секреты.

## 11. CSV И ЭКСПЛУАТАЦИЯ

### `GET /master/export/children.csv`

**Доступ:** master.

  **Колонки:** id,full_name,birth_date,family_id,parent_full_name,parent_phone,parent_email.

### `GET /master/export/attendance.csv`

**Доступ:** master.

  **Query:** from,to обязательны; group_id?.

  **Колонки:** booking_id,lesson_id,starts_at,child_id,child_full_name,status,
  cancel_reason,charged,subscription_id,invoice_id,marked_at.

### `GET /master/export/payments.csv`

**Доступ:** master.

  **Query:** from,to обязательны; status?,method?.

  **Колонки:** id,invoice_id,child_id,family_id,amount_minor,currency,method,status,
  paid_at,provider_payment_id,receipt_id.

### `GET /master/export/all.zip`

**Доступ:** master.

  **Query:** from?,to?. Если отсутствуют оба — все данные; иначе требуются оба.
  Архив из трёх CSV; дети — все, посещения/оплаты — за диапазон.
CSV: UTF-8, заголовки колонок, корректное экранирование кавычек/переводов строк.
Текстовые ячейки, начинающиеся с =,+,-,@, табуляции/CR, обезвреживать для Excel;
числа экспортировать как числа. Content-Disposition: attachment; потоковая отдача.

Фоновые задачи внутри бэкенда (четвёртый сервис не обязателен):
- напоминания о сроке/остатке по настройкам, о неоплаченных счетах;
- доставка outbox, повтор после сбоя без дублей;
- сверка pending с ЮKassa, повтор создания с прежним ключом при потере ответа;
- контроль инвариантов денежных движений и остатков без скрытого исправления истории;
- очистка истёкших сессий. Отзыв проверяется сразу, а не только по расписанию.
Резервное копирование БД и файлов чеков — задача эксплуатации, не публичный endpoint.
Периодичность, хранение вне сервера и проверка восстановления должны быть настроены
перед передачей системы; команды для текущей заготовки приведены в README.

## 12. БЕЗОПАСНОСТЬ И ПРОВЕРКИ ПРИЁМКИ

HTTPS; разрешённые Origin; CSRF; лимиты запросов и размера тела, в том числе webhook.
При перегрузке webhook — 429/503 с повтором, не потеря события. Не доверять
X-Forwarded-For от произвольного клиента; список доверенных прокси задаёт сервер.
Секреты — только серверные. Демо использует вымышленные семьи и test-платежи.
Для ответов кабинета, ссылок доступа, платёжных данных — Cache-Control: no-store.

Проверить:
1. семья с двумя детьми, группа capacity=2, запись без email;
2. счёт на 4 занятия → тестовый succeeded → остаток 4, деньги 0 после оплаты счёта;
3. attended → остаток 3 → прикрепление и скачивание чека родителем;
4. два конкурентных запроса на последнее место: один успех, другой 409;
5. повтор ребёнка на занятии — 409; отмена и новая запись возможны при наличии места;
6. повтор/параллельный webhook: одно зачисление, один абонемент, остаток всё ещё 3;
7. attended → бесплатная отмена → возврат 1; повтор исправления не меняет остаток;
8. смена attended → missed при обоих платных состояниях не возвращает занятие;
9. поздняя отмена использует cancelled_at; смена правила не переписывает прошлое;
10. посещение без абонемента фиксируется с разовым долгом; наличная оплата гасит его;
11. отзыв ссылки блокирует и старую ссылку, и все её сессии;
12. чужие UUID в чтении/записи/скачивании/CSV не раскрывают данные;
13. отмена занятия мастером компенсирует списания и уведомляет все затронутые семьи;
14. выгрузка CSV, уведомления, резервная копия и проверенное восстановление.

## 13. ДОПОЛНИТЕЛЬНЫЙ ЭТАП (НЕ ОБЯЗАТЕЛЬНЫЙ MVP)

Предварительные маршруты листа ожидания: GET /families/me/waitlist,
POST /waitlist, DELETE /waitlist/:id, POST /waitlist/:id/accept.
Перед реализацией отдельно согласовать срок предложения и резервирование места;
waiting/offered не должны допускать двойную запись или превышение вместимости.
Отработка пропуска, импорт «МойКласс», мессенджеры и страница мастерской — отдельные
задачи. Не выдавать предварительные маршруты за работающие функции.

## Источники для интеграции

Проверены 2026-11-14.

- [Уведомления](https://yookassa.ru/developers/using-api/webhooks)
- [СБП](https://yookassa.ru/developers/payment-acceptance/integration-scenarios/manual-integration/other/sbp)
