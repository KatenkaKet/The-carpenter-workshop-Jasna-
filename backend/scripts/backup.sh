#!/usr/bin/env bash
set -euo pipefail
umask 077
project_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$project_root"
mkdir -p backups
backup_path="backups/jasna-$(date -u +%Y%m%dT%H%M%SZ)-$$.dump"
tmp_path="${backup_path}.partial"
trap 'rm -f -- "$tmp_path"' EXIT
docker compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" -Fc' > "$tmp_path"
mv -- "$tmp_path" "$backup_path"
printf '%s\n' "$project_root/$backup_path"
