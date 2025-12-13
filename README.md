# Image Processor

Сервис фоновой обработки изображений с использованием Apache Kafka для асинхронной обработки.

## Возможности

- Загрузка изображений через HTTP API
- Фоновая обработка через Apache Kafka
- Ресайз изображений
- Генерация миниатюр
- Поддержка форматов: JPEG, PNG, GIF
- Веб-интерфейс для управления изображениями
- Хранение исходных и обработанных изображений
- Отслеживание статуса обработки

## Архитектура

Сервис построен по архитектуре BigTech Go Service:

```
/cmd/imageprocessor/         - точка входа (main.go)
/internal/app/               - инициализация и жизненный цикл
/internal/config/            - конфигурация
/internal/domain/            - доменные модели
/internal/service/           - бизнес-логика
/internal/repo/              - репозитории (PostgreSQL, файловое хранилище)
/internal/transport/http/    - HTTP handlers и веб-интерфейс
/internal/transport/kafka/   - Kafka producer/consumer
/internal/observability/     - логирование
/internal/migrations/        - миграции БД (встраиваются в бинарник)
```

Подробное описание архитектуры, потоков обработки и технологического стека см. в [ARCHITECTURE.md](ARCHITECTURE.md).

## Требования

- Go 1.25+
- PostgreSQL 12+
- Apache Kafka 2.8+

## Установка и запуск

### 1. Клонирование и установка зависимостей

```bash
go mod download
```

### 2. Запуск зависимостей (PostgreSQL и Kafka)

Для удобства можно использовать docker-compose:

```bash
docker-compose up -d
```

Это запустит:
- PostgreSQL на порту 5433
- Zookeeper на порту 2181
- Kafka на порту 9092

Альтернативно, можно установить зависимости вручную:

#### Настройка базы данных

Создайте базу данных PostgreSQL:

```sql
CREATE DATABASE imageprocessor;
```

#### Настройка Kafka

Убедитесь, что Kafka запущен и доступен. Создайте топик (опционально, создастся автоматически):

```bash
kafka-topics.sh --create --topic image-processing --bootstrap-server localhost:9092
```

### 3. Переменные окружения

Создайте файл `.env` или установите переменные окружения:

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
# Примечание: для docker-compose используйте порт 5433
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=imageprocessor
DB_SSLMODE=disable

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=image-processing
KAFKA_CONSUMER_GROUP=image-processor-group

# Storage
STORAGE_BASE_PATH=./storage

# Image Processing
IMAGE_MAX_FILE_SIZE=10485760  # 10MB
IMAGE_THUMBNAIL_WIDTH=200
IMAGE_THUMBNAIL_HEIGHT=200
IMAGE_PROCESSED_WIDTH=800
IMAGE_PROCESSED_HEIGHT=800
IMAGE_WATERMARK_ENABLED=false
IMAGE_WATERMARK_PATH=
```

### 4. Запуск сервиса

**Быстрый запуск одной командой (рекомендуется):**

```bash
make start
```

Эта команда автоматически:
- Запустит Docker контейнеры (PostgreSQL, Kafka)
- Дождется готовности сервисов
- Запустит приложение с автоматическим выполнением миграций

**Альтернативные способы запуска:**

```bash
# Запуск через go run (требуется наличие cmd/imageprocessor/main.go)
go run cmd/imageprocessor/main.go

# Или через собранный бинарник
make build
./bin/imageprocessor
```

**Примечание:** Для работы команд `make run`, `make build` и `go run` необходимо создать файл `cmd/imageprocessor/main.go` с точкой входа приложения.

Сервис будет доступен по адресу: http://localhost:8080

## API Endpoints

### POST /upload
Загружает изображение для обработки.

**Request:**
- Content-Type: `multipart/form-data`
- Field: `image` (файл изображения)

**Response:**
```json
{
  "id": "uuid",
  "original_path": "original/uuid.jpg",
  "status": "pending",
  "format": "jpeg",
  "original_width": 1920,
  "original_height": 1080,
  "created_at": "2024-01-01T00:00:00Z"
}
```

### GET /image/{id}
Возвращает обработанное изображение.

### GET /api/image/{id}
Возвращает информацию об изображении.

**Response:**
```json
{
  "id": "uuid",
  "original_path": "original/uuid.jpg",
  "processed_path": "processed/uuid.jpg",
  "thumbnail_path": "thumbnail/uuid.jpg",
  "status": "completed",
  "format": "jpeg",
  "original_width": 1920,
  "original_height": 1080,
  "processed_width": 800,
  "processed_height": 800,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:01Z"
}
```

### GET /api/images
Возвращает список изображений.

**Query Parameters:**
- `limit` (default: 50) - количество изображений
- `offset` (default: 0) - смещение

### DELETE /image/{id}
Удаляет изображение и все связанные файлы.

## Веб-интерфейс

Веб-интерфейс доступен по адресу http://localhost:8080/

Возможности:
- Загрузка изображений через форму
- Просмотр статуса обработки в реальном времени
- Отображение обработанных изображений
- Удаление изображений

## Обработка изображений

Сервис автоматически обрабатывает загруженные изображения:

1. **Ресайз** - уменьшение до указанных размеров (по умолчанию 800x800)
2. **Миниатюра** - создание миниатюры (по умолчанию 200x200)
3. **Водяной знак** - опционально (требует настройки)

Обработка происходит асинхронно через Kafka, что позволяет:
- Не блокировать пользователя при загрузке
- Масштабировать обработку
- Обрабатывать изображения параллельно

## Статусы обработки

- `pending` - ожидание обработки
- `processing` - обработка в процессе
- `completed` - обработка завершена
- `failed` - ошибка обработки

## Структура хранилища

```
storage/
  original/     - исходные изображения
  processed/    - обработанные изображения
  thumbnail/    - миниатюры
```

## Миграции базы данных

Сервис использует систему миграций для управления схемой базы данных. Миграции автоматически выполняются при запуске приложения.

Файлы миграций находятся в `internal/migrations/` и встраиваются в бинарный файл через `embed.FS`.

### Структура миграций

- `000001_init.up.sql` - создание таблиц и индексов (использует `IF NOT EXISTS` для безопасности)
- `000001_init.down.sql` - откат миграции

Миграции используют `IF NOT EXISTS`, что позволяет безопасно выполнять их даже если таблицы уже существуют.

### Создание новой миграции

Для создания новой миграции создайте два файла в `internal/migrations/`:
- `000002_description.up.sql` - применение изменений
- `000002_description.down.sql` - откат изменений

Миграции выполняются автоматически при старте приложения. Для ручного управления можно использовать CLI инструмент `migrate`, указав путь к `internal/migrations/`.

## Разработка

### Быстрый старт

Для полного запуска приложения одной командой:

```bash
make start
```

Эта команда запустит Docker контейнеры и приложение автоматически.

### Запуск тестов

```bash
go test ./...
```

### Сборка

```bash
# Сборка бинарника
make build

# Или напрямую (требуется наличие cmd/imageprocessor/main.go)
go build -o bin/imageprocessor cmd/imageprocessor/main.go
```

Бинарный файл создается **только** в `bin/imageprocessor` и включает в себя все миграции и веб-интерфейс. При сборке через `make build` любой бинарник в корне проекта автоматически удаляется для поддержания чистоты структуры проекта.

## Лицензия

MIT

