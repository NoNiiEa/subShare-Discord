package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// --- 1. Read the Body (Optional: Remove this block if you don't want to log bodies) ---
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
		}

		// Restore the io.ReadCloser to its original state
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// ------------------------------------------------------------------------------------

		// Log the Request Details
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		if len(bodyBytes) > 0 {
			log.Printf("Body: %s", string(bodyBytes))
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)

		// Log how long it took
		log.Printf("Completed %s in %v", r.URL.Path, time.Since(start))
	})
}