#!/bin/bash

set -e

echo "=== Запуск приложения ==="

# Загрузка переменных окружения вручную
export RUN_ADDRESS=:8080
export LOG_LEVEL=debug
export DATABASE_URI=postgres://validator:val1dat0r@localhost:5432/project-sem-1?sslmode=disable
echo "Переменные окружения загружены."

# Компиляция приложения
echo "Компиляция Go-приложения..."
go build -o app ../cmd/priceanalyzer/

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

# Сохранение PID в файл для использования в других шагах, если необходимо
echo $APP_PID > app.pid

# Ожидание завершения скрипта (приложение продолжит работать в фоне)
# Это необходимо, чтобы GitHub Actions не завершил шаг, пока приложение работает
wait $APP_PID