
# CheckMesh - Архитектура проекта

## 🏗️ Общая архитектура



## 📁 Структура проекта

- **Hackaton/**
    - **cmd/** — точки входа приложений
        - **backend/** — запуск API сервера
            - `main.go`
        - **agent/** — запуск агента проверок
            - `main.go`
    - **internal/** — внутренняя логика приложения
        - **backend/** — логика серверной части
            - **handlers/** — HTTP и WebSocket обработчики
                - `checks.go` — обработка проверок
                - `agents.go` — управление агентами
                - `websocket.go` — соединение в реальном времени
            - **services/** — бизнес-логика
                - `check_service.go`
                - `agent_service.go`
                - `queue_service.go`
            - **storage/** — работа с базой данных и очередями
                - `database.go`
                - `check_store.go`
                - `agent_store.go`
                - `redis_queue.go`
            - **models/** — структуры данных
                - `check.go`
                - `agent.go`
                - `result.go`
        - **agent/** — логика клиента-агента
            - **worker/** — обработка задач
                - `task_processor.go`
                - `heartbeat.go`
            - **checks/** — реализации проверок
                - `http_check.go`
                - `ping_check.go`
                - `tcp_check.go`
                - `dns_check.go`
                - `traceroute.go`
            - **client/** — клиент API
                - `api_client.go`
    - **pkg/** — общие утилиты
        - **uuid/** — генерация UUID
        - **validator/** — валидация данных
        - **logger/** — логирование
    - **frontend/** — веб-интерфейс (Vue.js)
        - **src/**
            - **components/**
                - `CheckForm.vue`
                - `ResultsMap.vue`
                - `AgentsStatus.vue`
            - **views/**
                - `Dashboard.vue`
                - `History.vue`
            - **api/**
                - `client.js`
        - `package.json`
    - **deployments/** — конфигурации развёртывания
        - **backend/**
            - `Dockerfile`
            - `config.yaml`
        - **agent/**
            - `Dockerfile`
            - `agent-config.yaml`
        - **database/**
            - `init.sql`
    - **scripts/** — вспомогательные скрипты
        - `setup-local.sh`
        - `start-agents.sh`
    - `go.mod`
    - `docker-compose.yml`
    - `README.md`





## 🎯 Ключевые компоненты

### Backend (Go)
- **HTTP Server** - REST API + WebSocket
- **Check Service** - управление проверками
- **Agent Service** - регистрация и мониторинг агентов  
- **Queue Service** - распределение задач
- **Storage** - работа с PostgreSQL и Redis

### Agent (Go)
- **Task Worker** - обработчик задач из очереди
- **Check Executors** - выполнение сетевых проверок
- **API Client** - отправка результатов
- **Heartbeat** - мониторинг состояния

### Frontend (SPA)
- **Check Form** - создание проверок
- **Results Map** - визуализация на карте
- **Real-time Updates** - WebSocket соединения
- **Agents Status** - мониторинг агентов

## 💾 Хранение данных

### PostgreSQL Таблицы
- **checks** - информация о проверках
- **check_results** - результаты выполнения
- **agents** - зарегистрированные агенты

### Redis
- **check_tasks** - очередь задач
- **agent_heartbeats** - статусы агентов



## 🔧 Технологии

- **Backend**: Go, Gorilla Mux, WebSocket
- **Agent**: Go, стандартные сетевые библиотеки
- **Frontend**: React/Vue, WebSocket
- **Базы данных**: PostgreSQL, Redis
- **Очередь**: Redis Pub/Sub
- **Контейнеризация**: Docker
