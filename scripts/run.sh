#!/bin/bash

set -e

PG_USER=validator
PG_PASS=val1dat0r
PG_HOST=localhost
PG_PORT=5432
PG_DB=project-sem-1

echo "=== Запуск приложения ==="

# Загрузка переменных окружения 
export RUN_ADDRESS=:8080
export LOG_LEVEL=debug 
export DATABASE_URI=postgres://${PG_USER}:${PG_PASS}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=disable

echo "Переменные окружения загружены."

# Запуск приложения в фоновом режиме
echo "Запуск приложения..."
./app &

# Сохранение PID приложения, чтобы можно было его завершить позже
APP_PID=$!

# Ожидание, пока приложение не станет доступным
echo "Ожидание запуска приложения на порту 8080..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "Приложение запущено и доступно."
        break
    else
        echo "Ожидание..."
        sleep 1
    fi

    if [ $i -eq 30 ]; then
        echo "Ошибка: Приложение не запустилось в течение 30 секунд."
        kill $APP_PID
        exit 1
    fi

done

# Сохранение PID в файл
echo $APP_PID > app.pid