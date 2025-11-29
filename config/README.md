# Configuration System

This directory contains the hybrid configuration system for the Hauslet services.

## Overview

The configuration system uses two sources:
1. **Environment Variables** (`.env`) - For secrets and deployment-specific values
2. **YAML Files** (`config/defaults/*.yaml`) - For service configuration and business logic

## Directory Structure

```
config/
├── config.go              # ENV variable loader
├── structs.go             # Configuration type definitions
├── yaml.go                # YAML configuration loader
├── defaults/              # Default YAML configurations (committed to git)
│   ├── calendar.yaml     # Calendar service settings
│   ├── features.yaml     # Feature flags
│   └── queue.yaml        # Queue/NATS settings
└── overrides/             # User-specific overrides (gitignored)
    └── *.yaml            # Optional: override any default config
```

## Configuration Categories

### Environment Variables (.env)
**Purpose:** Deployment-specific values and secrets

**Examples:**
- Database credentials (`DB_PASSWORD`, `DB_HOST`)
- API keys (`GOOGLE_CLIENT_SECRET`, `RESEND_API_KEY`)
- Service URLs (`NATS_URL`, `REDIS_ADDR`, `ELASTICSEARCH_URL`)
- Encryption keys (`JWT_SECRET`, `ENCRYPTION_KEY`)

**Access in code:**
```go
cfg := config.Load()
dbPassword := cfg.Storage.DB.DBPassword  // From ENV
natsURL := cfg.Infra.NATS.URL            // From ENV
```

### YAML Configuration (config/defaults/*.yaml)
**Purpose:** Service behavior, feature flags, and business rules

**Examples:**
- Calendar settings (maintenance hour, cleanup days)
- Queue subjects and consumers
- Feature flags (email queue enabled, OAuth enabled)

**Access in code:**
```go
cfg := config.Load()
maintenanceHour := cfg.YAML.Calendar.MaintenanceHour  // From YAML
emailSubject := cfg.YAML.Queue.Subjects["email"]      // From YAML
```

## Configuration Files

### calendar.yaml
Calendar service settings:
- `maintenance_hour` - Hour for maintenance operations (0-23)
- `weekly_check_day` - Day for weekly checks (0=Sunday)
- `cleanup_old_days` - Days to keep old data
- `scheduler_enabled` - Enable/disable scheduler
- `immediate_generation` - Generate calendars immediately
- `retry_delay_minutes` - Retry delay for failed ops

### queue.yaml
Queue/NATS JetStream settings:
- `stream_name` - JetStream stream name
- `subjects` - Map of job types to NATS subjects
  - `email`: Email sending jobs
  - `notification`: Notification jobs
  - `moderation`: Moderation jobs
- `consumers` - Map of consumer names per job type

### features.yaml
Feature toggle flags:
- `email_queue_enabled` - Use queue for email sending
- `oauth_enabled` - Enable OAuth authentication
- `rate_limiting_enabled` - Enable rate limiting

## Overriding Defaults

To override default configurations without modifying the defaults:

1. Create `config/overrides/` directory (it's gitignored)
2. Copy the file you want to override from `defaults/` to `overrides/`
3. Modify only the values you want to change

**Example:**
```yaml
# config/overrides/queue.yaml
queue:
  stream_name: JOBS_DEV  # Override for local development
  subjects:
    email: "email.send.dev"
```

The override will merge with defaults, replacing only the specified values.

## Usage

### Loading Configuration

```go
package main

import "hauslet/config"

func main() {
    cfg := config.Load()

    // Access ENV-based config
    dbHost := cfg.Storage.DB.DBHost
    natsURL := cfg.Infra.NATS.URL

    // Access YAML-based config
    streamName := cfg.YAML.Queue.StreamName
    emailSubject := cfg.YAML.Queue.Subjects["email"]
    maintenanceHour := cfg.YAML.Calendar.MaintenanceHour
}
```

### Common Patterns

**Queue Setup:**
```go
emailSubject := cfg.YAML.Queue.Subjects["email"]
q, err := queue.New(ctx, cfg.Infra.NATS.URL, cfg.YAML.Queue.StreamName, []string{emailSubject})
```

**Worker Setup:**
```go
consumerName := cfg.YAML.Queue.Consumers["email"]
emailSubject := cfg.YAML.Queue.Subjects["email"]

consumerCfg := jetstream.ConsumerConfig{
    Name:          consumerName,
    Durable:       consumerName,
    FilterSubject: emailSubject,
}
```

## Migration from ENV-only Config

**Before (ENV):**
```bash
NATS_STREAM=EMAILS
NATS_SUBJECT_SEND=email.send
CALENDAR_MAINTENANCE_HOUR=2
```

```go
streamName := cfg.Infra.NATS.StreamName
subject := cfg.Infra.NATS.SubjectSend
```

**After (YAML):**
```yaml
# config/defaults/queue.yaml
queue:
  stream_name: JOBS
  subjects:
    email: "email.send"
```

```go
streamName := cfg.YAML.Queue.StreamName
subject := cfg.YAML.Queue.Subjects["email"]
```

## Benefits

1. **Separation of Concerns**
   - Secrets stay in ENV (never committed)
   - Business logic in YAML (versioned, reviewable)

2. **Type Safety**
   - YAML configs are strongly typed
   - Validated at load time

3. **Environment Independence**
   - Same YAML works across all environments
   - Only ENV vars change per deployment

4. **Override Support**
   - Developers can customize locally
   - No risk of committing secrets

5. **Documentation**
   - YAML files serve as living documentation
   - Comments explain each setting

## Troubleshooting

**Config load failures:**
```bash
# Check if YAML files exist
ls -la config/defaults/

# Validate YAML syntax
yamllint config/defaults/*.yaml
```

**Override not working:**
- Ensure override file is in `config/overrides/` (not `defaults/`)
- Check YAML indentation (use spaces, not tabs)
- Verify override structure matches defaults

**Missing values:**
- Check if ENV variable is set: `echo $NATS_URL`
- Verify YAML file exists and is readable
- Review config loading logs for errors
