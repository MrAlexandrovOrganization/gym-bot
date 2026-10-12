# check=skip=InvalidDefaultArgInFrom

# Версии обязательны и передаются через Make/Compose.
# Этап 1: Сборка приложения
ARG GO_VERSION
ARG ALPINE_VERSION
FROM golang:${GO_VERSION}-alpine AS builder

# Устанавливаем необходимые пакеты для сборки
RUN apk add --no-cache git ca-certificates tzdata

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY logx ./logx
COPY *.go ./

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o gym-bot .

# Этап 2: Минимальный образ для запуска
FROM alpine:${ALPINE_VERSION}

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates tzdata

# Создаём непривилегированного пользователя
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем скомпилированное приложение из builder
COPY --from=builder /app/gym-bot .

# Меняем владельца файлов
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Запускаем приложение
CMD ["./gym-bot"]
