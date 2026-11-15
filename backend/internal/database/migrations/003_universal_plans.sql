SET search_path TO jasna, public;
ALTER TABLE plans ALTER COLUMN group_id DROP NOT NULL;
ALTER TABLE invoices ALTER COLUMN group_id DROP NOT NULL;
ALTER TABLE subscriptions ALTER COLUMN group_id DROP NOT NULL;
-- Future purchases are universal; paid subscription scope and invoice snapshots stay intact.
UPDATE plans SET group_id=NULL, version=version+1 WHERE group_id IS NOT NULL;
ALTER TABLE bookings DROP CONSTRAINT bookings_funding_kind_check;
ALTER TABLE bookings ADD CHECK (funding_kind IN ('legacy','subscription','single_lesson','awaiting_payment'));
ALTER TABLE payments DROP CONSTRAINT payments_method_check;
ALTER TABLE payments ADD CHECK (method IN ('cash','transfer','online','sbp','test'));
ALTER TABLE group_policies ALTER COLUMN group_id DROP NOT NULL;
CREATE UNIQUE INDEX group_policy_global ON group_policies ((true)) WHERE group_id IS NULL;
