package rest

import (
	"log/slog"
	"net/http"
	"tenzoflow/internal/status"
)

const ApiVersion = "v1"
const apiPrefix = "/api/" + ApiVersion

type Options struct {
	Log           *slog.Logger
	Status        *status.Service
	StatusEnabled bool
	StatusToken   string
	CORSOrigins   []string
}

// NewRouter registers all routes and wraps them in middleware.
//
//	/healthz          infrastructure probe: only the overall state, public
//	/api/v1/status    detailed report for the web UI, protected
func NewRouter(opts Options) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealth(opts.Status))
	if opts.StatusEnabled && opts.Status != nil {
		var h http.Handler = handleStatus(opts.Status)
		if opts.StatusToken != "" {
			h = requireBearerToken(opts.StatusToken)(h)
		}
		mux.Handle("GET "+apiPrefix+"/status", h)
	}
	mux.HandleFunc("/", handleNotFound)

	return chain(mux, requestID, logging(opts.Log), cors(opts.CORSOrigins), recoverer(opts.Log))
}

func handleHealth(svc *status.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		state := status.StateOK
		if svc != nil {
			state = svc.Report(r.Context()).Status
		}
		code := http.StatusOK
		if state == status.StateDown {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("cache-control", "no-store")
		writeJSON(w, code, map[string]string{"status": string(state)})
	}
}

func handleStatus(svc *status.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("cache-control", "no-store")
		writeJSON(w, http.StatusOK, svc.Report(r.Context()))
	}
}

func handleNotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "route not found")
}
