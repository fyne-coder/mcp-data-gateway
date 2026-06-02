package auth

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

func HTTPStatus(err error) int {
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden
	}
	return http.StatusUnauthorized
}
