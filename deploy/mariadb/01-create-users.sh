#!/bin/sh
set -eu

for value in "$MARIADB_ROOT_PASSWORD" "$MIMIR_DB_RUNTIME_PASSWORD" "$MIMIR_DB_MIGRATION_PASSWORD"; do
  case "$value" in
    ""|*[!0-9a-fA-F]*)
      echo "Database credentials must be non-empty hexadecimal secrets" >&2
      exit 1
      ;;
  esac
  if [ "${#value}" -lt 32 ]; then
    echo "Database credentials must contain at least 32 hexadecimal characters" >&2
    exit 1
  fi
done

export MYSQL_PWD="$MARIADB_ROOT_PASSWORD"
mariadb --protocol=socket -uroot <<SQL
CREATE USER IF NOT EXISTS 'mimir_runtime'@'%' IDENTIFIED BY '${MIMIR_DB_RUNTIME_PASSWORD}';
CREATE USER IF NOT EXISTS 'mimir_migrator'@'%' IDENTIFIED BY '${MIMIR_DB_MIGRATION_PASSWORD}';
GRANT SELECT, INSERT, UPDATE ON metadata.* TO 'mimir_runtime'@'%';
GRANT ALL PRIVILEGES ON metadata.* TO 'mimir_migrator'@'%';
SQL
unset MYSQL_PWD
