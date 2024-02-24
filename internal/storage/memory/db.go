package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avstrong/booking/internal/booking"
	"github.com/avstrong/booking/internal/logger"
)

type Config struct {
	L *logger.Logger
}

type transaction struct {
	id                 string
	roomModifications  map[string]*booking.RoomAvailability
	orderModifications map[int]*booking.Order
	eventModifications map[int]*booking.Event
	rollbackActions    []func()
}

type DB struct {
	mu                   sync.Mutex
	l                    *logger.Logger
	roomAvailabilities   map[string]*booking.RoomAvailability
	events               map[int]*booking.Event
	orders               map[int]*booking.Order
	transactions         map[string]*transaction
	nextTrxID            int64
	orderIdempotencyKeys map[string]*booking.Order
}

func New(conf Config) *DB {
	//nolint:exhaustruct
	return &DB{
		l:                    conf.L,
		roomAvailabilities:   make(map[string]*booking.RoomAvailability),
		events:               make(map[int]*booking.Event),
		orders:               make(map[int]*booking.Order),
		transactions:         make(map[string]*transaction),
		orderIdempotencyKeys: make(map[string]*booking.Order),
	}
}

func (db *DB) BeginTransaction(ctx context.Context, _ string) (context.Context, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID := fmt.Sprintf("trx-%d", db.nextTrxID)
	db.nextTrxID++

	db.transactions[trxID] = &transaction{
		id:                 trxID,
		roomModifications:  make(map[string]*booking.RoomAvailability),
		orderModifications: make(map[int]*booking.Order),
		eventModifications: make(map[int]*booking.Event),
		rollbackActions:    []func(){},
	}

	return withTransactionID(ctx, trxID), nil
}

func (db *DB) CommitTransaction(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID, ok := transactionIDFromContext(ctx)
	if !ok || trxID == "" {
		return ErrTransactionIDNotFoundInCtx
	}

	trx, exists := db.transactions[trxID]
	if !exists {
		return fmt.Errorf("transaction %s not found: %w", trxID, ErrTransactionNotFound)
	}

	idempotencyKey, ok := booking.IdempotencyKeyFromContext(ctx)
	if !ok || idempotencyKey == "" {
		return booking.ErrIdempotencyKey
	}

	for key, room := range trx.roomModifications {
		db.roomAvailabilities[key] = room
	}

	for _, order := range trx.orderModifications {
		db.orders[order.ID] = order
		db.orderIdempotencyKeys[idempotencyKey] = order
	}

	for _, event := range trx.eventModifications {
		db.events[event.ID] = event
	}

	delete(db.transactions, trxID)

	return nil
}

func (db *DB) RollbackTransaction(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID, ok := transactionIDFromContext(ctx)
	if !ok || trxID == "" {
		return ErrTransactionIDNotFoundInCtx
	}

	trx, exists := db.transactions[trxID]
	if !exists {
		return fmt.Errorf("transaction %s not found: %w", trxID, ErrTransactionNotFound)
	}

	for _, action := range trx.rollbackActions {
		action()
	}

	delete(db.transactions, trxID)

	return nil
}

func (db *DB) SaveRoomAvailabilities(ctx context.Context, availabilities []*booking.RoomAvailability) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID, ok := transactionIDFromContext(ctx)
	if !ok || trxID == "" {
		return ErrTransactionIDNotFoundInCtx
	}

	trx, exists := db.transactions[trxID]
	if !exists {
		return fmt.Errorf("transaction %s not found: %w", trxID, ErrTransactionNotFound)
	}

	for _, availability := range availabilities {
		key := fmt.Sprintf("%s_%s_%s", availability.HotelID, availability.RoomID, availability.Date.Format(time.RFC3339))
		if _, ok := trx.roomModifications[key]; ok {
			continue
		}

		trx.roomModifications[key] = availability

		originalRoom, exists := db.roomAvailabilities[key]
		if exists {
			trx.rollbackActions = append(trx.rollbackActions, func() {
				db.roomAvailabilities[key] = originalRoom
			})

			continue
		}

		trx.rollbackActions = append(trx.rollbackActions, func() {
			delete(db.roomAvailabilities, key)
		})
	}

	return nil
}

func (db *DB) SaveOrder(ctx context.Context, order *booking.Order) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID, ok := transactionIDFromContext(ctx)
	if !ok || trxID == "" {
		return ErrTransactionIDNotFoundInCtx
	}

	trx, exists := db.transactions[trxID]
	if !exists {
		return fmt.Errorf("transaction %s not found: %w", trxID, ErrTransactionNotFound)
	}

	if _, ok = trx.orderModifications[order.ID]; ok {
		return nil
	}

	trx.orderModifications[order.ID] = order
	trx.rollbackActions = append(trx.rollbackActions, func() {
		delete(db.orders, order.ID)
	})

	return nil
}

func (db *DB) SaveEvent(ctx context.Context, event *booking.Event) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	trxID, ok := transactionIDFromContext(ctx)
	if !ok || trxID == "" {
		return ErrTransactionIDNotFoundInCtx
	}

	trx, exists := db.transactions[trxID]
	if !exists {
		return fmt.Errorf("transaction %s not found: %w", trxID, ErrTransactionNotFound)
	}

	if _, ok = trx.eventModifications[event.ID]; ok {
		return nil
	}

	trx.eventModifications[event.ID] = event
	trx.rollbackActions = append(trx.rollbackActions, func() {
		delete(db.events, event.ID)
	})

	return nil
}

func (db *DB) GetAvailabilities(_ context.Context, inputs []booking.GetAvailabilityInput) ([]*booking.RoomAvailability, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var unavailableDates []time.Time

	availabilityErr := booking.NewAvailabilityError()

	var result []*booking.RoomAvailability

	for _, input := range inputs {
		for d := input.From; !d.After(input.To); d = d.AddDate(0, 0, 1) {
			key := fmt.Sprintf("%s_%s_%s", input.HotelID, input.RoomID, d.Format(time.RFC3339))

			roomAvailability, ok := db.roomAvailabilities[key]
			if !ok || roomAvailability.Quota < 1 {
				unavailableDates = append(unavailableDates, d)

				continue
			}

			result = append(result, roomAvailability)
		}

		if len(unavailableDates) > 0 {
			availabilityErr.AddUnavailableRoom(input.HotelID, input.RoomID, unavailableDates)
			unavailableDates = nil
		}
	}

	if availabilityErr.UnavailableRoomsCount() > 0 {
		return nil, availabilityErr
	}

	return result, nil
}

func (db *DB) GetOrderByIdempotencyKey(ctx context.Context) (*booking.Order, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	key, ok := booking.IdempotencyKeyFromContext(ctx)
	if !ok || key == "" {
		return nil, booking.ErrIdempotencyKey
	}

	order, exists := db.orderIdempotencyKeys[key]
	if exists {
		return order, nil
	}

	return nil, booking.ErrRecordNotFound
}
