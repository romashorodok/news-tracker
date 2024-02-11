package httputils

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/fx"
)

type Handler interface {
	OnRouter(http.Handler)
}

func AsHandler(groupTag string, handler any) any {
	return fx.Annotate(handler, fx.ResultTags(groupTag), fx.As(new(Handler)))
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, errorMessage ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	json.NewEncoder(w).Encode(ErrorResponse{
		Message: strings.Join(errorMessage, " "),
	})
}
