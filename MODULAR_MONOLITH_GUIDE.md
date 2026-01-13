# Руководство по Modular Monolith архитектуре

## Структура проекта

```
internal/
├── modules/                    # Бизнес-модули
│   ├── channel/               # Модуль каналов
│   │   ├── domain/            # Доменные сущности и правила
│   │   ├── service/           # Бизнес-логика
│   │   ├── repository/        # Интерфейсы и реализации хранилища
│   │   └── handler/           # Обработчики (опционально)
│   ├── feed/                  # Модуль RSS фидов
│   ├── message/               # Модуль сообщений
│   └── user/                  # Модуль пользователей
├── shared/                    # Общий код
│   ├── config/                # Конфигурация
│   ├── errors/                # Общие ошибки
│   └── logger/                # Логирование
└── di/                        # Dependency Injection
```

## Принципы модульности

### 1. Независимость модулей
- Каждый модуль имеет свой domain, service, repository
- Модули взаимодействуют через интерфейсы
- Избегайте прямых зависимостей между модулями

### 2. Общий код в shared/
- Конфигурация
- Общие ошибки
- Утилиты

### 3. Repository Pattern
- Интерфейсы в `repository/repository.go`
- Реализации в `repository/file_storage.go` или `repository/postgres.go`
- Легко заменить реализацию (FileStorage → PostgreSQL)

## Примеры использования

### Channel Module

```go
// internal/modules/channel/service/service.go
package service

import (
    "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/domain"
    "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/repository"
)

type Service struct {
    repo repository.Repository
}

func New(repo repository.Repository) *Service {
    return &Service{repo: repo}
}

func (s *Service) AddChannel(ch *domain.Channel) error {
    // Бизнес-логика
    return s.repo.SaveChannel(ch)
}
```

### Feed Module (использует Channel и Message)

```go
// internal/modules/feed/service/service.go
package service

import (
    channelRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/channel/repository"
    messageRepo "github.com/reshetovitsme/rss-telegram-feed/internal/modules/message/repository"
)

type Service struct {
    channelRepo channelRepo.Repository
    messageRepo messageRepo.Repository
}
```

## Миграция на базу данных

Когда будете готовы к БД:

1. Создайте `repository/postgres.go` в каждом модуле
2. Реализуйте интерфейс Repository
3. Обновите DI контейнер для использования PostgreSQL
4. FileStorage останется для тестов и разработки

## Тестирование

Каждый модуль тестируется независимо:

```go
// internal/modules/channel/service/service_test.go
func TestService_AddChannel(t *testing.T) {
    mockRepo := &MockRepository{}
    service := New(mockRepo)
    // тесты...
}
```

## Преимущества для вашего проекта

✅ **Масштабируемость**: Легко выделить модуль в микросервис  
✅ **Тестируемость**: Модули тестируются изолированно  
✅ **Масштабирование инстансов**: Можно масштабировать отдельные модули  
✅ **База данных**: Легко заменить FileStorage на PostgreSQL  
✅ **Небольшая команда**: Четкая структура, легко ориентироваться
