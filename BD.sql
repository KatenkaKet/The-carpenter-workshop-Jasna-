CREATE TYPE "user_role" AS ENUM (
  'master',
  'parent'
);

CREATE TYPE "lesson_status" AS ENUM (
  'scheduled',
  'cancelled',
  'completed'
);

CREATE TYPE "booking_status" AS ENUM (
  'active',
  'cancelled_by_parent',
  'cancelled_by_master'
);

CREATE TYPE "attendance_status" AS ENUM (
  'pending',
  'attended',
  'missed',
  'cancelled_by_master'
);

CREATE TYPE "attendance_rule_status" AS ENUM (
  'attended',
  'missed',
  'cancelled_by_master',
  'cancelled_by_parent'
);

CREATE TYPE "subscription_status" AS ENUM (
  'active',
  'expired',
  'exhausted',
  'frozen'
);

CREATE TYPE "ledger_reason" AS ENUM (
  'attendance_charge',
  'attendance_reversal',
  'manual_adjustment',
  'subscription_purchase'
);

CREATE TYPE "payment_method" AS ENUM (
  'cash',
  'transfer',
  'online_yookassa',
  'online_sbp'
);

CREATE TYPE "payment_purpose" AS ENUM (
  'subscription',
  'single_lesson'
);

CREATE TYPE "payment_status" AS ENUM (
  'pending',
  'succeeded',
  'failed',
  'refunded'
);

CREATE TYPE "waitlist_status" AS ENUM (
  'waiting',
  'offered',
  'converted',
  'expired',
  'cancelled'
);

