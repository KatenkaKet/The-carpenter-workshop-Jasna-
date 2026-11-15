SET search_path TO jasna, public;
ALTER TABLE lessons ADD COLUMN is_makeup boolean NOT NULL DEFAULT false;
ALTER TABLE payments DROP CONSTRAINT payments_status_check;
ALTER TABLE payments ADD CONSTRAINT payments_status_check CHECK(status IN ('pending','succeeded','canceled','voided'));
ALTER TABLE payments ADD COLUMN voided_at timestamptz, ADD COLUMN void_reason text;
ALTER TABLE subscriptions ADD COLUMN suspended_at timestamptz;
ALTER TABLE receipts ADD COLUMN deleted_at timestamptz;
ALTER TABLE attendance ADD COLUMN applied_rule jsonb;
ALTER TABLE bookings ADD COLUMN funding_kind text NOT NULL DEFAULT 'legacy' CHECK(funding_kind IN ('legacy','subscription','single_lesson'));
CREATE TABLE subscription_reservations (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), created_at timestamptz NOT NULL DEFAULT now(), version integer NOT NULL DEFAULT 1,
 booking_id uuid NOT NULL UNIQUE REFERENCES bookings(id), subscription_id uuid NOT NULL REFERENCES subscriptions(id),
 status text NOT NULL DEFAULT 'reserved' CHECK(status IN ('reserved','consumed','released'))
);
CREATE INDEX reservations_subscription ON subscription_reservations(subscription_id,status);
CREATE TABLE group_policies (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), created_at timestamptz NOT NULL DEFAULT now(), version integer NOT NULL DEFAULT 1,
 group_id uuid NOT NULL UNIQUE REFERENCES groups(id), missed_charge_lesson boolean NOT NULL,
 free_cancel_before_hours integer NOT NULL CHECK(free_cancel_before_hours BETWEEN 0 AND 8760), transfer_allowed boolean NOT NULL
);
-- Reserve only existing funded future bookings; never remove bookings or invent credit.
DO $$ DECLARE booking_row record; sid uuid; BEGIN
 FOR booking_row IN SELECT b.id,b.child_id,l.group_id,(l.starts_at AT TIME ZONE 'Asia/Yekaterinburg')::date AS lesson_day
 FROM bookings b JOIN lessons l ON l.id=b.lesson_id JOIN attendance a ON a.booking_id=b.id
 WHERE b.status='active' AND a.status='pending' AND l.status='scheduled' AND l.starts_at>now()
 AND NOT EXISTS(SELECT 1 FROM invoices i WHERE i.booking_id=b.id)
 ORDER BY l.starts_at,b.created_at,b.id LOOP
  SELECT s.id INTO sid FROM subscriptions s WHERE s.child_id=booking_row.child_id AND s.group_id=booking_row.group_id
  AND s.valid_from<=booking_row.lesson_day AND s.valid_until>=booking_row.lesson_day
  AND s.lessons_remaining>(SELECT count(*) FROM subscription_reservations r WHERE r.subscription_id=s.id AND r.status='reserved')
  ORDER BY s.valid_until,s.id LIMIT 1;
  IF sid IS NOT NULL THEN
   INSERT INTO subscription_reservations(booking_id,subscription_id) VALUES(booking_row.id,sid);
   UPDATE bookings SET funding_kind='subscription' WHERE id=booking_row.id;
  END IF;
 END LOOP;
END $$;
