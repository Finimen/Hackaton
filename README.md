<div align="center">
# CheckMesh - Site "Checker" Platform 🔊

![Status](https://img.shields.io/badge/Status-Active-success)
![License](https://img.shields.io/badge/License-Proprietary-red)
![Version](https://img.shields.io/badge/Version-0.1.6B-blue)
![Platform](https://img.shields.io/badge/Platform-Web-informational)

A solution that will make checking "Why a website isn't loading" a snap.

</div>

### The Eliminaters
#### **Сапсай К.В.** - 📧 finimensniper@gmail.com
- Created the agent features (core) part of the project
#### **Печерикин Д.Д** - pecherikindanielman@mail.ru
- Created the backend part of the project
#### **Карпенко Д.В. - Yahooilla@yandex.ru
- Created the frontend part of the project


## 🚀 Features

### Core Functionality
- **HTTP Check**           - Avg timeout < 1s
- **Ping Check**           - Avg timeout < 75ms, PL = 0%
- **DNS Check**            - Avg timeout < 100ms
- **TCP Check**            - Avg timeout < 100ms
- **HTTPS with SSL Check** - Avg timeout < 200ms

### Technical Features
- **TODO** - TODO

## 🛠 Tech Stack

### Backend
- **Go 1.21+** - Primary programming language
- **Gin** - HTTP web framework
- **PostgreSQL** - Primary database
- **Redis** - Caching and token blacklisting
- **Redis Pub/Sub** - Queue for connection backend part with agent one

### Infrastructure
- **Docker** - Containerization
- **Viper** - Configuration management

## 📁 Project Structure

```
soundtube/
├── cmd/
│   ├── agent/              # Entry point of agent part
│   │   └── tests/
│   └── backend/            # Entry point of backend part
├── docs/                   
├── internal/
│   └── backend/ 
│   │   ├── handlers/       # End points layer
│   │   ├── services/       # Business logic layer
│   │   ├── dependencies/   # DI implementation
│   │   ├── models/         # Models layer
│   │   ├── server/         # Server layer
│   │   └── storage/        # Data access layer
│   └── agent/
│   │   ├── clients/        # API of the agent
│   │   ├── domain/         # Business logic layer
│   │   ├── handlers/       # Queue endpoints layer
│   │   └── runners/        # Checks layer
├── pkg/
│   ├── config/             # Configuration management
│   ├── middleware/         # HTTP middleware
│   └── utils/              # Shared utilities
├── configs/                # Configuration files
├── scripts/                # Small features for app in general
└── static/                 # Static files and uploads

```

---

## 🔧 API Documentation

### Authentication Endpoints
<div align="center">

# TODO

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register new user |
| POST | `/api/auth/login` | User login |
| POST | `/api/auth/logout` | User logout |
| GET | `/api/auth/verify-email` | Verify email address |

### Sounds Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/sounds` | Get all sounds |
| POST | `/api/sounds` | Create sound record |
| POST | `/api/sounds/upload` | Upload audio file |
| PATCH | `/api/sounds/{id}` | Update sound |
| DELETE | `/api/sounds/{id}` | Delete sound |

### Reactions Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| PUT | `/api/sounds/{id}/reactions` | Add reaction to sound |
| DELETE | `/api/sounds/{id}/reactions` | Remove reaction from sound |
| GET | `/api/sounds/{id}/reactions` | Get sound reactions |

</div>


## 🎯 Division into parts

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

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/Finimen/Soundtube/blob/main/License.md) file for details.

