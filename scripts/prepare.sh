#!/bin/bash

# Завершаем скрипт при любой ошибке
set -e

echo "Инициализация базы данных"

# Конфигурация подключения
HOST="localhost"
PORT=5432
USERNAME="validator"
PASSWORD="val1dat0r"
DATABASE="project-sem-1"

export PASSWORD

# Создание нового пользователя и базы данных
echo "Создаем нового пользователя и базу..."
psql -U "$SUPERUSER" -h "$HOST" -p "$PORT" <<EOF
DO \$\$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'validator') THEN
    CREATE ROLE validator WITH LOGIN PASSWORD 'val1dat0r';
    END IF;
END \$\$;

DO \$\$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '$DATABASE') THEN
    CREATE DATABASE $DATABASE WITH OWNER validator;
    END IF;
END \$\$;

ALTER DATABASE $DATABASE OWNER TO validator;
 