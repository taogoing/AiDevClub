package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		// Resource ZIP uploads and downloads are valid up to 50 MiB. Leave
		// whole-request deadlines disabled so slow, valid transfers are not cut off.
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}
}

func ServeHTTP(ctx context.Context, server *http.Server) error {
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			_ = server.Close()
		}
		err := <-serveErr
		if shutdownErr != nil {
			return fmt.Errorf("shutdown http server: %w", shutdownErr)
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
