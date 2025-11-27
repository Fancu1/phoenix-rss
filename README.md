# Phoenix RSS

A modern, AI-powered RSS aggregator built on a scalable microservice architecture in Go. Phoenix RSS is engineered to showcase production-ready patterns for distributed systems, including an event-driven core and a seamlessly integrated web UI.

## Core Features

-   **Microservice Architecture**: A suite of independent, single-responsibility services (API Gateway, User, Feed, AI, Scheduler) communicating over gRPC for high performance.
-   **Integrated Web UI**: A fast, modern SvelteKit frontend is embedded directly into the API Gateway, delivering a complete application experience in a single, easy-to-deploy binary.
-   **AI-Powered Content Processing**: Leverages Large Language Models via Kafka events to automatically process and summarize article content, enhancing discoverability.
-   **Background Article Refresh**: Scheduler-driven article checks use conditional HTTP requests (ETag / Last-Modified) to refresh content only when sources change, respecting robots.txt and publishing work to Kafka for resilient processing.
-   **Event-Driven Core**: Uses Kafka for asynchronous, decoupled communication between services, ensuring resilience and scalability.
-   **Containerized & Production-Ready**: Fully containerized with Docker and orchestrated via Docker Compose, featuring healthchecks and automated initialization.

## Tech Stack

| Category                  | Technology                               |
| ------------------------- | ---------------------------------------- |
| Language                  | Go                                       |
| API Gateway & Web Serving | Gin                                      |
| Frontend                  | SvelteKit (via `adapter-static`)         |
| Service Communication     | gRPC + Protocol Buffers                  |
| Database                  | PostgreSQL                               |
| Event Bus / Messaging     | Kafka                                    |
| Containerization          | Docker & Docker Compose                  |

## Project Structure

```
.
├── web/                  # SvelteKit frontend application
├── cmd/                  # Application entry points for each service
├── internal/             # Private application and library code
├── docker/               # Per-service Dockerfiles
├── protos/               # Protocol Buffer definitions
├── db/                   # Database migrations
└── docker-compose.yml    # Docker Compose orchestration
```

## Usage

Phoenix RSS is designed to be run entirely with Docker Compose. A single command handles everything: infrastructure startup, database migrations, Kafka topic creation, and service orchestration.

### Prerequisites

-   Docker
-   Docker Compose (v2+)

### Quick Start

1. Create your `.env` file from the template:

```bash
cp env.example .env
# Edit .env with your values (e.g., AI_SERVICE_LLM_API_KEY)
```

2. Start the application:

```bash
docker compose up -d
```

That's it! The system will automatically:
- Start PostgreSQL, Redis, and Kafka with health checks
- Create required Kafka topics
- Run database migrations
- Start all application services in the correct order

The web application will be available at `http://localhost:8080`.

### Stopping the Application

```bash
docker compose down
```

### Rebuilding After Code Changes

```bash
# Rebuild specific service
docker compose build feed-service
docker compose up -d feed-service

# Rebuild all services
docker compose up --build -d
```

## Development

For local development without Docker:

```bash
# Start infrastructure only
make infra-up

# Run migrations
make migrate-up

# Run individual services
make run-api-service
make run-user-service
make run-feed-service

# Run tests
make test
```
