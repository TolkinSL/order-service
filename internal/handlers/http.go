package handlers

import (
	"encoding/json"
	"net/http"
	"order-service/internal/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type HTTPHandler struct {
	cache      OrderCache
	repository OrderRepository
	log        *logrus.Logger
}

type OrderCache interface {
	Get(orderUID string) (*models.Order, bool)
	Set(orderUID string, order *models.Order)
	Size() int
}

type OrderRepository interface {
	GetOrder(orderUID string) (*models.Order, error)
}

func NewHTTPHandler(cache OrderCache, repo OrderRepository, logger *logrus.Logger) *HTTPHandler {
	return &HTTPHandler{
		cache:      cache,
		repository: repo,
		log:        logger,
	}
}

func (h *HTTPHandler) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/order/{order_uid}", h.GetOrder).Methods("GET")
	api.HandleFunc("/health", h.HealthCheck).Methods("GET")
	api.HandleFunc("/cache/stats", h.CacheStats).Methods("GET")

	router.HandleFunc("/", h.ServeIndex).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))

	router.Use(h.loggingMiddleware)
	router.Use(h.corsMiddleware)

	return router
}

func (h *HTTPHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderUID := vars["order_uid"]

	if orderUID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "order_uid is required")
		return
	}

	h.log.Infof("Fetching order: %s", orderUID)

	if order, found := h.cache.Get(orderUID); found {
		h.log.Debugf("Order %s found in cache", orderUID)
		h.writeJSONResponse(w, http.StatusOK, order)
		return
	}

	h.log.Debugf("Order %s not in cache, fetching from database", orderUID)
	order, err := h.repository.GetOrder(orderUID)
	if err != nil {
		if err == models.ErrOrderNotFound {
			h.writeErrorResponse(w, http.StatusNotFound, "order not found")
			return
		}
		h.log.Errorf("Failed to get order from database: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.cache.Set(orderUID, order)
	h.log.Debugf("Order %s added to cache", orderUID)

	h.writeJSONResponse(w, http.StatusOK, order)
}

func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":     "ok",
		"cache_size": h.cache.Size(),
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPHandler) CacheStats(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"cache_size": h.cache.Size(),
	}
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *HTTPHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/index.html")
}

func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Errorf("Failed to encode JSON response: %v", err)
	}
}

func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]string{
		"error": message,
	}
	h.writeJSONResponse(w, statusCode, response)
}

func (h *HTTPHandler) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.log.Infof("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPHandler) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
