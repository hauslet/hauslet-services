package seeders

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func parseOwnerIDsCSV(raw string) ([]uuid.UUID, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	seen := make(map[uuid.UUID]bool, len(parts))
	result := make([]uuid.UUID, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		id, err := uuid.Parse(value)
		if err != nil {
			return nil, fmt.Errorf("invalid owner id %q: %w", value, err)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}

	return result, nil
}

func resolveTargetOwnerIDs(raw string, available []uuid.UUID) ([]uuid.UUID, error) {
	if len(available) == 0 {
		return nil, fmt.Errorf("no available owners found")
	}

	requested, err := parseOwnerIDsCSV(raw)
	if err != nil {
		return nil, err
	}
	if len(requested) == 0 {
		return available, nil
	}

	availableSet := make(map[uuid.UUID]bool, len(available))
	for _, id := range available {
		availableSet[id] = true
	}

	filtered := make([]uuid.UUID, 0, len(requested))
	for _, id := range requested {
		if availableSet[id] {
			filtered = append(filtered, id)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("none of the configured SEED_LISTING_OWNER_IDS exist in users table")
	}

	return filtered, nil
}
