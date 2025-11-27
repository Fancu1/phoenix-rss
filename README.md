# Phoenix RSS

A modern, AI-powered RSS aggregator built on a scalable microservice architecture in Go. Phoenix RSS is engineered to showcase production-ready patterns for distributed systems, including an event-driven core and a seamlessly integrated web UI.

## Core Features

-   **Microservice Architecture**: A suite of independent, single-responsibility services (API Gateway, User, Feed, AI, Scheduler) communicating over gRPC for high performance.
-   **Integrated Web UI**: A fast, modern SvelteKit frontend is embedded directly into the API Gateway, delivering a complete application experience in a single, easy-to-deploy binary.
-   **AI-Powered Content Processing**: Leverages Large Language Models via Kafka events to automatically process and summarize article content, enhancing discoverability.
-   **Background Article Refresh**: Scheduler-driven article checks use conditional HTTP requests (ETag / Last-Modified) to refresh content only when sources change, respecting robots.txt and publishing work to Kafka for resilient processing.
-   **Event-Driven Core**: Uses Kafka for asynchronous, decoupled communication between services, ensuring resilience and scalability.
-   **Containerized & Production-Ready**: Fully containerized with Docker and orchestrated via Docker Compose, featuring healthchecks and a streamlined multi-stage build process.

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
├── docs/                 # Project design and architecture documents
├── protos/               # Protocol Buffer definitions
├── db/                   # Database migrations
└── docker-compose.yml    # Docker Compose setup
```

## Usage

Phoenix RSS is designed to be run entirely with Docker Compose.

### Prerequisites

-   Docker
-   Docker Compose (v2+)

### 1. Initial Setup & Migration

First, create your `.env` file from the template. You only need to do this once.

```bash
cp env.example .env
# Remember to edit .env with your sensitive values (e.g., API keys)
```

Next, run the database migrations:

```bash
docker compose run --rm migrator up
```

### 2. Starting All Services

To build and start the entire application stack for the first time or after major changes, run:

```bash
docker compose up --build -d
```

The web application will be available at `http://localhost:8080`.

### 3. Stopping the Application

```bash
docker compose down
```
