# 1. Стадия сборки (builder)
FROM golang:1.25-alpine AS builder

# Устанавливаем дополнительные инструменты, если нужны (git для загрузки приватных зависимостей)
RUN apk add --no-cache git

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем файлы зависимостей
COPY go.mod go.sum ./

# Скачиваем зависимости (используем кэширование слоев Docker)
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o service-a ./main.go

# 2. Финальная стадия (минимальный образ)
FROM alpine:latest

# Устанавливаем рабочую директорию
WORKDIR /root/

# Копируем скомпилированный бинарник из стадии сборки
COPY --from=builder /app/service-a .

# При необходимости устанавливаем зависимости времени выполнения (например, для TLS)
RUN apk add --no-cache ca-certificates

# Опционально: создаем непривилегированного пользователя для запуска
RUN adduser -D -g '' appuser
USER appuser

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./service-a"]