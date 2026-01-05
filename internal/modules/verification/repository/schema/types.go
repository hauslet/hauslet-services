package schema

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSONBMap is a custom type for JSONB columns
type JSONBMap map[string]any

// Value implements the driver.Valuer interface
func (j JSONBMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONBMap) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONB value")
	}

	result := make(map[string]any)
	err := json.Unmarshal(bytes, &result)
	*j = JSONBMap(result)
	return err
}
