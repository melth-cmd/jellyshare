package middleware

import "net/http"

// BodyLimit caps request bodies. Every API body in this app is small JSON; a
// larger one is a mistake or an attempt to make the process read forever.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders applies the browser-side protections that do not depend on the
// page's markup. HSTS is only sent over HTTPS; sending it from a plain-HTTP dev
// server would pin the browser to a scheme that is not there.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "SAMEORIGIN")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// Share links are private hand-offs, not public pages; keep them out of
		// search indexes even if an address leaks.
		header.Set("X-Robots-Tag", "noindex, nofollow")
		if requestIsHTTPS(r) {
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
