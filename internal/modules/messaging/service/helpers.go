package service

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// Helpers for SendMessage
func copyMetadata(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	maps.Copy(out, src)
	return out
}

func marshalJSON(payload map[string]any) ([]byte, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	return json.Marshal(payload)
}

func sanitizeAttachments(inputs []MessageAttachmentInput) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(inputs))
	for _, att := range inputs {
		if strings.TrimSpace(att.URL) == "" {
			return nil, fmt.Errorf("attachment url required")
		}
		entry := map[string]any{
			"url":      att.URL,
			"filename": att.Filename,
			"type":     att.Type,
		}
		if att.Metadata != nil {
			entry["meta"] = att.Metadata
		}
		if att.ID != nil {
			entry["id"] = att.ID.String()
		}
		out = append(out, entry)
	}
	return out, nil
}

func sanitizeMetaKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, key)
}
