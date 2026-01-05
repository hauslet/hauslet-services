# Hauslet Services - Database Seeding Guide

## Quick Start

```bash
# 1. Ensure your database is migrated
export DATABASE_URL="postgres://user:pass@localhost:5432/hauslet?sslmode=disable"
make migrate

# 2. Run the seeder
make seed

# 3. Or reset everything and start fresh
make fresh  # This does: reset-db -> migrate -> seed
```

## What Gets Seeded?

The seeding system creates realistic test data for your entire application:

### Users & Authentication (30 users by default)

- **Admin accounts**: `admin@hauslet.com`, `support@hauslet.com` (password: `admin123`)
- **Test accounts**:
  - `host@hauslet.com` / `host123` - For testing property listing
  - `guest@hauslet.com` / `guest123` - For testing bookings
  - `agent@hauslet.com` / `agent123` - For testing agent workflows
- **Random users**: Nigerian names, OAuth identities (Google, GitHub)

### Properties (50 properties by default)

- Spread across major Nigerian cities (Lagos, Abuja, Port Harcourt, Ibadan)
- Realistic locations: Victoria Island, Ikoyi, Lekki, Maitama, etc.
- Proper geolocation data (latitude/longitude)
- Various property types: apartments, duplexes, villas, studios
- Appropriate amenities: 24hr power, security, parking, generators, pools, etc.

### Listings (75 listings by default)

- **40% Shortlets**: For vacation/short-term stays
- **35% Rentals**: Annual or monthly rent
- **25% Sales**: Properties for sale
- Realistic pricing based on location tier:
  - Premium areas (VI, Ikoyi): ₦50M-₦500M (sale), ₦50k-₦300k/night (shortlet)
  - Mid-tier areas (Ikeja, Surulere): ₦15M-₦50M (sale), ₦20k-₦80k/night (shortlet)
  - Budget areas (Gbagada, Ojota): ₦5M-₦15M (sale), ₦8k-₦25k/night (shortlet)

### Businesses (5 companies by default)

- Property management companies
- Real estate agencies
- Team members and invitations

### Bookings (100 bookings by default)

- Past, current, and future bookings
- Various statuses: confirmed, completed, cancelled
- Proper calendar integration
- Realistic check-in/check-out dates

### Reviews (60 reviews by default)

- Guest reviews of properties/hosts
- Host reviews of guests
- Normal distribution (mostly 4-5 stars)
- Realistic Nigerian feedback

### Financial Data

- Platform wallets
- User wallets
- Transaction history
- Payment references (Paystack format)

## Configuration Options

### Environment Variables

Control seed quantities via environment variables:

```bash
export SEED_USERS=50
export SEED_PROPERTIES=100
export SEED_LISTINGS=150
export SEED_BOOKINGS=200
export SEED_BUSINESSES=10

make seed
```

### Available Flags

```bash
# Default configuration (good for development)
go run db/seeds/main.go

# Minimal configuration (fast, for testing)
go run db/seeds/main.go -config=test

# Seed only specific modules
go run db/seeds/main.go -modules=auth,property

# Clear existing data before seeding
go run db/seeds/main.go -clear

# Combine flags
go run db/seeds/main.go -config=test -clear -modules=auth,property,listing
```

## Module Seeding Order

The seeders run in this specific order to maintain referential integrity:

1. **Auth** → Users and OAuth identities
2. **Business** → Companies and teams
3. **Property** → Physical properties
4. **Listing** → Commercial listings
5. **Booking** → Reservations
6. **Review** → User reviews
7. **Finance** → Wallets and transactions

## Test Accounts

Login to your app with these pre-created accounts:

| Email | Password | Role | Use Case |
|-------|----------|------|----------|
| <admin@hauslet.com> | admin123 | Admin | Full system access, moderation |
| <host@hauslet.com> | host123 | User | Create listings, manage properties |
| <guest@hauslet.com> | guest123 | User | Browse and book properties |
| <agent@hauslet.com> | agent123 | User | Real estate agent workflows |

