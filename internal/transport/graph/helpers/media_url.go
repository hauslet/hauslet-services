package helpers

import (
	"fmt"
	"strings"

	"hauslet/internal/modules/property/domain"
)

// BuildListingMediaURLs populates URLs (including thumbnails) from keys using the provided CDN host.
func BuildListingMediaURLs(media []domain.ListingMedia, cdnHost string) []domain.ListingMedia {
	host := strings.TrimSuffix(cdnHost, "/")
	out := make([]domain.ListingMedia, len(media))
	for i, m := range media {
		out[i] = m
		if host != "" {
			if m.URL == "" && (strings.HasPrefix(m.Key, "http://") || strings.HasPrefix(m.Key, "https://")) {
				out[i].URL = m.Key
			} else if m.URL == "" && m.Key != "" {
				out[i].URL = fmt.Sprintf("%s/%s", host, strings.TrimPrefix(m.Key, "/"))
			} else if m.URL != "" && !strings.HasPrefix(m.URL, "http") {
				out[i].URL = fmt.Sprintf("%s/%s", host, strings.TrimPrefix(m.URL, "/"))
			}
			if len(m.Thumbnails) > 0 {
				thumbs := make(domain.ThumbnailMap, len(m.Thumbnails))
				for name, t := range m.Thumbnails {
					thumb := t
					if t.URL == "" && (strings.HasPrefix(t.Key, "http://") || strings.HasPrefix(t.Key, "https://")) {
						thumb.URL = t.Key
					} else if t.URL == "" && t.Key != "" {
						thumb.URL = fmt.Sprintf("%s/%s", host, strings.TrimPrefix(t.Key, "/"))
					} else if t.URL != "" && !strings.HasPrefix(t.URL, "http") {
						thumb.URL = fmt.Sprintf("%s/%s", host, strings.TrimPrefix(t.URL, "/"))
					}
					thumbs[name] = thumb
				}
				out[i].Thumbnails = thumbs
			}
		}
	}
	return out
}
