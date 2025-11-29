# GraphQL Module

This directory is intended to hold the GraphQL implementation for the Hauslet API. As of now, this module is not yet implemented.

## Planned Purpose

This module will be responsible for defining the GraphQL schema, generating Go models and resolvers, and connecting the GraphQL engine to the application's business logic.

## Planned Technology

- **Engine**: [`99designs/gqlgen`](https://github.com/99designs/gqlgen) - A schema-first GraphQL library for Go.

## Planned Structure

```
/graph
├── schema.graphqls     # GraphQL schema definitions
├── generated.go        # Auto-generated code from gqlgen
├── resolver.go         # Root resolver struct
└── *.resolvers.go      # Resolver implementations
```