CREATE TABLE "users" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "role" user_role NOT NULL,
  "email" text UNIQUE NOT NULL,
  "full_name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "access_tokens" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid,
  "role" user_role NOT NULL,
  "token_hash" bytea UNIQUE NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "consumed_at" timestamptz,
  "revoked_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "sessions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "token_hash" bytea UNIQUE NOT NULL,
  "user_id" uuid NOT NULL,
  "role" user_role NOT NULL,
  "family_id" uuid,
  "access_token_id" uuid,
  "expires_at" timestamptz NOT NULL,
  "revoked_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "last_seen_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "families" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "parent_user_id" uuid UNIQUE NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "children" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "family_id" uuid NOT NULL,
  "full_name" text NOT NULL,
  "birth_date" date,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "groups" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "master_id" uuid NOT NULL,
  "title" text NOT NULL,
  "capacity" int NOT NULL,
  "is_individual" boolean NOT NULL DEFAULT false,
  "is_active" boolean NOT NULL DEFAULT true,
  "description" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "lessons" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "group_id" uuid NOT NULL,
  "master_id" uuid NOT NULL,
  "starts_at" timestamptz NOT NULL,
  "ends_at" timestamptz NOT NULL,
  "status" lesson_status NOT NULL DEFAULT 'scheduled',
  "cancel_reason" text,
  "notes" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "attendance_rules" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "group_id" uuid,
  "status" attendance_rule_status NOT NULL,
  "charge_lesson" boolean NOT NULL,
  "charge_before_hours" int,
  "valid_from" date NOT NULL,
  "valid_to" date,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "bookings" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "lesson_id" uuid NOT NULL,
  "child_id" uuid NOT NULL,
  "status" booking_status NOT NULL DEFAULT 'active',
  "cancelled_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "attendance" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "booking_id" uuid UNIQUE NOT NULL,
  "status" attendance_status NOT NULL DEFAULT 'pending',
  "previous_status" attendance_status,
  "marked_by" uuid,
  "marked_at" timestamptz,
  "comment" text,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "subscriptions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "child_id" uuid NOT NULL,
  "lessons_total" int NOT NULL,
  "lessons_remaining" int NOT NULL,
  "valid_from" date NOT NULL,
  "valid_until" date NOT NULL,
  "status" subscription_status NOT NULL DEFAULT 'active',
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "subscription_ledger" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "subscription_id" uuid NOT NULL,
  "delta" int NOT NULL,
  "reason" ledger_reason NOT NULL,
  "attendance_id" uuid,
  "created_by" uuid NOT NULL,
  "comment" text,
  "idempotency_key" text UNIQUE,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "payments" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "family_id" uuid NOT NULL,
  "child_id" uuid,
  "amount" numeric(12,2) NOT NULL,
  "currency" varchar(3) NOT NULL DEFAULT 'RUB',
  "method" payment_method NOT NULL,
  "purpose" payment_purpose NOT NULL,
  "status" payment_status NOT NULL DEFAULT 'pending',
  "provider_payment_id" text UNIQUE,
  "idempotency_key" text UNIQUE,
  "paid_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "receipts" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "payment_id" uuid UNIQUE NOT NULL,
  "file_url" text,
  "external_link" text,
  "uploaded_by" uuid NOT NULL,
  "uploaded_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "waitlist_entries" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "lesson_id" uuid NOT NULL,
  "child_id" uuid NOT NULL,
  "family_id" uuid NOT NULL,
  "status" waitlist_status NOT NULL DEFAULT 'waiting',
  "position" int NOT NULL,
  "offered_until" timestamptz,
  "created_by" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now()),
  "updated_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "notifications" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "user_id" uuid NOT NULL,
  "family_id" uuid,
  "type" text NOT NULL,
  "payload" jsonb NOT NULL DEFAULT ('{}'::jsonb),
  "read_at" timestamptz,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "audit_log" (
  "id" BIGINT GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
  "event_type" text NOT NULL,
  "actor_user_id" uuid,
  "actor_role" user_role,
  "resource_type" text NOT NULL,
  "resource_id" uuid,
  "changes" jsonb,
  "metadata" jsonb,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "users" ("role");

CREATE INDEX ON "access_tokens" ("user_id");

CREATE INDEX ON "access_tokens" ("expires_at");

CREATE INDEX "idx_access_tokens_user" ON "access_tokens" ("user_id", "created_at");

CREATE INDEX ON "sessions" ("user_id");

CREATE INDEX ON "sessions" ("access_token_id");

CREATE INDEX ON "sessions" ("expires_at");

CREATE INDEX ON "children" ("family_id");

CREATE INDEX ON "children" ("full_name");

CREATE INDEX ON "lessons" ("group_id", "starts_at");

CREATE INDEX ON "lessons" ("starts_at");

CREATE INDEX ON "lessons" ("status");

CREATE INDEX ON "attendance_rules" ("group_id", "status", "valid_from");

CREATE UNIQUE INDEX "uq_bookings_lesson_child" ON "bookings" ("lesson_id", "child_id");

CREATE INDEX ON "bookings" ("lesson_id", "status");

CREATE INDEX ON "bookings" ("child_id");

CREATE INDEX ON "subscriptions" ("child_id", "status");

CREATE INDEX ON "subscriptions" ("valid_until");

CREATE INDEX ON "subscription_ledger" ("subscription_id", "created_at");

CREATE INDEX ON "subscription_ledger" ("attendance_id");

CREATE INDEX ON "payments" ("family_id", "created_at");

CREATE INDEX ON "payments" ("child_id");

CREATE INDEX ON "payments" ("status");

CREATE INDEX ON "waitlist_entries" ("lesson_id", "status", "position");

CREATE INDEX "idx_waitlist_unique_active" ON "waitlist_entries" ("lesson_id", "child_id", "status");

CREATE INDEX ON "waitlist_entries" ("child_id");

CREATE INDEX ON "notifications" ("user_id", "read_at");

CREATE INDEX ON "notifications" ("created_at");

CREATE INDEX ON "audit_log" ("resource_type", "resource_id", "created_at");

CREATE INDEX ON "audit_log" ("actor_user_id", "created_at");

CREATE INDEX ON "audit_log" ("event_type", "created_at");

COMMENT ON COLUMN "users"."email" IS 'хранить в lower-case';

COMMENT ON COLUMN "groups"."capacity" IS 'CHECK capacity > 0';

COMMENT ON COLUMN "lessons"."ends_at" IS 'CHECK ends_at > starts_at';

COMMENT ON COLUMN "attendance_rules"."group_id" IS 'NULL = правило по умолчанию';

COMMENT ON COLUMN "attendance_rules"."charge_before_hours" IS 'CHECK >= 0; NULL = всегда';

COMMENT ON COLUMN "attendance_rules"."valid_to" IS 'CHECK valid_to >= valid_from';

COMMENT ON COLUMN "subscriptions"."lessons_total" IS 'CHECK lessons_total > 0';

COMMENT ON COLUMN "subscriptions"."lessons_remaining" IS 'CHECK >= 0; кеш из ledger';

COMMENT ON COLUMN "subscriptions"."valid_until" IS 'CHECK valid_until >= valid_from';

COMMENT ON COLUMN "subscription_ledger"."delta" IS 'CHECK delta <> 0';

COMMENT ON COLUMN "subscription_ledger"."idempotency_key" IS 'защита от дублей';

COMMENT ON COLUMN "payments"."amount" IS 'CHECK amount > 0';

COMMENT ON COLUMN "payments"."provider_payment_id" IS 'идемпотентность вебхука';

ALTER TABLE "access_tokens" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "sessions" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "sessions" ADD FOREIGN KEY ("family_id") REFERENCES "families" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "sessions" ADD FOREIGN KEY ("access_token_id") REFERENCES "access_tokens" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "families" ADD FOREIGN KEY ("parent_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "children" ADD FOREIGN KEY ("family_id") REFERENCES "families" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "groups" ADD FOREIGN KEY ("master_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "lessons" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "lessons" ADD FOREIGN KEY ("master_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "attendance_rules" ADD FOREIGN KEY ("group_id") REFERENCES "groups" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "attendance_rules" ADD FOREIGN KEY ("created_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "bookings" ADD FOREIGN KEY ("lesson_id") REFERENCES "lessons" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "bookings" ADD FOREIGN KEY ("child_id") REFERENCES "children" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "attendance" ADD FOREIGN KEY ("booking_id") REFERENCES "bookings" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "attendance" ADD FOREIGN KEY ("marked_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "subscriptions" ADD FOREIGN KEY ("child_id") REFERENCES "children" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "subscription_ledger" ADD FOREIGN KEY ("subscription_id") REFERENCES "subscriptions" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "subscription_ledger" ADD FOREIGN KEY ("attendance_id") REFERENCES "attendance" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "subscription_ledger" ADD FOREIGN KEY ("created_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "payments" ADD FOREIGN KEY ("family_id") REFERENCES "families" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "payments" ADD FOREIGN KEY ("child_id") REFERENCES "children" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "receipts" ADD FOREIGN KEY ("payment_id") REFERENCES "payments" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "receipts" ADD FOREIGN KEY ("uploaded_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "waitlist_entries" ADD FOREIGN KEY ("lesson_id") REFERENCES "lessons" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "waitlist_entries" ADD FOREIGN KEY ("child_id") REFERENCES "children" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "waitlist_entries" ADD FOREIGN KEY ("family_id") REFERENCES "families" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "waitlist_entries" ADD FOREIGN KEY ("created_by") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "notifications" ADD FOREIGN KEY ("user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "notifications" ADD FOREIGN KEY ("family_id") REFERENCES "families" ("id") DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "audit_log" ADD FOREIGN KEY ("actor_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