## Common Tasks

### Reset Everything

```bash
# Warning: This deletes ALL data!
make fresh
```

### Seed Only New Users

```bash
go run db/seeds/main.go -modules=auth
```

### Quick Test Seed (5 of everything)

```bash
go run db/seeds/main.go -config=test -clear
```

### Seed Without Bookings/Reviews (faster)

```bash
export SEED_BOOKINGS=false
export SEED_REVIEWS=false
make seed
```

## Extending the Seeders

### Add a New Module Seeder

1. Create a new file: `db/seeds/seeders/mymodule_seeder.go`
2. Implement the seeder function:

```go
package seeders

func SeedMyModule(ctx *SeedContext) error {
    config := ctx.Config.(*SeedConfig)
    
    for i := 0; i < config.MyModuleCount; i++ {
        // Create your records
        record := MyRecord{
            // ... populate fields
        }
        if err := ctx.DB.Create(&record).Error; err != nil {
            return err
        }
    }
    
    return nil
}
```

1. Add it to `db/seeds/main.go`:

```go
if shouldSeed("mymodule", moduleList) {
    runSeeder("MyModule", func() error {
        return seeders.SeedMyModule(context)
    })
}
```

### Add Custom Test Data

Edit the data files in `db/seeds/data/`:

- `nigerian_locations.go` - Add more cities/areas
- `nigerian_names.go` - Add more names
- `property_data.go` - Customize property descriptions, amenities

## Performance Tips

1. **Reduce quantities** for faster seeding:

   ```bash
   SEED_USERS=10 SEED_PROPERTIES=20 make seed
   ```

2. **Use test config** for CI/CD:

   ```bash
   go run db/seeds/main.go -config=test
   ```

3. **Disable media generation** if you don't need it:

   ```bash
   SEED_MEDIA=false make seed
   ```

4. **Seed in stages** during development:

   ```bash
   # First, seed base data
   go run db/seeds/main.go -modules=auth,property
   
   # Later, add listings
   go run db/seeds/main.go -modules=listing
   ```

## Troubleshooting

### "Failed to connect to database"

- Ensure `DATABASE_URL` is exported
- Check that PostgreSQL is running: `docker compose ps`
- Verify credentials and database name

### "Foreign key constraint violation"

- Run migrations first: `make migrate`
- Ensure you're seeding modules in order
- If in doubt, use `make fresh`

### "Table does not exist"

- You need to run migrations: `make migrate`
- Check that all migrations succeeded

### "Duplicate key error"

- The seeder is idempotent for admin accounts
- Use `-clear` flag to start fresh
- Or reset: `make fresh`

## Best Practices

1. **Never seed production databases** - This is for dev/test only
2. **Version control seed code** - Commit changes to seeders
3. **Don't commit seed output** - Never commit generated data
4. **Update seeds with schema** - When you change models, update seeders
5. **Document special data** - If you add specific test scenarios, document them

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Seed Test Database
  env:
    DATABASE_URL: postgres://test:test@localhost:5432/hauslet_test?sslmode=disable
  run: |
    make migrate
    go run db/seeds/main.go -config=test
```

### Docker Compose Integration

```yaml
services:
  db:
    image: postgres:15
    # ... postgres config
    
  seed:
    build: .
    depends_on:
      - db
    environment:
      DATABASE_URL: postgres://hauslet:hauslet@db:5432/hauslet?sslmode=disable
    command: >
      sh -c "
        sleep 5 &&
        make migrate &&
        go run db/seeds/main.go
      "
```

## Future Enhancements

- [ ] Faker library integration for more diverse data
- [ ] Export/import seed data for consistent environments
- [ ] Seed data versioning (v1, v2 datasets)
- [ ] Performance benchmarking
- [ ] Idempotent seeding for all entities (not just admins)
- [ ] More granular module control
- [ ] Seed data validation
- [ ] Custom seed scenarios (high-traffic, edge-cases, etc.)
