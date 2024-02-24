package web

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/avstrong/booking/internal/booking"
	"github.com/avstrong/booking/internal/boost"
	"github.com/avstrong/booking/internal/logger"
)

type Server struct {
	srv      *http.Server
	router   *http.ServeMux
	l        *logger.Logger
	conf     Conf
	bManager *booking.Manager
	boost    *boost.Manager
}

type Conf struct {
	L                 *logger.Logger
	ServerLogger      *log.Logger
	Host              string
	Port              string
	ReadHeaderTimeout time.Duration
	LivenessEndpoint  string
}

func New(ctx context.Context, conf Conf, bookingManager *booking.Manager, boost *boost.Manager) (*Server, error) {
	mux := http.NewServeMux()

	//nolint:exhaustruct
	srv := &http.Server{
		Addr:              net.JoinHostPort(conf.Host, conf.Port),
		ReadHeaderTimeout: conf.ReadHeaderTimeout * time.Second, //nolint:durationcheck
		ErrorLog:          conf.ServerLogger,
		Handler:           mux,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	server := &Server{
		srv:      srv,
		router:   mux,
		l:        conf.L,
		conf:     conf,
		bManager: bookingManager,
		boost:    boost,
	}

	server.addRoutes(mux)

	return server, nil
}

func (s *Server) Srv() *http.Server {
	return s.srv
}
