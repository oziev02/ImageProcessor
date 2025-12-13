# Архитектура проекта ImageProcessor

## Обзор

ImageProcessor - это сервис для асинхронной обработки изображений, построенный на основе BigTech Go Service Architecture. Сервис предоставляет HTTP API для загрузки изображений и использует Apache Kafka для фоновой обработки.

## Архитектурные принципы

Проект следует принципам чистой архитектуры с четким разделением слоев:

1. **Domain Layer** - чистые бизнес-сущности без зависимостей
2. **Service Layer** - бизнес-логика и use cases
3. **Repository Layer** - адаптеры для работы с данными
4. **Transport Layer** - адаптеры для внешних интерфейсов (HTTP, Kafka)
5. **App Layer** - композиция и инициализация компонентов

Направление зависимостей строго соблюдается: Transport → Service → Repository → Domain.

## Структура проекта

```
/cmd/imageprocessor/          - Точка входа приложения (main.go)
/bin/                         - Артефакты сборки (бинарные файлы)
/internal/
  /app/                       - Инициализация, композиция, жизненный цикл
  /config/                    - Конфигурация из переменных окружения
  /domain/                    - Доменные модели и бизнес-правила
  /migrations/                - Миграции БД (встраиваются в бинарник)
  /service/                   - Бизнес-логика и use cases
  /repo/                      - Репозитории для работы с данными
  /transport/
    /http/                    - HTTP handlers и роутинг
      /web/                   - Веб-интерфейс (HTML, CSS, JS)
    /kafka/                   - Kafka producer и consumer
  /observability/             - Логирование
/storage/                     - Файловое хранилище изображений (создается автоматически)
```

**Примечание:** Файл `cmd/imageprocessor/main.go` должен быть создан для точки входа приложения. Он должен инициализировать и запускать `internal/app.App`.

## Слои архитектуры

### 1. Domain Layer (`internal/domain/`)

Содержит чистые бизнес-сущности без внешних зависимостей.

**Основные сущности:**

- **Image** - основная сущность изображения:
  - ID, пути к файлам (original, processed, thumbnail)
  - Статус обработки (pending, processing, completed, failed)
  - Формат (JPEG, PNG, GIF)
  - Размеры изображений
  - Метки времени создания и обновления

- **ProcessingTask** - задача для фоновой обработки:
  - ImageID, путь к изображению
  - Формат и размеры оригинала

- **ProcessingStatus** - перечисление статусов обработки
- **ImageFormat** - поддерживаемые форматы изображений

**Доменные ошибки:**
- ErrInvalidImageID
- ErrInvalidImagePath
- ErrImageNotFound
- ErrInvalidFormat

Доменный слой не содержит бизнес-логику, только структуры данных и базовую валидацию.

### 2. Repository Layer (`internal/repo/`)

Адаптеры для работы с хранилищами данных. Не содержат бизнес-логику.

**ImageRepository** - работа с PostgreSQL:
- Create - создание записи об изображении
- GetByID - получение по ID
- Update - обновление записи (статус, пути к обработанным файлам)
- Delete - удаление записи
- List - получение списка с пагинацией

Использует pgx/v5 для работы с PostgreSQL. Все SQL-запросы параметризованы. Ошибки БД преобразуются в доменные ошибки.

**StorageRepository** - работа с файловой системой:
- Save - сохранение файла
- Read - чтение файла
- Delete - удаление файла
- Exists - проверка существования файла

Управляет структурой директорий: original/, processed/, thumbnail/.

### 3. Service Layer (`internal/service/`)

Содержит бизнес-логику и use cases. Зависит только от domain и интерфейсов репозиториев.

**ImageService** - основные операции с изображениями:
- Upload - загрузка изображения:
  * Валидация размера файла
  * Генерация UUID для идентификации
  * Определение формата
  * Сохранение оригинального файла
  * Получение размеров изображения
  * Создание записи в БД со статусом "pending"
  * Отправка задачи в Kafka для обработки

- GetByID - получение информации об изображении
- Delete - удаление изображения и связанных файлов
- List - получение списка изображений

**ProcessorService** - обработка изображений:
- ProcessImage - асинхронная обработка изображения:
  * Обновление статуса на "processing"
  * Загрузка оригинального изображения
  * Декодирование изображения
  * Создание обработанной версии (resize до указанных размеров)
  * Создание миниатюры
  * Сохранение обработанных файлов
  * Обновление записи в БД со статусом "completed"

Использует библиотеку nfnt/resize для изменения размера изображений.

### 4. Transport Layer (`internal/transport/`)

Адаптеры для внешних интерфейсов. Тонкий слой, делегирующий работу в сервисы.

