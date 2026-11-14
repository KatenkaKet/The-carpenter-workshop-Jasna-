#!/usr/bin/env bash
set -euo pipefail
if [[ $# -ne 1 || ! -f $1 ]]; then
  printf 'Usage: %s /absolute/path/backup.dump\n' "$0" >&2
  exit 2
fi
backup_path=$(realpath -- "$1")
project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$project_root"
restore_db="jasna_restore_check_$(date -u +%Y%m%d%H%M%S)_$$"
docker compose exec -T db sh -c 'createdb -U "$POSTGRES_USER" "$1"' sh "$restore_db"
docker compose exec -T db sh -c 'pg_restore --exit-on-error --no-owner -U "$POSTGRES_USER" -d "$1"' sh "$restore_db" < "$backup_path"
docker compose exec -T db sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$1" -c "SELECT count(*) AS subscriptions FROM jasna.subscriptions; SELECT count(*) AS receipts FROM jasna.receipts; SELECT count(*) AS invalid_balances FROM jasna.subscriptions s WHERE lessons_remaining<>(SELECT coalesce(sum(delta),0) FROM jasna.subscription_ledger l WHERE l.subscription_id=s.id);"' sh "$restore_db"
printf 'Restored into isolated database %s. Production database was not changed.\n' "$restore_db"
