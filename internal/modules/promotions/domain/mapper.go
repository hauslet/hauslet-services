package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// These mapper functions will convert between domain models and GORM schemas
// Schema types will be defined in repository/schema package

// Helper functions for JSON marshaling/unmarshaling

// MarshalMetadata converts a map to JSON string
func MarshalMetadata(metadata map[string]interface{}) (string, error) {
	if metadata == nil {
		return "{}", nil
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalMetadata converts a JSON string to map
func UnmarshalMetadata(data string) (map[string]interface{}, error) {
	if data == "" || data == "{}" {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(data), &metadata)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// MarshalFeatures converts feature map to JSON string
func MarshalFeatures(features map[string]bool) (string, error) {
	if features == nil {
		return "{}", nil
	}
	data, err := json.Marshal(features)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalFeatures converts a JSON string to feature map
func UnmarshalFeatures(data string) (map[string]bool, error) {
	if data == "" || data == "{}" {
		return make(map[string]bool), nil
	}
	var features map[string]bool
	err := json.Unmarshal([]byte(data), &features)
	if err != nil {
		return nil, err
	}
	return features, nil
}

// UUID helper functions

// UUIDToPtr converts a UUID to a pointer
func UUIDToPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

// UUIDFromPtr converts a UUID pointer to a value
func UUIDFromPtr(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// Time helper functions

// TimeToPtr converts a time.Time to a pointer
func TimeToPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// TimeFromPtr converts a time pointer to a value
func TimeFromPtr(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
