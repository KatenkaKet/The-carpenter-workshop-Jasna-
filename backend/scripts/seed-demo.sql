-- Fictional, additive demo data. Safe to run again: marker prevents duplication.
BEGIN;
SET LOCAL search_path TO jasna, public;
SELECT pg_advisory_xact_lock(7341022);
DO $$
DECLARE
 m uuid; u uuid; f uuid; c uuid; g uuid; l uuid; future_l uuid; rule uuid; plan uuid;
 b uuid; a uuid; inv uuid; pay uuid; sub uuid; entry uuid; receipt uuid; link uuid;
 i integer; slot timestamptz; snapshot jsonb; response jsonb; request_body text;
 png bytea := decode('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=','base64');
BEGIN
 IF EXISTS(SELECT 1 FROM audit_log WHERE request_id='jasna-demo-seed-v1') THEN
  RAISE NOTICE 'Demo data already exists; no changes made'; RETURN;
 END IF;
 SELECT id INTO m FROM users WHERE role='master' ORDER BY created_at LIMIT 1;
 IF m IS NULL THEN RAISE EXCEPTION 'Start backend and create master first'; END IF;
 SELECT greatest(now()+interval '2 days',coalesce(max(ends_at),now())+interval '1 day') INTO slot FROM lessons WHERE status='scheduled';
 FOR i IN 1..3 LOOP
  INSERT INTO users(role,full_name,email,phone) VALUES('parent','ДЕМО — Родитель '||i,'demo-parent-'||i||'@example.invalid',NULL) RETURNING id INTO u;
  INSERT INTO families(parent_user_id) VALUES(u) RETURNING id INTO f;
  INSERT INTO children(family_id,full_name,birth_date) VALUES(f,'ДЕМО — Ребёнок '||i,(current_date-interval '9 years')::date) RETURNING id INTO c;
  INSERT INTO children(family_id,full_name,birth_date) VALUES(f,'ДЕМО — Младший ребёнок '||i,(current_date-interval '7 years')::date);
  INSERT INTO groups(master_id,title,capacity,description) VALUES(m,'ДЕМО — Столярная группа '||i,4,'Вымышленные данные для Postman') RETURNING id INTO g;
  INSERT INTO attendance_rules(group_id,valid_from,attended_charge_lesson,missed_charge_lesson,parent_cancel_charge_lesson,free_cancel_before_hours,parent_cancel_allowed,cancel_deadline_hours,transfer_allowed,transfer_deadline_hours,transfer_charge_lesson,single_lesson_price_minor,ending_notice_days,ending_notice_lessons)
  VALUES(g,current_date-30,true,false,false,24,true,0,true,0,false,100000,3,1) RETURNING id INTO rule;
  SELECT to_jsonb(r) INTO snapshot FROM attendance_rules r WHERE id=rule;
  INSERT INTO plans(group_id,title,lessons_total,validity_days,price_minor) VALUES(g,'ДЕМО — 4 занятия',4,30,400000) RETURNING id INTO plan;
  INSERT INTO lessons(group_id,master_id,starts_at,ends_at,capacity,status,notes)
  VALUES(g,m,now()-make_interval(days=>i+1),now()-make_interval(days=>i+1)+interval '1 hour',4,'completed','ДЕМО — проведённое занятие') RETURNING id INTO l;
  INSERT INTO bookings(lesson_id,child_id,family_id,rule_snapshot,rule_id) VALUES(l,c,f,snapshot,rule) RETURNING id INTO b;
  INSERT INTO invoices(family_id,child_id,group_id,kind,plan_snapshot,amount_minor,status,valid_from,valid_until)
  SELECT f,c,g,'subscription',to_jsonb(p),400000,'paid',current_date-10,current_date+19 FROM plans p WHERE id=plan RETURNING id INTO inv;
  INSERT INTO payments(invoice_id,family_id,child_id,amount_minor,method,status,paid_at,comment)
  VALUES(inv,f,c,400000,CASE WHEN i=2 THEN 'transfer' ELSE 'cash' END,'succeeded',now()-interval '10 days','ДЕМО: вымышленная ручная оплата, деньги не переводились') RETURNING id INTO pay;
  INSERT INTO money_ledger(family_id,invoice_id,amount_minor,reason,source_id) VALUES(f,inv,-400000,'invoice',inv);
  INSERT INTO money_ledger(family_id,invoice_id,payment_id,amount_minor,reason,source_id) VALUES(f,inv,pay,400000,'payment',pay);
  INSERT INTO subscriptions(invoice_id,child_id,family_id,group_id,lessons_total,valid_from,valid_until)
  VALUES(inv,c,f,g,4,current_date-10,current_date+19) RETURNING id INTO sub;
  INSERT INTO subscription_ledger(subscription_id,delta,reason,payment_id,created_by,comment) VALUES(sub,4,'subscription_purchase',pay,m,'ДЕМО — покупка');
  INSERT INTO attendance(booking_id,status,charged,subscription_id,marked_at,marked_by,comment)
  VALUES(b,'attended',true,sub,now()-make_interval(days=>i+1)+interval '1 hour',m,'ДЕМО — посещение') RETURNING id INTO a;
  INSERT INTO subscription_ledger(subscription_id,delta,reason,attendance_id,created_by,comment) VALUES(sub,-1,'attendance_charge',a,m,'ДЕМО — списание') RETURNING id INTO entry;
  UPDATE attendance SET charge_entry_id=entry WHERE id=a;
  INSERT INTO lessons(group_id,master_id,starts_at,ends_at,capacity,notes)
  VALUES(g,m,slot+make_interval(days=>i),slot+make_interval(days=>i)+interval '1 hour',4,'ДЕМО — можно проверить запись') RETURNING id INTO future_l;
  INSERT INTO bookings(lesson_id,child_id,family_id,rule_snapshot,rule_id) VALUES(future_l,c,f,snapshot,rule) RETURNING id INTO b;
  INSERT INTO attendance(booking_id) VALUES(b);
  -- A separate open invoice demonstrates monetary debt independently of remaining lessons.
  INSERT INTO invoices(family_id,child_id,group_id,kind,plan_snapshot,amount_minor,valid_from,valid_until)
  SELECT f,c,g,'subscription',to_jsonb(p),400000,current_date+20,current_date+49 FROM plans p WHERE id=plan RETURNING id INTO inv;
  INSERT INTO money_ledger(family_id,invoice_id,amount_minor,reason,source_id) VALUES(f,inv,-400000,'invoice',inv);
  INSERT INTO receipts(payment_id,kind,mime_type,file_name,content,uploaded_by,version)
  VALUES(pay,'file','image/png','DEMO-not-a-real-receipt.png',png,m,2) RETURNING id INTO receipt;
  INSERT INTO receipt_versions(receipt_id,snapshot,content,comment,actor_id)
  SELECT receipt,(to_jsonb(r)-'content')||jsonb_build_object('version',1,'file_name','DEMO-first-version.png'),png,'ДЕМО — сохранённая предыдущая версия',m FROM receipts r WHERE id=receipt;
  -- No usable credentials are seeded. Create a fresh family link through master API.
  INSERT INTO access_links(family_id,token_hash,expires_at,revoked_at)
  VALUES(f,encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),now()-interval '1 day',now()) RETURNING id INTO link;
  INSERT INTO sessions(user_id,family_id,access_link_id,role,token_hash,csrf_token,expires_at,revoked_at)
  VALUES(u,f,link,'parent',encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),gen_random_uuid()::text,now()-interval '1 day',now());
  INSERT INTO outbox(user_id,family_id,type,payload,event_key,delivered_at)
  VALUES(u,f,'payment.required',jsonb_build_object('invoice_id',inv),'demo-seed-v1-'||i,now());
  INSERT INTO notifications(user_id,family_id,type,payload,event_key,read_at)
  VALUES(u,f,'payment.required',jsonb_build_object('invoice_id',inv),'demo-seed-v1-'||i,CASE WHEN i=1 THEN now() ELSE NULL END);
  SELECT to_jsonb(x) INTO response FROM bookings x WHERE id=b;
  request_body:=jsonb_build_object('lesson_id',future_l,'child_id',c)::text;
  INSERT INTO idempotency(user_id,route,key,body_hash,status_code,response)
  VALUES(u,'/api/v1/bookings','demo-seed-v1-booking-'||i,encode(sha256(convert_to(request_body,'UTF8')),'hex'),201,response);
  INSERT INTO audit_log(event_type,actor_user_id,actor_role,resource_type,resource_id,after_state,reason,request_id)
  VALUES('demo.seeded',m,'master','families',f,jsonb_build_object('family_id',f,'child_id',c,'subscription_id',sub,'receipt_id',receipt),'Добавлены вымышленные данные; платежей провайдеру нет','jasna-demo-seed-v1');
 END LOOP;
END $$;
COMMIT;
