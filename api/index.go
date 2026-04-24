package handler

import (
	"net/http"

	"bukadata/internal/app"
)

var h = app.New()

func Handler(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTP(w, r)
}
