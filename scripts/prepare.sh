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

# Проверка соединения с базой данных
echo "Проверяем доступность базы данных"
if ! psql -U "$USERNAME" -h "$HOST" -p "$PORT" -d "$DATABASE" -c "\q" &> /dev/null; then
  echo "Не удается подключиться к базе $DATABASE. Начинаем диагностику."

  # Пробуем подключиться как суперпользователь postgres
  echo "Проверяем соединение с пользователем postgres"
  SUPERUSER="postgres"
  if ! psql -U "$SUPERUSER" -h "$HOST" -p "$PORT" -c "\q" &> /dev/null; then
    echo "Ошибка: Не удалось подключиться с правами суперпользователя."
    exit 1
  fi

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
EOF
else
  echo "База данных $DATABASE доступна."
fi

echo "Настройка базы данных завершена."
