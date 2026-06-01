package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mading-alier/ppay-backend/internal/handlers"
	"github.com/mading-alier/ppay-backend/internal/store"
)

func main() {
	ctx := context.Background()

	st, err := store.NewStore(ctx)
	if err != nil {
		log.Fatalf("create store: %v", err)
	}
	defer st.Close()

	h := handlers.NewHandler(st)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.HealthHandler)
	mux.HandleFunc("/tx/airtime", h.AirtimeHandler)
	mux.HandleFunc("/tx/data-bundle", h.DataBundleHandler)
	mux.HandleFunc("/tx/status", h.TxStatusHandler)
	mux.HandleFunc("/tx/events", h.TxEventsHandler)
	mux.HandleFunc("/tx/reconcile", h.TxReconcileHandler)

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
