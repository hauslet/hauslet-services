# Database

This directory is intended for database-related assets.

## Purpose

This directory should contain all raw SQL migration files, schema definitions, and database-related scripts. Using a dedicated directory for these assets helps separate the database schema evolution from the application's business logic.

## Migrations

When using a migration tool like `golang-migrate/migrate` or `pressly/goose`, the SQL migration files should be placed here.

### Example Structure

```
/db
├── migrations/
│   ├── 001_create_users_table.up.sql
│   ├── 001_create_users_table.down.sql
│   ├── 002_create_properties_table.up.sql
│   └── 002_create_properties_table.down.sql
└── seeds/
    └── development.sql
```
