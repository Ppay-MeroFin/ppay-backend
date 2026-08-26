package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mading-alier/ppay-backend/internal/config"
	"github.com/mading-alier/ppay-backend/internal/handlers"
	"github.com/mading-alier/ppay-backend/internal/mtnmomo"
	"github.com/mading-alier/ppay-backend/internal/store"
	"github.com/mading-alier/ppay-backend/internal/ussd"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	mtnClient := mtnmomo.NewCollectionClient(
		cfg.MTNCollectionBaseURL,
		cfg.MTNCollectionSubscriptionKey,
		cfg.MTNCollectionAPIUser,
		cfg.MTNCollectionAPIKey,
		cfg.MTNCollectionTargetEnv,
	)
	mtnClient.SetCallbackURL(cfg.MTNCollectionCallbackURL)

	token, err := mtnClient.GetAccessToken(ctx)
	if err != nil {
		log.Printf("mtn token error on startup (continuing, will retry on demand): %v", err)
	} else {
		prefix := token.AccessToken
		if len(prefix) > 16 {
			prefix = prefix[:16]
		}
		log.Printf("mtn collection token acquired, prefix=%s", prefix)
	}

	st, err := store.NewStore(ctx)
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	defer st.Close()

	h := handlers.NewHandler(st, mtnClient)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.HealthHandler)
	mux.HandleFunc("POST /v1/auth/pin", h.CreatePIN)
	mux.HandleFunc("POST /v1/auth/pin/verify", h.VerifyPIN)
	mux.HandleFunc("POST /tx/airtime", h.AirtimeHandler)
	mux.HandleFunc("POST /tx/data-bundle", h.DataBundleHandler)
	mux.HandleFunc("GET /tx/status/{ref}", h.TxStatusHandler)
	mux.HandleFunc("GET /tx/events/{ref}", h.TxEventsHandler)
	mux.HandleFunc("POST /tx/reconcile/{ref}", h.TxReconcileHandler)

	mux.HandleFunc("POST /callbacks/mtn/collection", h.MTNCollectionCallbackHandler)
	mux.HandleFunc("PUT /callbacks/mtn/collection", h.MTNCollectionCallbackHandler)

	ussdStore := ussd.NewInMemorySessionStore()
	ussdMenus := ussd.NewStaticMenuEngine()
	ussdService := ussd.NewService(ussdMenus)
	ussdEngine := ussd.NewSessionEngine(ussdStore, ussdMenus, ussdService)
	ussdHandler := ussd.NewHandler(ussdStore, ussdEngine)

	mux.HandleFunc("POST /ussd", ussdHandler.USSDHandler)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("ppay-backend listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
