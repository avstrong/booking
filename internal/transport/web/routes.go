package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/avstrong/booking/internal/booking"
)

func (s *Server) checkRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) (*booking.BookInput, string) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, "Idempotency-Key header is missing", http.StatusBadRequest)

		return nil, ""
	}

	var input booking.BookInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)

		return nil, ""
	}

	if s.boost != nil {
		strategies, err := s.boost.Strategies(ctx)
		if err != nil {
			s.l.LogErrorf("Could not get boost strategies: %v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		if len(strategies) != 0 {
			input.BoostStrategies = strategies
		}
	}

	return &input, idempotencyKey
}

func (s *Server) createOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	input, idempotencyKey := s.checkRequest(ctx, w, r)
	if idempotencyKey == "" {
		return
	}

	ctx = booking.NewContextWithIdempotencyKey(ctx, idempotencyKey)

	out, err := s.bManager.CreateOrder(ctx, input)
	if inputErr := booking.IsInputError(err); inputErr != nil {
		w.WriteHeader(http.StatusBadRequest)

		if err = json.NewEncoder(w).Encode(inputErr.Fields()); err != nil {
			s.l.LogErrorf("Could not encode validation err: %v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	if availabilityErr := booking.IsAvailabilityError(err); availabilityErr != nil {
		w.WriteHeader(http.StatusPreconditionFailed)

		if err = json.NewEncoder(w).Encode(availabilityErr.Fields()); err != nil {
			s.l.LogErrorf("Could not encode availability err: %v", err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	if err != nil {
		s.l.LogErrorf("Could not create an order: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)

	if err = json.NewEncoder(w).Encode(out); err != nil {
		s.l.LogErrorf("Could not encode result of order creating: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) livenessHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addRoutes(r *http.ServeMux) {
	r.Handle(
		"POST /api/orders/v1",
		s.applyMiddlewares(http.HandlerFunc(s.createOrderHandler), s.loggerMiddleware(), s.recoverMiddleware()),
	)
	r.Handle(
		fmt.Sprintf("GET %s", s.conf.LivenessEndpoint),
		s.applyMiddlewares(http.HandlerFunc(s.livenessHandler), s.loggerMiddleware(), s.recoverMiddleware()),
	)
}
