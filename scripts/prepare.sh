# Завершаем скрипт при любой ошибке
set -e
PG_USER=validator
PG_PASS=val1dat0r
PG_HOST=localhost
PG_PORT=5432
PG_DB=project-sem-1

# Установка зависимостей Go
echo "Установка зависимостей Go..."
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go mod tidy

# Проверяем, доступна ли команда migrate
if ! command -v migrate &> /dev/null; then
    echo "Ошибка: migrate не установлен или недоступен в PATH"
    exit 1
fi

# Проверяем наличие каталога migrations
if [ ! -d "migrations" ]; then
    echo "Ошибка: каталог migrations не найден!"
    exit 1
fi

# Запуск миграции
echo "Запуск миграции..." 
migrate -path=migrations -database "postgres://${PG_USER}:${PG_PASS}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=disable" up

# Компиляция приложения
echo "Компиляция Go-приложения..."
go build -o app cmd/priceanalyzer/main.go