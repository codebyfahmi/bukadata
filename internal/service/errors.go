package service

import (
	"context"
	"errors"
	"net/http"

	"bukadata/internal/httpx"
)

type contextKey string

const principalContextKey contextKey = "principal"

func NewError(status int, code, message string) error {
	return httpx.NewError(status, code, message)
}

func WriteServiceError(w http.ResponseWriter, err error) {
	var svcErr *httpx.Error
	if errors.As(err, &svcErr) {
		httpx.WriteError(w, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func principalFromContext(ctx context.Context) any {
	return ctx.Value(principalContextKey)
}
