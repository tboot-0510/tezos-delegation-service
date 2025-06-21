package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"tezos-delegation-service/internal/middleware"
	"tezos-delegation-service/internal/service"

	"github.com/gorilla/mux"
)

type ApiServer struct {
	svc service.XtzService
}

func NewApiServer(svc service.XtzService) *ApiServer {
	return &ApiServer{
		svc: svc,
	}
}

func (s *ApiServer) Start(port string) {
	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware(middleware.Logger))
	router.HandleFunc("/xtz/delegations", s.handleGetDelegations).Methods("GET")

	logger := middleware.Logger

	logger.Info("Server started 🚀🚀🚀", "port", port)

	if err := http.ListenAndServe(port, router); err != nil {
		panic(err)
	}
}

func (s *ApiServer) handleGetDelegations(w http.ResponseWriter, r *http.Request) {
	logger := r.Context().Value(middleware.LoggerKey).(*slog.Logger)

	yearParam := r.URL.Query().Get("year")

	year, err := func() (int, error) {
		if yearParam == "" {
			return time.Now().Year(), nil
		}
		parsedYear, parseErr := strconv.Atoi(yearParam)
		return verifyYear(parsedYear, parseErr)
	}()

	if err != nil {
		logger.Error("Invalid year parameter", "error", err)
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid year parameter"})
		return
	}

	entry, err := s.svc.GetDelegations(context.Background(), year)
	if err != nil {
		logger.Error("Error fetching delegations", "error", err)
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

func writeJSON(w http.ResponseWriter, s int, v any) error {
	w.WriteHeader(s)
	w.Header().Add("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(v)
}

type InvalidYearError struct {
	Year int
}

func (e *InvalidYearError) Error() string {
	return "Invalid year: " + strconv.Itoa(e.Year)
}

func verifyYear(year int, err error) (int, error) {

	if err != nil {
		return 0, err
	}

	if year < 2018 || year > time.Now().Year() {
		return 0, &InvalidYearError{Year: year}
	}

	return year, nil
}
