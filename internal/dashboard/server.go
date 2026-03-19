package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/store"
)

const wsClientQueueSize = 64

// StartServer starts the dashboard HTTP + WebSocket server.
func StartServer(ctx context.Context, addr string, cfg *store.GlobalConfig) error {
	hub := newWSHub()
	go hub.run(ctx)

	events.Bus.On("*", func(e events.EventData) {
		hub.broadcast(e)
	})

	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":    "ok",
			"uptime":    time.Since(startTime).String(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			log.Printf("failed to encode response: %v", err)
		}
	})

	r.With(apiKeyAuth(cfg.AdminAPIKey)).Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.handleWS(w, r)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(apiKeyAuth(cfg.AdminAPIKey))
		r.Get("/users", handleListUsers)
		r.Get("/stats", handleStats)
		r.Get("/config", handleGetConfig)
		r.Post("/config", handleSetConfig)
		r.Get("/logs", handleGetLogs)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/dashboard.html")
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	srv := &http.Server{Addr: addr, Handler: r}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		hub.closeAll(websocket.StatusGoingAway, "server shutting down")
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("failed to shutdown server: %v", err)
		}
	}()

	return srv.ListenAndServe()
}

var startTime = time.Now()

func apiKeyAuth(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				http.Error(w, `{"error":"admin api key not configured"}`, http.StatusServiceUnavailable)
				return
			}
			key := r.Header.Get("X-API-Key")
			if key != apiKey {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := store.ListAllUsers()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := store.GetStats()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusInternalServerError)
		return
	}
	costStats := store.GetAllCostStats()
	result := map[string]any{"logs": stats, "costs": costStats}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := store.GetAllConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cfg); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func handleSetConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	for k, v := range body {
		if err := store.SetConfig(k, v); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	logs, err := store.GetLogs(date, 100)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}

type wsClient struct {
	conn      *websocket.Conn
	ctx       context.Context
	outbound  chan events.EventData
	closeOnce sync.Once
}

type wsHub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
	msgCh   chan events.EventData
}

func newWSHub() *wsHub {
	return &wsHub{
		clients: make(map[*wsClient]bool),
		msgCh:   make(chan events.EventData, 256),
	}
}

func (h *wsHub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.closeAll(websocket.StatusGoingAway, "server stopped")
			return
		case msg := <-h.msgCh:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.outbound <- msg:
				default:
					go h.removeClient(client, websocket.StatusPolicyViolation, "client queue full")
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *wsHub) broadcast(event events.EventData) {
	select {
	case h.msgCh <- event:
	default:
	}
}

func (h *wsHub) addClient(client *wsClient) {
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	go h.writeLoop(client)
}

func (h *wsHub) writeLoop(client *wsClient) {
	for {
		select {
		case <-client.ctx.Done():
			h.removeClient(client, websocket.StatusNormalClosure, "client closed")
			return
		case msg, ok := <-client.outbound:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := wsjson.Write(writeCtx, client.conn, msg)
			cancel()
			if err != nil {
				h.removeClient(client, websocket.StatusNormalClosure, "write failed")
				return
			}
		}
	}
}

func (h *wsHub) removeClient(client *wsClient, status websocket.StatusCode, reason string) {
	h.mu.Lock()
	_, ok := h.clients[client]
	if ok {
		delete(h.clients, client)
	}
	h.mu.Unlock()
	if !ok {
		return
	}
	client.close(status, reason)
}

func (h *wsHub) closeAll(status websocket.StatusCode, reason string) {
	h.mu.Lock()
	clients := make([]*wsClient, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
		delete(h.clients, client)
	}
	h.mu.Unlock()
	for _, client := range clients {
		client.close(status, reason)
	}
}

func (c *wsClient) close(status websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		close(c.outbound)
		_ = c.conn.Close(status, reason)
	})
}

func (h *wsHub) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("WebSocket accept error: %v", err)
		return
	}

	client := &wsClient{
		conn:     conn,
		ctx:      r.Context(),
		outbound: make(chan events.EventData, wsClientQueueSize),
	}
	h.addClient(client)
	defer h.removeClient(client, websocket.StatusNormalClosure, "client disconnected")

	for {
		_, _, err := conn.Read(r.Context())
		if err != nil {
			return
		}
	}
}
