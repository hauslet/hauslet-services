# Seeding Quick Reference

## One-Time Setup

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/hauslet?sslmode=disable"
make migrate
```

## Daily Development

### Full Reset (Start Fresh)

```bash
make fresh
```

### Quick Seed

```bash
make seed
```

### Test Seed (Fast - 50 records total)

```bash
make seed-test
```

## Test Accounts

| Email | Password | Role |
|-------|----------|------|
| <admin@hauslet.com> | admin123 | Admin |
| <host@hauslet.com> | host123 | Host |
| <guest@hauslet.com> | guest123 | Guest |
| <agent@hauslet.com> | agent123 | Agent |

## Custom Seeding

### Control Quantities

```bash
SEED_USERS=100 SEED_PROPERTIES=200 make seed
```

### Specific Modules Only

```bash
go run db/seeds/main.go -modules=auth,property
```

### Clear and Reseed

```bash
go run db/seeds/main.go -clear
```

## What Gets Created (Default)

- 30 Users (admins, hosts, guests)
- 50 Properties (Lagos, Abuja, PH, Ibadan)
- 75 Listings (40% shortlet, 35% rent, 25% sale)
- 100 Bookings (past, current, future)
- 60 Reviews
- 5 Businesses

## Troubleshooting

### "Failed to connect"

```bash
# Check database is running
docker compose ps

# Verify DATABASE_URL
echo $DATABASE_URL
```

### "Foreign key violation"

```bash
# Ensure migrations ran
make migrate

# Or full reset
make fresh
```

### "Duplicate key error"

```bash
# Clear and reseed
make seed-clear
```

## Files to Customize

- `db/seeds/config.go` - Default quantities
- `db/seeds/data/nigerian_locations.go` - Add more locations
- `db/seeds/data/property_data.go` - Prices, amenities
- `db/seeds/seeders/*.go` - Seeder logic

## Performance Tips

```bash
# Minimal seed (< 2 seconds)
go run db/seeds/main.go -config=test

# Disable media
SEED_MEDIA=false make seed

# Skip reviews
SEED_REVIEWS=false make seed
```

## For Full Docs

- `db/seeds/README.md` - Architecture
- `db/seeds/GUIDE.md` - Complete usage guide
- `db/seeds/EXAMPLES.md` - Implementation patterns
- `db/seeds/IMPLEMENTATION_SUMMARY.md` - Overview
