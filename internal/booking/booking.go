package booking

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/avstrong/booking/internal/logger"
)

type idGenerator interface {
	GetID(ctx context.Context) (int, error)
}

type storageReader interface {
	GetAvailabilities(ctx context.Context, properties []GetAvailabilityInput) ([]*RoomAvailability, error)
	GetOrderByIdempotencyKey(ctx context.Context) (*Order, error)
}

type storageWriter interface {
	BeginTransaction(ctx context.Context, level string) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
	SaveRoomAvailabilities(ctx context.Context, availabilities []*RoomAvailability) error
	SaveEvent(ctx context.Context, event *Event) error
	SaveOrder(ctx context.Context, order *Order) error
}

type storage interface {
	storageReader
	storageWriter
}

type BoostStrategy interface {
	Apply(order *Order) error
}

type Manager struct {
	l           *logger.Logger
	storage     storage
	idGenerator idGenerator
}

func New(l *logger.Logger, storage storage, idGenerator idGenerator) *Manager {
	return &Manager{
		l:           l,
		storage:     storage,
		idGenerator: idGenerator,
	}
}

func (b *BookInput) validate() error {
	inputErr := newInputError()

	if _, err := mail.ParseAddress(b.Payer.Email); err != nil {
		inputErr.addError("payer.email", "provide valid email")
	}

	if len(b.Places) == 0 {
		inputErr.addError("places", "provide at least one place")
	}

	for _, place := range b.Places {
		if place.HotelID == "" {
			inputErr.addError("place.hotelID", "provide place.hotelID")
		}

		if place.RoomID == "" {
			inputErr.addError("place.roomID", "provide place.roomID")
		}

		if place.From.Before(time.Now().UTC()) {
			inputErr.addError("place.from", "place.from must not be in the past")
		}

		if place.To.Before(time.Now().UTC()) {
			inputErr.addError("place.from", "place.to must not be in the past")
		}

		if place.From.After(place.To) {
			inputErr.addError("place.from", "place.from must be before place.to")
		}
	}

	if inputErr.fieldsCount() > 0 {
		return inputErr
	}

	return nil
}

func (b *BookInput) prepareDates() {
	for idx := range b.Places {
		b.Places[idx].From = b.Places[idx].From.Truncate(24 * time.Hour) //nolint:gomnd
		b.Places[idx].To = b.Places[idx].To.Truncate(24 * time.Hour)     //nolint:gomnd
	}
}

func (m *Manager) buildOrder(ctx context.Context, input *BookInput) (*Order, *Event, error) {
	id, err := m.idGenerator.GetID(ctx)
	if err != nil {
		return nil, nil, ErrNextID
	}

	//nolint:exhaustruct // price is only added for test
	order := &Order{
		ID: id,
		Payer: Payer{
			Email: input.Payer.Email,
		},
		Places:    input.Places,
		CreatedAt: time.Now().UTC(),
	}

	if input.BoostStrategies != nil {
		for _, strategy := range input.BoostStrategies {
			if err := strategy.Apply(order); err != nil {
				return nil, nil, fmt.Errorf("apply strategy to order: %w", err)
			}
		}
	}

	event, err := m.buildEvent(ctx, order.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("build event for order %v: %w", order.ID, err)
	}

	return order, event, nil
}

func (m *Manager) buildEvent(ctx context.Context, orderID int) (*Event, error) {
	id, err := m.idGenerator.GetID(ctx)
	if err != nil {
		return nil, ErrNextID
	}

	return &Event{
		ID:        id,
		OrderID:   orderID,
		CreatedAt: time.Now().UTC(),
	}, nil
}

func (m *Manager) getRoomAvailabilities(ctx context.Context, input *BookInput) ([]*RoomAvailability, error) {
	req := make([]GetAvailabilityInput, 0, len(input.Places))

	for _, property := range input.Places {
		req = append(req, GetAvailabilityInput{
			HotelID: property.HotelID,
			RoomID:  property.RoomID,
			From:    property.From,
			To:      property.To,
		})
	}

	availabilities, err := m.storage.GetAvailabilities(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get availabilities from storage: %w", err)
	}

	return availabilities, nil
}

func (m *Manager) updateRoomAvailabilities(
	input *BookInput,
	availabilities []*RoomAvailability,
) ([]*RoomAvailability, error) {
	var updatedRoomAvailabilities []*RoomAvailability

	availabilityMap := make(map[string]*RoomAvailability)

	for _, availability := range availabilities {
		key := fmt.Sprintf("%s_%s_%s", availability.HotelID, availability.RoomID, availability.Date.Format("2006-01-02"))
		availabilityMap[key] = availability
	}

	for _, property := range input.Places {
		currentDate := property.From
		for !currentDate.After(property.To) {
			key := fmt.Sprintf("%s_%s_%s", property.HotelID, property.RoomID, currentDate.Format("2006-01-02"))
			if availability, ok := availabilityMap[key]; ok {
				availability.Quota--

				updatedRoomAvailabilities = append(updatedRoomAvailabilities, availability)
				currentDate = currentDate.AddDate(0, 0, 1)

				continue
			}

			return nil, fmt.Errorf(
				"data are not the same. Check storage. Input %+v | RoomAvailabilities %+v: %w",
				input,
				availabilities,
				ErrLogic,
			)
		}
	}

	return updatedRoomAvailabilities, nil
}

//nolint:funlen,cyclop // it's linear simple code
func (m *Manager) CreateOrder(ctx context.Context, input *BookInput) (_ *Order, err error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	order, err := m.storage.GetOrderByIdempotencyKey(ctx)
	if err != nil && !errors.Is(err, ErrRecordNotFound) {
		return nil, fmt.Errorf("get order by idempotency key: %w", err)
	}

	if !errors.Is(err, ErrRecordNotFound) {
		return order, nil
	}

	input.prepareDates()

	availabilities, err := m.getRoomAvailabilities(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("get availabilities: %w", err)
	}

	availabilities, err = m.updateRoomAvailabilities(input, availabilities)
	if err != nil {
		return nil, fmt.Errorf("update availabilities: %w", err)
	}

	order, event, err := m.buildOrder(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("build order: %w", err)
	}

	ctx, err = m.storage.BeginTransaction(ctx, "READ COMMITTED")
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			if err = m.storage.RollbackTransaction(ctx); err != nil {
				m.l.LogErrorf("Could not rollback booking transaction after panic %v", p)
			}

			m.l.LogInfo("Transaction has been roll backed after panic")

			panic(p)
		}

		if err != nil {
			if err = m.storage.RollbackTransaction(ctx); err != nil {
				m.l.LogErrorf("Could not rollback booking transaction after error %v", err.Error())
			}

			m.l.LogInfo("Transaction has been roll backed after error")

			return
		}

		if err = m.storage.CommitTransaction(ctx); err != nil {
			m.l.LogErrorf("Could not commit booking transaction, err %v", err.Error())
		}

		m.l.LogInfo("Transaction has been committed")
	}()

	if err = m.storage.SaveOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("save order to storage: %w", err)
	}

	if err = m.storage.SaveRoomAvailabilities(ctx, availabilities); err != nil {
		return nil, fmt.Errorf("save room availabilities to storage: %w", err)
	}

	if err = m.storage.SaveEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("save event to storage: %w", err)
	}

	return order, nil
}
