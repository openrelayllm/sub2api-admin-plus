package sitecatalog

import (
	"net/http"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func badRequest(code string, message string) error {
	return infraerrors.New(http.StatusBadRequest, code, message)
}

func internalError(message string) error {
	return infraerrors.New(http.StatusInternalServerError, "SITE_CATALOG_INTERNAL_ERROR", message)
}

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "SITE_CATALOG_DB_NOT_CONFIGURED", "site catalog database is not configured")
}
