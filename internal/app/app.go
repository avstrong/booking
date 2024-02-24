package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/avstrong/booking/internal/booking"
	"github.com/avstrong/booking/internal/idgen/simple"
	"github.com/avstrong/booking/internal/logger"
	"github.com/avstrong/booking/internal/migration"
	"github.com/avstrong/booking/internal/storage/memory"
	"github.com/avstrong/booking/internal/transport/web"
)

func Run(l *logger.Logger) error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer cancel()

	// Load config

	storage := memory.New(memory.Config{L: l})
	if err := migration.Up(ctx, l, storage); err != nil {
		return fmt.Errorf("up test migration: %w", err)
	}

	l.LogInfo("Test migration has been applied")

	idGen := simple.New()
	bookManager := booking.New(l, storage, idGen)

	webConf := web.Conf{
		L:                 l,
		ServerLogger:      log.Default(),
		Host:              "localhost",
		Port:              "8092",
		ReadHeaderTimeout: 20, //nolint:gomnd
		LivenessEndpoint:  "/liveness",
	}

	srv, err := web.New(ctx, webConf, bookManager, nil)
	if err != nil {
		return fmt.Errorf("init http server: %w", err)
	}

	//nolint:contextcheck
	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*4) //nolint:gomnd
		defer cancel()

		if err := srv.Srv().Shutdown(ctx); err != nil {
			l.LogErrorf("Failed to stop http server: %v", err.Error())
		}
	}()

	l.LogInfo("Application is running on %v:%v...", webConf.Host, webConf.Port)

	if err := srv.Srv().ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		l.LogErrorf("Failed to run http server: %v", err.Error())

		cancel()
	}

	l.LogInfo("Application stopped gracefully")

	return nil
}