#### HTTP (`internal/transport/http/`)

**Handler** - HTTP handlers:
- Upload - прием multipart/form-data с изображением
- GetImage - возврат обработанного изображения
- GetImageInfo - возврат метаданных об изображении
- ListImages - список изображений с пагинацией
- DeleteImage - удаление изображения
- Index - отдача веб-интерфейса

Handlers валидируют входные данные, вызывают соответствующие методы сервисов, преобразуют ошибки в HTTP статусы. Не содержат бизнес-логику.

**Server** - HTTP сервер:
- Настройка роутера chi
- Middleware: RequestID, RealIP, Logger, Recoverer, Timeout
- Запуск и graceful shutdown

#### Kafka (`internal/transport/kafka/`)

**Producer** - отправка задач обработки:
- SendTask - сериализация ProcessingTask в JSON и отправка в топик
- Использует kafka-go с балансировщиком LeastBytes

**Consumer** - получение и обработка задач:
- Start - запуск цикла чтения сообщений из топика
- Десериализация ProcessingTask из JSON
- Вызов ProcessorService для обработки
- Commit сообщения после успешной обработки
- Продолжение работы при ошибках обработки отдельных задач

### 5. App Layer (`internal/app/`)

Композиция всех компонентов и управление жизненным циклом.

**Инициализация (New):**
1. Загрузка конфигурации из переменных окружения
2. Инициализация логгера
3. Подключение к PostgreSQL и создание таблиц
4. Создание репозиториев (ImageRepository, StorageRepository)
5. Создание Kafka producer
6. Создание сервисов (ImageService, ProcessorService)
7. Создание Kafka consumer
8. Создание HTTP handler и server

**Запуск (Start):**
1. Запуск Kafka consumer в отдельной goroutine
2. Запуск HTTP сервера в отдельной goroutine
3. Ожидание сигнала завершения (SIGINT, SIGTERM)
4. Graceful shutdown:
   * Остановка Kafka consumer (отмена контекста)
   * Завершение HTTP сервера с таймаутом 30 секунд
   * Закрытие Kafka consumer
   * Закрытие соединения с БД

### 6. Config Layer (`internal/config/`)

Структурированная конфигурация с загрузкой из переменных окружения и значениями по умолчанию.

**Секции конфигурации:**
- Server - настройки HTTP сервера (host, port, таймауты)
- Database - настройки PostgreSQL
- Kafka - брокеры, топик, consumer group
- Storage - базовый путь для файлового хранилища
- Image - параметры обработки изображений (размеры, лимиты)

Валидация конфигурации выполняется при загрузке.

### 7. Observability (`internal/observability/`)

Логирование на основе стандартной библиотеки slog:
- JSON формат вывода
- Уровень логирования: Info

## Поток обработки изображения

### 1. Загрузка (Upload Flow)

```
HTTP Client
    ↓ POST /upload (multipart/form-data)
HTTP Handler
    ↓ Upload()
ImageService
    ↓ Валидация размера файла
    ↓ Сохранение оригинального файла (StorageRepository)
    ↓ Получение размеров изображения
    ↓ Создание записи в БД (ImageRepository) со статусом "pending"
    ↓ Создание ProcessingTask
    ↓ Отправка в Kafka (Producer)
    ↓ Возврат Image с ID и статусом "pending"
HTTP Handler
    ↓ JSON response
HTTP Client
```

### 2. Обработка (Processing Flow)

```
Kafka Topic (image-processing)
    ↓ Сообщение с ProcessingTask
Kafka Consumer
    ↓ Десериализация JSON → ProcessingTask
    ↓ ProcessImage(task)
ProcessorService
    ↓ Получение Image из БД (ImageRepository)
    ↓ Обновление статуса на "processing"
    ↓ Загрузка оригинального файла (StorageRepository)
    ↓ Декодирование изображения
    ↓ Resize для processed версии
    ↓ Resize для thumbnail
    ↓ Сохранение processed файла (StorageRepository)
    ↓ Сохранение thumbnail файла (StorageRepository)
    ↓ Обновление Image в БД (ImageRepository) со статусом "completed"
    ↓ Commit сообщения в Kafka
```

### 3. Получение изображения (Get Image Flow)

```
HTTP Client
    ↓ GET /image/{id}
HTTP Handler
    ↓ GetImage(id)
    ↓ GetByID(id)
ImageService
    ↓ GetByID(id)
ImageRepository
    ↓ SELECT из PostgreSQL
    ↓ Возврат Image
HTTP Handler
    ↓ Read() из StorageRepository
    ↓ Stream файла в HTTP response
HTTP Client
```

