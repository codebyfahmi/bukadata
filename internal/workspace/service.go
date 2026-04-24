package workspace

import (
	"net/http"
	"strings"

	"bukadata/internal/service"
)

const HeaderWorkspaceSlug = "X-Workspace-Slug"

func ResolveSlug(r *http.Request, fallback string) (string, error) {
	header := strings.TrimSpace(r.Header.Get(HeaderWorkspaceSlug))
	bodyOrQuery := strings.TrimSpace(fallback)
	if header != "" && bodyOrQuery != "" && header != bodyOrQuery {
		return "", service.NewError(http.StatusBadRequest, "workspace_mismatch", "workspace slug mismatch between header and request payload")
	}
	if header != "" {
		return header, nil
	}
	if bodyOrQuery != "" {
		return bodyOrQuery, nil
	}
	return "", service.NewError(http.StatusBadRequest, "missing_workspace_slug", "workspace slug is required")
}
