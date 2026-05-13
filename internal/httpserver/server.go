package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"stalwart-event-notification/internal/config"
	"stalwart-event-notification/internal/notifications"
	"stalwart-event-notification/internal/storage"
	"stalwart-event-notification/internal/webhook"
)

const maxWebhookBodyBytes = 2 << 20

type Notifier interface {
	ProcessEvent(ctx context.Context, event webhook.Event) error
}

type Server struct {
	config   config.Config
	store    storage.Store
	notifier Notifier
	metrics  *notifications.Metrics
	logger   *slog.Logger
	botOK    func(context.Context) bool
}

func New(
	cfg config.Config,
	store storage.Store,
	notifier Notifier,
	metrics *notifications.Metrics,
	logger *slog.Logger,
	botOK func(context.Context) bool,
) *Server {
	return &Server{
		config:   cfg,
		store:    store,
		notifier: notifier,
		metrics:  metrics,
		logger:   logger,
		botOK:    botOK,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/metrics", s.handleMetrics)
	return mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		_, _ = w.Write([]byte("OK"))
	case http.MethodPost:
		s.handleWebhook(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes))
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if s.config.WebhookKey != "" && !webhook.VerifySignature(body, s.config.WebhookKey, r.Header.Get("X-Signature")) {
		s.logger.Warn("webhook rejected invalid signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if !webhook.VerifyBasicAuth(r.Header.Get("Authorization"), s.config.WebhookUsername, s.config.WebhookPassword) {
		s.logger.Warn("webhook rejected invalid basic auth")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	payload, err := webhook.ParsePayload(body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	s.metrics.Inc("webhook_requests_total", 1)
	s.metrics.Inc("webhook_events_received", int64(len(payload.Events)))

	for _, event := range payload.Events {
		if event.ID == "" {
			event.ID = time.Now().UTC().Format("20060102150405.000000000")
		}
		if !webhook.IsKnownEventType(event.Type) {
			continue
		}
		if err := s.notifier.ProcessEvent(r.Context(), event); err != nil {
			s.logger.Error("process event failed", "error", err, "event_type", event.Type)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbOK := s.store.Ping(ctx) == nil
	botOK := true
	if s.botOK != nil {
		botOK = s.botOK(ctx)
	}
	status := http.StatusOK
	if !dbOK || !botOK {
		status = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       dbOK && botOK,
		"database": dbOK,
		"bot":      botOK,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.config.WebhookUsername != "" && !webhook.VerifyBasicAuth(r.Header.Get("Authorization"), s.config.WebhookUsername, s.config.WebhookPassword) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = w.Write([]byte(s.metrics.RenderPrometheus()))
}

func ListenAndServe(ctx context.Context, addr string, handler http.Handler, logger *slog.Logger) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		logger.Error("http server stopped", "error", err)
		return err
	}
}
