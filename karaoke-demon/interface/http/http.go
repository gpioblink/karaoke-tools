package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/interface/handler"
)

type HttpInterface struct {
	router       map[string]handler.HandlerFunc
	musicService application.MusicService
	server       *http.Server
}

type WebhookRequest struct {
	RequestNo string `json:"request_no"`
}

var DefaultRouter = map[string]handler.HandlerFunc{
	"reserve": handler.ReserveSongWebhook,
}

func NewHttpInterface(service *application.MusicService, router map[string]handler.HandlerFunc) *HttpInterface {
	mux := http.NewServeMux()

	httpInterface := &HttpInterface{
		router:       router,
		musicService: *service,
		server: &http.Server{
			Addr:    ":8787",
			Handler: corsMiddleware(mux),
		},
	}

	// webhook endpoint
	mux.HandleFunc("/webhook/reserve", httpInterface.handleReserveWebhook)

	return httpInterface
}

// corsMiddleware adds CORS headers to allow requests from any origin
func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func (h *HttpInterface) handleReserveWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode webhook request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.RequestNo == "" {
		http.Error(w, "request_no is required", http.StatusBadRequest)
		return
	}

	// Call the handler
	ctx := context.Background()
	handlerReq := handler.NewRequest("reserve", []string{req.RequestNo})

	if handlerFunc, ok := h.router["reserve"]; ok {
		handlerFunc(ctx, h.musicService, *handlerReq)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Song %s reserved successfully", req.RequestNo),
		})
	} else {
		log.Printf("Handler not found for reserve action")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *HttpInterface) Run() error {
	log.Printf("Starting HTTP server on %s", h.server.Addr)
	return h.server.ListenAndServe()
}

// TODO: 終了時のシグナルハンドリングを実装
// func (h *HttpInterface) Stop() error {
// 	log.Println("Stopping HTTP server...")
// 	return h.server.Shutdown(context.Background())
// }
