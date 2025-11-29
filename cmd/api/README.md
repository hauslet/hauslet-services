# API Server

This directory contains the main entry point for the Hauslet HTTP/GraphQL API server.

## Purpose

The `api` server is responsible for:
- Initializing the application configuration.
- Setting up database, cache, and queue connections.
- Configuring the HTTP router (`chi`) and its middleware.
- Registering all API routes (REST and GraphQL).
- Starting and gracefully shutting down the HTTP server.

## How to Run

To start the API server, run the following command from the project root:

```bash
go run ./cmd/api/main.go
```

The server will start on the host and port specified in your configuration (e.g., `localhost:8080`).

## Key Components

- `main.go`: The application entry point.
- `server/server.go`: Contains the core server setup logic.
- `server/routes.go`: Defines all API routes and maps them to their handlers.
- `server/middleware/`: Contains custom HTTP middleware used by the router.
