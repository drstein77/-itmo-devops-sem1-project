# Завершаем скрипт при любой ошибке
set -e

# Установка зависимостей Go
echo "Установка зависимостей Go..."
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go mod tidy

# Проверяем, доступна ли команда migrate
if ! command -v migrate &> /dev/null; then
    echo "Ошибка: migrate не установлен или недоступен в PATH"
    exit 1
fi

# Запуск миграции
echo "Запуск миграции..."
migrate -path=migrations -database "postgres://validator:val1dat0r@localhost:5432/project-sem-1?sslmode=disable" up

# Компиляция приложения
echo "Компиляция Go-приложения..."
go build -o app cmd/priceanalyzer/main.go