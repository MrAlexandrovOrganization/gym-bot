.PHONY: help up down restart logs logs-bot logs-db build rebuild ps db clean stop env

DOCKER_COMPOSE := docker compose

# Цвета для вывода
GREEN  := \033[0;32m
YELLOW := \033[0;33m
NC     := \033[0m # No Color

help: ## Показать эту справку
	@echo "$(GREEN)Доступные команды:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'

env: ## Создать .env файл из .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN)✓$(NC) Создан файл .env из .env.example"; \
		echo "$(YELLOW)⚠$(NC) Не забудьте заполнить токен бота и ID чата в .env"; \
	else \
		echo "$(YELLOW)⚠$(NC) Файл .env уже существует"; \
	fi

up: ## Запустить все контейнеры
	@echo "$(GREEN)Запуск контейнеров...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✓$(NC) Контейнеры запущены"
	@echo "$(YELLOW)Используйте 'make logs' для просмотра логов$(NC)"

down: ## Остановить все контейнеры
	@echo "$(YELLOW)Остановка контейнеров...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)✓$(NC) Контейнеры остановлены"

stop: ## Остановить все контейнеры (алиас для down)
	@$(MAKE) down

restart: ## Перезапустить все контейнеры
	@echo "$(YELLOW)Перезапуск контейнеров...$(NC)"
	$(DOCKER_COMPOSE) restart
	@echo "$(GREEN)✓$(NC) Контейнеры перезапущены"

logs: ## Показать логи всех контейнеров (Ctrl+C для выхода)
	$(DOCKER_COMPOSE) logs -f

logs-bot: ## Показать только логи бота (Ctrl+C для выхода)
	$(DOCKER_COMPOSE) logs -f bot

logs-db: ## Показать только логи базы данных (Ctrl+C для выхода)
	$(DOCKER_COMPOSE) logs -f postgres

ps: ## Показать статус контейнеров
	@$(DOCKER_COMPOSE) ps

build: ## Собрать Docker образы
	@echo "$(GREEN)Сборка Docker образов...$(NC)"
	$(DOCKER_COMPOSE) build
	@echo "$(GREEN)✓$(NC) Образы собраны"

rebuild: ## Пересобрать образы и перезапустить контейнеры
	@echo "$(GREEN)Пересборка и перезапуск...$(NC)"
	$(DOCKER_COMPOSE) up -d --build
	@echo "$(GREEN)✓$(NC) Готово"

db: ## Подключиться к PostgreSQL через psql
	@echo "$(GREEN)Подключение к базе данных...$(NC)"
	@$(DOCKER_COMPOSE) exec postgres psql -U $${POSTGRES_USER:-gymbot} -d $${POSTGRES_DB:-gymbot}

db-reset: ## Пересоздать базу данных (удалит все данные!)
	@echo "$(YELLOW)⚠ ВНИМАНИЕ! Это удалит все данные из базы!$(NC)"
	@read -p "Вы уверены? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)Удаление базы данных...$(NC)"; \
		$(DOCKER_COMPOSE) down -v; \
		echo "$(GREEN)Создание новой базы данных...$(NC)"; \
		$(DOCKER_COMPOSE) up -d; \
		echo "$(GREEN)✓$(NC) База данных пересоздана"; \
	else \
		echo "$(GREEN)Отменено$(NC)"; \
	fi

clean: ## Остановить контейнеры и удалить volumes (удалит все данные!)
	@echo "$(YELLOW)⚠ ВНИМАНИЕ! Это удалит все данные, включая базу данных!$(NC)"
	@read -p "Вы уверены? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(YELLOW)Очистка...$(NC)"; \
		$(DOCKER_COMPOSE) down -v; \
		echo "$(GREEN)✓$(NC) Всё очищено"; \
	else \
		echo "$(GREEN)Отменено$(NC)"; \
	fi

config: ## Показать конфигурацию docker-compose
	$(DOCKER_COMPOSE) config

dev: ## Запустить в режиме разработки (без -d)
	$(DOCKER_COMPOSE) up

shell: ## Открыть shell в контейнере бота
	$(DOCKER_COMPOSE) exec bot sh

test: ## Проверить конфигурацию и подключение
	@echo "$(GREEN)Проверка конфигурации...$(NC)"
	@if [ ! -f .env ]; then \
		echo "$(YELLOW)⚠$(NC) Файл .env не найден. Создайте его командой: make env"; \
		exit 1; \
	fi
	@echo "$(GREEN)✓$(NC) Файл .env существует"
	@$(DOCKER_COMPOSE) config > /dev/null && echo "$(GREEN)✓$(NC) docker-compose.yml валиден"
	@echo "$(GREEN)✓$(NC) Конфигурация в порядке"
