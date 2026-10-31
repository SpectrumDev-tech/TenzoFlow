package rest

import (
	"log/slog"
	"net/http"
)

func NewRouter(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	//
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("/", handleNotFound)

	return chain(mux, requestID, logging(log), recoverer(log))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "route not found")
}
