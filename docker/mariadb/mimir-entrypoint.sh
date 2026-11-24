#!/bin/bash
set -euo pipefail

: "${MARIADB_DATABASE:?MARIADB_DATABASE is required}"
: "${MARIADB_USER:?MARIADB_USER is required}"
: "${MARIADB_ROOT_PASSWORD:?MARIADB_ROOT_PASSWORD is required}"
: "${LOADED_DB_NAME:?LOADED_DB_NAME is required}"
: "${LOADED_DB_USER:?LOADED_DB_USER is required}"
: "${LOADED_DB_PASSWORD:?LOADED_DB_PASSWORD is required}"

if [[ ! "$MARIADB_DATABASE" =~ ^[A-Za-z0-9_]+$ ||
      ! "$MARIADB_USER" =~ ^[A-Za-z0-9_]+$ ||
      ! "$LOADED_DB_NAME" =~ ^[A-Za-z0-9_]+$ ||
      ! "$LOADED_DB_USER" =~ ^[A-Za-z0-9_]+$ ]]; then
    echo "Database and account names may contain only letters, digits, and underscores" >&2
    exit 1
fi

if [[ "$MARIADB_DATABASE" == "$LOADED_DB_NAME" ||
      "$MARIADB_USER" == "$LOADED_DB_USER" ]]; then
    echo "Metadata and loaded-data databases and accounts must be distinct" >&2
    exit 1
fi

escape_sql_string() {
    local value=${1//\\/\\\\}
    value=${value//\'/\\\'}
    printf "'%s'" "$value"
}

docker-entrypoint.sh "$@" &
server_pid=$!
trap 'kill -TERM "$server_pid" 2>/dev/null || true' TERM INT

attempt=0
until MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb --user=root --execute="SELECT 1" >/dev/null 2>&1; do
    if ! kill -0 "$server_pid" 2>/dev/null; then
        wait "$server_pid"
        echo "MariaDB stopped before it became ready for loaded-data provisioning" >&2
        exit 1
    fi
    attempt=$((attempt + 1))
    if (( attempt >= 60 )); then
        echo "Timed out waiting for MariaDB to provision the loaded-data database" >&2
        exit 1
    fi
    sleep 1
done

loaded_db_password=$(escape_sql_string "$LOADED_DB_PASSWORD")
MYSQL_PWD="$MARIADB_ROOT_PASSWORD" mariadb --user=root <<SQL
CREATE DATABASE IF NOT EXISTS \`$LOADED_DB_NAME\`;
CREATE USER IF NOT EXISTS '$LOADED_DB_USER'@'%' IDENTIFIED BY $loaded_db_password;
ALTER USER '$LOADED_DB_USER'@'%' IDENTIFIED BY $loaded_db_password;
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, ALTER, INDEX, CREATE TEMPORARY TABLES
    ON \`$LOADED_DB_NAME\`.* TO '$LOADED_DB_USER'@'%';
SQL

wait "$server_pid"
