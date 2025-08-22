# Phoenix RSS

A modern, scalable RSS aggregator built on a microservice architecture in Go. This project is designed to showcase best practices in building distributed, cloud-native systems.

## Architecture Philosophy

Phoenix RSS is built upon a foundation of clean architecture and domain-driven design principles. Our goal is to create a system that is not just functional, but also maintainable, scalable, and a pleasure to develop.

-   **Service Independence**: Each microservice (API Gateway, User Service, Feed Service) is a self-contained unit with a single responsibility, its own data schema, and an independent deployment lifecycle.
-   **Asynchronous & Event-Driven**: We use Kafka as the central nervous system of our application. Services communicate asynchronously through events, ensuring loose coupling and high resilience.
-   **API-First Design**: Communication between services is contract-driven, using gRPC and Protocol Buffers for high-performance, strongly-typed internal APIs. The public-facing API is a RESTful interface managed by the API Gateway.
-   **Observability by Design**: (Planned) The system is being built with observability in mind, preparing for integration with Prometheus and Grafana to provide deep insights into application performance and health.

## Tech Stack

-   **Language**: Go
-   **API Gateway**: Gin
-   **Microservice Communication**: gRPC + Protocol Buffers
-   **Database**: PostgreSQL
-   **Event Bus**: Kafka
-   **Containerization**: Docker & Docker Compose

## Directory Structure

The project follows the standard Go project layout to ensure a clean separation of concerns.

```
.
├── cmd/                  # Application entry points for each service
│   ├── api-service/      # API Gateway main
│   ├── user-service/     # User Service main
│   ├── feed-service/     # Feed Service main
│   └── scheduler-service/ # Scheduler Service main
├── internal/             # Private application and library code, isolated per service
│   ├── api-service/      # API Gateway private logic (gRPC clients, HTTP handlers)
│   ├── feed-service/     # Feed service domain logic, handlers, and repository
│   ├── user-service/     # User service domain logic, handlers, and repository
│   └── scheduler-service/ # Scheduler service logic and gRPC clients
├── pkg/                  # Shared libraries intended for use across services
│   ├── logger/           # Centralized logging utility
│   └── ierr/             # Standardized error handling package
├── protos/               # Protocol Buffer definitions and generated Go code
├── configs/              # Project configuration files
└── docker-compose.yml    # Docker Compose setup for local development
```

## Getting Started

The application consists of multiple services that must be run independently. For local development, using Docker Compose is the recommended approach.

### 1. Start Dependent Services

Ensure Docker is running and start the infrastructure:

```bash
docker-compose up -d postgres kafka
```

### 2. Run Database Migrations

Apply the latest database schema.

```bash
go run ./cmd/migrator up
```

### 3. Run the Services

Services must be started in the correct order. Open a new terminal for each service.

```bash
# Start User Service (gRPC)
go run ./cmd/user-service/main.go

# Start Feed Service (gRPC)
go run ./cmd/feed-service/main.go

# Start Scheduler Service (Background)
go run ./cmd/scheduler-service/main.go

# Start the API Gateway (HTTP)
go run ./cmd/api-service/main.go
```
