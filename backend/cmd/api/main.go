package main

import (
 "context"
 "errors"
 "log"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"

 "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
 ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
 defer stop()

 databaseURL := os.Getenv("DATABASE_URL")
 if databaseURL == "" {
  log.Fatal("DATABASE_URL is required")
 }
 pool, err := pgxpool.New(ctx, databaseURL)
 if err != nil {
  log.Fatalf("configure database: %v", err)
 }
 defer pool.Close()

 connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
 defer cancel()
 if err := pool.Ping(connectCtx); err != nil {
  log.Fatalf("connect to database: %v", err)
 }
 if err := runMigrations(ctx, pool); err != nil {
  log.Fatalf("run migrations: %v", err)
 }

 addr := os.Getenv("HTTP_ADDR")
 if addr == "" {
  port := os.Getenv("PORT")
  if port == "" {
   port = "8080"
  }
  addr = ":" + port
 }
 api := &server{db: pool, webOrigin: os.Getenv("WEB_ORIGIN"), hub: newRealtimeHub()}
 httpServer := &http.Server{
  Addr: addr, Handler: api.routes(), ReadHeaderTimeout: 5 * time.Second,
  ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
 }
 go func() {
  log.Printf("API listening on %s", addr)
  if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
   log.Printf("HTTP server failed: %v", err)
   stop()
  }
 }()
 <-ctx.Done()
 shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
 defer shutdownCancel()
 if err := httpServer.Shutdown(shutdownCtx); err != nil {
  log.Printf("HTTP shutdown: %v", err)
 }
}
