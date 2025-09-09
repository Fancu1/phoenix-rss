# Phoenix RSS

A modern, scalable RSS aggregator built on a microservice architecture in Go. This project showcases best practices in building distributed, cloud-native systems with AI-powered content processing capabilities.

## Architecture Philosophy

Phoenix RSS is built upon a foundation of clean architecture and domain-driven design principles. Our goal is to create a system that is not just functional, but also maintainable, scalable, and a pleasure to develop.

-   **Service Independence**: Each microservice (API Gateway, User Service, Feed Service, AI Service) is a self-contained unit with a single responsibility, its own data schema, and an independent deployment lifecycle.
-   **Asynchronous & Event-Driven**: We use Kafka as the central nervous system of our application. Services communicate asynchronously through events, ensuring loose coupling and high resilience.
-   **AI-Powered Content Processing**: Integrated AI service automatically processes RSS articles to generate summaries and enhance content discoverability through Large Language Model APIs.
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
│   ├── scheduler-service/ # Scheduler Service main
│   └── ai-service/       # AI Processing Service main
├── internal/             # Private application and library code, isolated per service
│   ├── api-service/      # API Gateway private logic (gRPC clients, HTTP handlers)
│   ├── feed-service/     # Feed service domain logic, handlers, and repository
│   ├── user-service/     # User service domain logic, handlers, and repository
│   ├── scheduler-service/ # Scheduler service logic and gRPC clients
│   └── ai-service/       # AI service with LLM integration and event processing
├── pkg/                  # Shared libraries intended for use across services
│   ├── logger/           # Centralized logging utility
│   └── ierr/             # Standardized error handling package
├── protos/               # Protocol Buffer definitions and generated Go code
├── configs/              # Project configuration files (DEPRECATED - see below)
└── docker-compose.yml    # Docker Compose setup for local development
```

## Configuration

This project uses a simple and powerful layered configuration system, making it easy to run in any environment.

### Configuration Priority (Highest to Lowest)

1.  **Environment Variables**: The ultimate override, perfect for production and CI/CD systems (e.g., `DATABASE_HOST=...`).
2.  **.env File**: The primary method for local development. Contains all user-specific overrides. **This is the only file you need to edit.**
3.  **Code Defaults**: Default values are hard-coded directly in `internal/config/config.go`. They serve as a reliable baseline and ensure the application can always start.

The `configs/config.yaml` file has been **deprecated and removed** in favor of this cleaner, code-first approach.

### Quick Start Guide

#### 1. Create Your Local Configuration

Copy the example environment file. This is the **only configuration step** you need to do.

```bash
cp env.example .env
```

#### 2. Edit `.env`

Open the newly created `.env` file and customize the values as needed, such as your database password or AI service API key.

```dotenv
# .env - Example customizations
DATABASE_PASSWORD=mysecretpassword
AI_SERVICE_LLM_API_KEY=sk-your-real-api-key
```

#### 3. Run the Application

```bash
# Build and run all services using Docker Compose
docker-compose up --build -d
```

### How It Works

-   The **`docker-compose.yml`** file uses the `env_file` directive to automatically load all variables from your `.env` file into each service's container.
-   The Go application, using the Viper library, reads these environment variables at startup and uses them to populate its configuration struct.
-   If an environment variable is not set in either the system or the `.env` file, the application falls back to the safe default values defined in the code.

This architecture ensures that the configuration is explicit, easy to manage, and free of redundancy. Adding a new configuration variable is as simple as adding a field to the Go struct and a corresponding entry in the `.env.example` file. **The `docker-compose.yml` file never needs to be changed.**

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

# Start AI Service (Background)
go run ./cmd/ai-service/main.go

# Start the API Gateway (HTTP)
go run ./cmd/api-service/main.go
```
