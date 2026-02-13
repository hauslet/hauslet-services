package middleware

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

// Intercept2FARedirect is a middleware that appends 2FA query parameters to
// the OAuth redirect URL when a 2FA-enabled user logs in via an OAuth provider.
//
// go-pkgz/auth redirects the browser to the original "from" URL after the
// OAuth callback. By the time the redirect is issued, the JWT cookie already
// contains 2FA pending attributes (set by ClaimsUpdater / EnrichClaims).
// This middleware intercepts the 307 response, reads those attributes from
// the Set-Cookie header, and appends them as query parameters to the Location
// URL. This allows the client to detect 2FA on first render without waiting
// for a /me round-trip.
func Intercept2FARedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only intercept OAuth callback endpoints
		if !isOAuthCallback(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		interceptor := &redirectInterceptor{ResponseWriter: w}
		next.ServeHTTP(interceptor, r)
	})
}

// redirectInterceptor wraps http.ResponseWriter to modify redirect Location
// headers when 2FA is pending.
type redirectInterceptor struct {
	http.ResponseWriter
}

// WriteHeader intercepts redirect responses to inject 2FA query parameters.
// The JWT cookie is already set on the response by go-pkgz/auth's Set()
// before the redirect is issued, so we can safely read it here.
func (ri *redirectInterceptor) WriteHeader(code int) {
	if code == http.StatusTemporaryRedirect {
		if params := extract2FAFromResponseCookies(ri.Header()); params != nil {
			if location := ri.Header().Get("Location"); location != "" {
				ri.Header().Set("Location", append2FAParams(location, params))
			}
		}
	}
	ri.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the underlying ResponseWriter for middleware compatibility
// (e.g., http.Flusher, http.Hijacker detection).
func (ri *redirectInterceptor) Unwrap() http.ResponseWriter {
	return ri.ResponseWriter
}

// twoFARedirectParams holds 2FA attributes extracted from the JWT cookie.
type twoFARedirectParams struct {
	TempToken string
	Method    string
	Email     string
}

// extract2FAFromResponseCookies reads Set-Cookie headers from the response,
// finds the JWT cookie, decodes its payload, and extracts 2FA attributes.
func extract2FAFromResponseCookies(headers http.Header) *twoFARedirectParams {
	for _, setCookie := range headers.Values("Set-Cookie") {
		if !strings.HasPrefix(setCookie, "JWT=") {
			continue
		}

		// Extract cookie value (between "JWT=" and the first ";")
		value := setCookie[len("JWT="):]
		if idx := strings.Index(value, ";"); idx >= 0 {
			value = value[:idx]
		}

		// Cookie values are URL-encoded by http.SetCookie
		decoded, err := url.QueryUnescape(value)
		if err != nil {
			continue
		}

		return decode2FAFromJWTPayload(decoded)
	}
	return nil
}

// decode2FAFromJWTPayload extracts 2FA attributes from a JWT's payload
// without verifying the signature. This is safe because the middleware runs
// server-side and the cookie was just created by us in the same request.
func decode2FAFromJWTPayload(jwtToken string) *twoFARedirectParams {
	parts := strings.SplitN(jwtToken, ".", 3)
	if len(parts) != 3 {
		return nil
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	// Minimal structure matching go-pkgz/auth's token.Claims JSON layout:
	//   { "user": { "attrs": { "login_state": "...", ... } } }
	var claims struct {
		User *struct {
			Attrs map[string]interface{} `json:"attrs"`
		} `json:"user"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	if claims.User == nil || claims.User.Attrs == nil {
		return nil
	}

	loginState, _ := claims.User.Attrs["login_state"].(string)
	if loginState != "2fa_pending" {
		return nil
	}

	tempToken, _ := claims.User.Attrs["2fa_temp_token"].(string)
	if tempToken == "" {
		return nil
	}

	return &twoFARedirectParams{
		TempToken: tempToken,
		Method:    strOrEmpty(claims.User.Attrs, "2fa_method"),
		Email:     strOrEmpty(claims.User.Attrs, "2fa_email"),
	}
}

// append2FAParams adds 2FA query parameters to a redirect URL.
func append2FAParams(location string, params *twoFARedirectParams) string {
	parsed, err := url.Parse(location)
	if err != nil {
		return location
	}

	q := parsed.Query()
	q.Set("2fa_pending", "true")
	q.Set("temp_token", params.TempToken)
	q.Set("method", params.Method)
	if params.Email != "" {
		q.Set("email", params.Email)
	}
	parsed.RawQuery = q.Encode()

	return parsed.String()
}

// isOAuthCallback checks if the request path is an OAuth callback endpoint.
func isOAuthCallback(path string) bool {
	return strings.HasSuffix(path, "/callback")
}

func strOrEmpty(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
