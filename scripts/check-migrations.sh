#!/usr/bin/env bash
# 校验 scripts/db/migrations 中每个 *_down.sql 均有对应的正向迁移文件（与 CI 集成/发版前检查一致）。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIR="$ROOT/scripts/db/migrations"

if [[ ! -d "$DIR" ]]; then
  echo "missing $DIR" >&2
  exit 1
fi

shopt -s nullglob
for down in "$DIR"/*_down.sql; do
  base=$(basename "$down")
  forward="${base/_down.sql/.sql}"
  if [[ ! -f "$DIR/$forward" ]]; then
    echo "migration: rollback $base has no forward file $forward" >&2
    exit 1
  fi
done

echo "check-migrations: OK ($DIR)"
