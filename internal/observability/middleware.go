package observability

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func RequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		sw.Header().Set("X-Request-Id", requestID)
		next.ServeHTTP(sw, r)
		payload := map[string]any{
			"request_id":     requestID,
			"method":         r.Method,
			"path":           r.URL.Path,
			"workspace_slug": r.Header.Get("X-Workspace-Slug"),
			"marketplace":    r.URL.Query().Get("marketplace"),
			"shop_key":       r.URL.Query().Get("shop_key"),
			"status_code":    sw.status,
			"duration_ms":    time.Since(start).Milliseconds(),
		}
		line, _ := json.Marshal(payload)
		log.Println(string(line))
	})
}

func newRequestID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
