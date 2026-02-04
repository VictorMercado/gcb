package main

import (
	"log"
	"net"
	"net/http"
	"strings"
)

// AuthMiddleware validates API key and optionally IP address
func AuthMiddleware(apiKey string, allowedIPs []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check API Key
			providedKey := r.Header.Get("X-API-Key")
			log.Println("Request : ", r)
			log.Println("Provided API Key: " + providedKey)
			log.Println("API Key: " + apiKey)
			if providedKey == "" || providedKey != apiKey {
				// Stealth mode: ignore request to hide server existence
				if hj, ok := w.(http.Hijacker); ok {
					if conn, _, err := hj.Hijack(); err == nil {
						log.Println("🔒 Stealth mode: Request ignored due to invalid API key")
						conn.Close()
						return
					}
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}

			// Check IP whitelist (if configured)
			if len(allowedIPs) > 0 {
				clientIP := getClientIP(r)
				if !isIPAllowed(clientIP, allowedIPs) {
					// Stealth mode for IP mismatch too
					if hj, ok := w.(http.Hijacker); ok {
						if conn, _, err := hj.Hijack(); err == nil {
							log.Println("🔒 Stealth mode: Request ignored due to invalid IP address")
							conn.Close()
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
					return
				}
			}

			// Authentication successful, proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client's real IP address from the request
// Priority: CF-Connecting-IP > X-Real-IP > X-Forwarded-For > RemoteAddr
func getClientIP(r *http.Request) string {
	// Check for Cloudflare's CF-Connecting-IP header (highest priority for CF proxy/tunnel)
	cfIP := r.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		return cfIP
	}

	// Check for X-Real-IP header (often set by reverse proxies)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Check for X-Forwarded-For header (common with proxies/load balancers)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Get the first IP in the list (original client IP)
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isIPAllowed checks if the client IP is in the whitelist
func isIPAllowed(clientIP string, allowedIPs []string) bool {
	parsedClientIP := net.ParseIP(clientIP)
	if parsedClientIP == nil {
		return false
	}

	for _, allowedIP := range allowedIPs {
		// Check if it's a CIDR range
		if strings.Contains(allowedIP, "/") {
			_, ipNet, err := net.ParseCIDR(allowedIP)
			if err != nil {
				continue
			}
			if ipNet.Contains(parsedClientIP) {
				return true
			}
		} else {
			// Direct IP comparison
			if clientIP == allowedIP {
				return true
			}
		}
	}
	return false
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			if isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
		return true
	}

	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
