.PHONY: help run build test clean install dev docker-build docker-run fmt lint

# Variables
BINARY_NAME=api-recaptcha
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=./cmd/server/main.go
GO=go
GOFLAGS=-v

# Colores para output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

## help: Muestra este mensaje de ayuda
help:
	@echo "$(GREEN)Comandos disponibles:$(NC)"
	@echo ""
	@grep -E '^## .*:' $(MAKEFILE_LIST) | sed 's/## /  $(YELLOW)/' | sed 's/:/ $(NC)-/'
	@echo ""

## run: Ejecuta la aplicación en modo desarrollo
run:
	@echo "$(GREEN)🚀 Iniciando servidor...$(NC)"
	@$(GO) run $(MAIN_PATH)

## dev: Ejecuta la aplicación con recarga automática (requiere air)
dev:
	@echo "$(GREEN)🔥 Iniciando servidor en modo desarrollo con hot-reload...$(NC)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(RED)❌ Error: 'air' no está instalado.$(NC)"; \
		echo "$(YELLOW)Instálalo con: go install github.com/cosmtrek/air@latest$(NC)"; \
		exit 1; \
	fi

## build: Compila el binario
build:
	@echo "$(GREEN)🔨 Compilando binario...$(NC)"
	@mkdir -p bin
	@$(GO) build $(GOFLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)✅ Binario creado en: $(BINARY_PATH)$(NC)"

## build-linux: Compila el binario para Linux
build-linux:
	@echo "$(GREEN)🔨 Compilando para Linux...$(NC)"
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_PATH)-linux $(MAIN_PATH)
	@echo "$(GREEN)✅ Binario Linux creado: $(BINARY_PATH)-linux$(NC)"

## build-windows: Compila el binario para Windows
build-windows:
	@echo "$(GREEN)🔨 Compilando para Windows...$(NC)"
	@mkdir -p bin
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_PATH).exe $(MAIN_PATH)
	@echo "$(GREEN)✅ Binario Windows creado: $(BINARY_PATH).exe$(NC)"

## build-mac: Compila el binario para macOS
build-mac:
	@echo "$(GREEN)🔨 Compilando para macOS...$(NC)"
	@mkdir -p bin
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BINARY_PATH)-mac $(MAIN_PATH)
	@echo "$(GREEN)✅ Binario macOS creado: $(BINARY_PATH)-mac$(NC)"

## build-all: Compila binarios para todas las plataformas
build-all: build-linux build-windows build-mac
	@echo "$(GREEN)✅ Todos los binarios compilados$(NC)"

## install: Instala las dependencias del proyecto
install:
	@echo "$(GREEN)📦 Instalando dependencias...$(NC)"
	@$(GO) mod download
	@$(GO) mod tidy
	@echo "$(GREEN)✅ Dependencias instaladas$(NC)"

## test: Ejecuta las pruebas
test:
	@echo "$(GREEN)🧪 Ejecutando pruebas...$(NC)"
	@$(GO) test -v ./...

## test-coverage: Ejecuta las pruebas con cobertura
test-coverage:
	@echo "$(GREEN)🧪 Ejecutando pruebas con cobertura...$(NC)"
	@$(GO) test -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Reporte de cobertura generado: coverage.html$(NC)"

## fmt: Formatea el código
fmt:
	@echo "$(GREEN)🎨 Formateando código...$(NC)"
	@$(GO) fmt ./...
	@echo "$(GREEN)✅ Código formateado$(NC)"

## lint: Ejecuta el linter (requiere golangci-lint)
lint:
	@echo "$(GREEN)🔍 Ejecutando linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(RED)❌ Error: 'golangci-lint' no está instalado.$(NC)"; \
		echo "$(YELLOW)Instálalo desde: https://golangci-lint.run/usage/install/$(NC)"; \
		exit 1; \
	fi

## clean: Limpia los archivos generados
clean:
	@echo "$(GREEN)🧹 Limpiando archivos generados...$(NC)"
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Archivos limpiados$(NC)"

## docker-build: Construye la imagen Docker
docker-build:
	@echo "$(GREEN)🐳 Construyendo imagen Docker...$(NC)"
	@docker build -t $(BINARY_NAME):latest .
	@echo "$(GREEN)✅ Imagen Docker creada: $(BINARY_NAME):latest$(NC)"

## docker-run: Ejecuta el contenedor Docker
docker-run:
	@echo "$(GREEN)🐳 Ejecutando contenedor Docker...$(NC)"
	@docker run --rm -p 8080:8080 --env-file .env $(BINARY_NAME):latest

## docker-stop: Detiene todos los contenedores del proyecto
docker-stop:
	@echo "$(GREEN)🐳 Deteniendo contenedores...$(NC)"
	@docker stop $$(docker ps -q --filter ancestor=$(BINARY_NAME):latest) 2>/dev/null || true

## setup-env: Crea el archivo .env desde .env.example
setup-env:
	@if [ ! -f .env ]; then \
		echo "$(GREEN)📝 Creando archivo .env desde .env.example...$(NC)"; \
		cp .env.example .env; \
		echo "$(YELLOW)⚠️  Recuerda editar .env con tus credenciales$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  El archivo .env ya existe$(NC)"; \
	fi

## mod-update: Actualiza las dependencias
mod-update:
	@echo "$(GREEN)📦 Actualizando dependencias...$(NC)"
	@$(GO) get -u ./...
	@$(GO) mod tidy
	@echo "$(GREEN)✅ Dependencias actualizadas$(NC)"

## serve: Alias para run
serve: run

## start: Compila y ejecuta el binario
start: build
	@echo "$(GREEN)🚀 Ejecutando binario...$(NC)"
	@./$(BINARY_PATH)

.DEFAULT_GOAL := help
