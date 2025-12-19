package middleware

import (
	"net/http"
	"os"
    "strings"
    "github.com/NoNiiEa/subShare-Discord/src/helper" // Import your helper for JSON errors
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

        // Example: Expecting "Bearer my-secret-token"
        // If you just want a simple API Key, you might use a custom header like "X-API-KEY"
        
        // 2. Validate format (Optional, but good practice for Bearer tokens)
        if !strings.HasPrefix(authHeader, "Bearer ") {
            helper.WriteJSON(w, http.StatusUnauthorized, "Missing or invalid Authorization header format")
            return
        }

        token := strings.TrimPrefix(authHeader, "Bearer ")

		// 3. Check against your secret (Load this from ENV in production!)
		expectedToken := os.Getenv("API_SECRET_TOKEN")
        if expectedToken == "" {
            // Fallback for development if env is missing
            expectedToken = "super-secret-password" 
        }

		if token != expectedToken {
            helper.WriteJSON(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// 4. Success! Pass to the next handler
		next.ServeHTTP(w, r)
	})
}