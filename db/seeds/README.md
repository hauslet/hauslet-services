# Database Seeding Strategy

## Overview

This directory contains a comprehensive seeding system for populating the Hauslet Services database with realistic test data. The seeding strategy is designed to be:

- **Idempotent**: Can be run multiple times without creating duplicates
- **Modular**: Each module has its own seed data
- **Realistic**: Uses Nigerian-specific data (locations, currencies, etc.)
- **Relational**: Properly maintains foreign key relationships
- **Customizable**: Easy to adjust quantities and data patterns

## Directory Structure

```txt
db/seeds/
├── README.md                 # This file
├── main.go                   # Seed runner entry point
├── config.go                 # Configuration for seed quantities
├── data/                     # Reusable seed data
│   ├── nigerian_locations.go
│   ├── nigerian_names.go
│   ├── property_data.go
│   └── lorem.go
├── seeders/                  # Module-specific seeders
│   ├── auth_seeder.go
│   ├── property_seeder.go
│   ├── listing_seeder.go
│   ├── business_seeder.go
│   ├── booking_seeder.go
│   ├── review_seeder.go
│   ├── payment_seeder.go
│   └── finance_seeder.go
└── utils/                    # Helper utilities
    ├── random.go
    └── db.go
```

## Usage

### Basic Seeding

```bash
# From project root
make seed

# Or run directly with Go
go run db/seeds/main.go
```

### Custom Configuration

```bash
# Set environment variables to control seed quantities
export SEED_USERS=50
export SEED_PROPERTIES=100
export SEED_LISTINGS=150
export SEED_BOOKINGS=200
go run db/seeds/main.go
```

### Specific Module Seeding

```bash
# Seed only specific modules
go run db/seeds/main.go --modules=auth,property
```

### Reset and Reseed

```bash
# Clear all data and reseed
make fresh  # This runs: reset-db -> migrate -> seed
```

## Seeding Order

The seeders run in a specific order to maintain referential integrity:

1. **Auth** - Users and identities (base entities)
2. **Business** - Business profiles and teams
3. **Property** - Physical properties
4. **Listing** - Commercial listings (sale, rent, shortlet)
5. **Booking** - Reservations and calendar events
6. **Review** - Guest and host reviews
7. **Payment** - Transactions and wallet operations
8. **Finance** - Platform wallets and financial records

## Default Seed Quantities

| Entity | Quantity | Notes |
|--------|----------|-------|
| Users | 30 | Mix of guests, hosts, agents, admins |
| Businesses | 5 | Property management companies |
| Properties | 50 | Across Lagos, Abuja, Port Harcourt |
| Listings | 75 | 40% shortlet, 35% rent, 25% sale |
| Bookings | 100 | Past, current, and future |
| Reviews | 60 | Mix of guest and host reviews |
| Transactions | 150 | Payments, refunds, payouts |

## Features

### Nigerian-Specific Data

- **Locations**: Real neighborhoods in Lagos, Abuja, Port Harcourt, Ibadan
- **Names**: Common Nigerian names (Yoruba, Igbo, Hausa)
- **Currency**: All prices in NGN (Naira)
- **Phone Numbers**: Nigerian format (+234)

### Realistic Patterns

- **Properties**: Distribution matches real market (more apartments than villas)
- **Pricing**: Location-based (Victoria Island > Ikeja > Surulere)
- **Bookings**: Seasonal patterns (more bookings in December)
- **Reviews**: Normal distribution (mostly 4-5 stars)

### Data Relationships

- Each user has 1-3 OAuth identities
- Hosts own multiple properties
- Businesses manage properties for multiple owners
- Listings have proper media attachments
- Bookings reference valid calendar events
- Reviews link to completed bookings

## Configuration

Edit `db/seeds/config.go` to customize:

```go
type SeedConfig struct {
    UserCount      int
    PropertyCount  int
    ListingCount   int
    BookingCount   int
    BusinessCount  int
    AdminEmails    []string  // Always create these admins
}
```

## Testing Accounts

The seeder always creates these test accounts:

| Email | Password | Role | Purpose |
|-------|----------|------|---------|
| <admin@hauslet.com> | admin123 | admin | Full system access |
| <host@hauslet.com> | host123 | user | Property owner |
| <guest@hauslet.com> | guest123 | user | Regular guest |
| <agent@hauslet.com> | agent123 | user | Real estate agent |

## Idempotency

The seeder is designed to be idempotent:

- Checks for existing records before creating
- Uses upsert where appropriate
- Cleans up orphaned records
- Maintains UUID consistency

## Best Practices

1. **Run seeds on empty DB**: While idempotent, best results on fresh schema
2. **Use environment-specific configs**: Different quantities for dev/staging/test
3. **Commit seed code, not output**: Never commit actual seeded data
4. **Version control seed data**: Update seeders when schema changes
5. **Document special cases**: Add comments for complex relationships

## Troubleshooting

### Foreign Key Violations

- Ensure seeders run in correct order
- Check that parent entities exist before creating children

### Duplicate Key Errors

- Clear database before re-seeding
- Ensure unique constraints are handled

### Performance Issues

- Reduce seed quantities in config
- Use batch inserts for large datasets
- Consider disabling triggers temporarily

## Future Enhancements

- [ ] Faker library integration for more diverse data
- [ ] Seed data export/import for consistent test environments
- [ ] Performance benchmarking for seed operations
- [ ] CLI flags for granular control
- [ ] Seed data versioning for different test scenarios