## Технологический стек

- **Язык:** Go 1.25+
- **База данных:** PostgreSQL 12+ (драйвер pgx/v5)
- **Очередь сообщений:** Apache Kafka 2.8+ (библиотека kafka-go)
- **HTTP роутинг:** chi
- **Обработка изображений:** стандартные библиотеки image/jpeg, image/png, image/gif; nfnt/resize для изменения размера
- **Логирование:** стандартная библиотека slog

## Особенности архитектуры

### Разделение ответственности

- Domain содержит только структуры данных и базовую валидацию
- Service содержит всю бизнес-логику
- Repository отвечает только за работу с хранилищами
- Transport отвечает только за преобразование протоколов

### Асинхронная обработка

Обработка изображений выполняется асинхронно через Kafka:
- Пользователь не ждет завершения обработки при загрузке
- Возможность масштабирования обработки (несколько consumer'ов)
- Устойчивость к ошибкам (сообщения можно повторить)

### Обработка ошибок

- Доменные ошибки определены в domain слое
- Ошибки БД преобразуются в доменные ошибки в repository
- Ошибки сервисов оборачиваются с контекстом (fmt.Errorf с %w)
- HTTP handlers преобразуют ошибки в соответствующие HTTP статусы

### Контекст

Все публичные методы принимают context.Context:
- Позволяет контролировать таймауты
- Поддерживает отмену операций
- Уважает cancellation в длительных операциях

### Тестируемость

Интерфейсы определены для всех зависимостей:
- ImageRepository, StorageRepository - можно легко мокировать
- ImageService, ProcessorService - можно тестировать с моками репозиториев
- Transport слои можно тестировать с моками сервисов

## База данных

### Миграции

Проект использует систему миграций на основе [golang-migrate/migrate](https://github.com/golang-migrate/migrate). Файлы миграций находятся в `internal/migrations/` и встраиваются в бинарный файл через `embed.FS` с использованием директивы `//go:embed *.sql`.

Миграции автоматически выполняются при запуске приложения в функции `runMigrations()` в `internal/app/app.go`.

**Структура миграций:**
- `000001_init.up.sql` - создание таблиц и индексов (использует `IF NOT EXISTS` для безопасности)
- `000001_init.down.sql` - откат миграции

Миграции используют `IF NOT EXISTS`, что позволяет безопасно выполнять их даже если таблицы уже существуют в базе данных.

### Схема таблицы images

```sql
CREATE TABLE images (
    id VARCHAR(255) PRIMARY KEY,
    original_path VARCHAR(500) NOT NULL,
    processed_path VARCHAR(500),
    thumbnail_path VARCHAR(500),
    status VARCHAR(50) NOT NULL,
    format VARCHAR(10) NOT NULL,
    original_width INTEGER NOT NULL,
    original_height INTEGER NOT NULL,
    processed_width INTEGER,
    processed_height INTEGER,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_images_status ON images(status);
CREATE INDEX idx_images_created_at ON images(created_at);
```

## Файловое хранилище

Структура директорий:
```
storage/
  original/     - исходные загруженные изображения
  processed/    - обработанные изображения (resize)
  thumbnail/    - миниатюры
```

Все файлы именуются по UUID изображения с расширением оригинального формата.

## Kafka

- **Топик:** image-processing (настраивается через KAFKA_TOPIC)
- **Consumer Group:** image-processor-group (настраивается через KAFKA_CONSUMER_GROUP)
- **Формат сообщения:** JSON с полями ProcessingTask
- **Key сообщения:** ImageID (для партиционирования)

## Масштабирование

Проект спроектирован для горизонтального масштабирования:

1. **HTTP сервер:** можно запускать несколько инстансов за load balancer
2. **Kafka Consumer:** несколько инстансов в одной consumer group обеспечат распределение нагрузки
3. **База данных:** можно использовать connection pooling (pgxpool)
4. **Файловое хранилище:** можно заменить на объектное хранилище (S3, MinIO)

## Безопасность

Текущая реализация ориентирована на разработку. Для production необходимо добавить:
- Аутентификацию и авторизацию
- Валидацию MIME-типов файлов
- Ограничение размеров запросов
- Rate limiting
- HTTPS/TLS
- CORS политики

## Расширяемость

Архитектура позволяет легко добавлять:
- Новые форматы изображений (через domain.ImageFormat)
- Новые типы обработки (через ProcessorService)
- Новые транспорты (gRPC, WebSocket)
- Новые хранилища (S3, MinIO через интерфейс StorageRepository)
- Метрики и трейсинг (OpenTelemetry)
- Дополнительные сервисы обработки

