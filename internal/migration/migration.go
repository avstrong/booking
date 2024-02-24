package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/avstrong/booking/internal/booking"
	"github.com/avstrong/booking/internal/logger"
)

type storage interface {
	BeginTransaction(ctx context.Context, level string) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
	SaveRoomAvailabilities(ctx context.Context, availabilities []*booking.RoomAvailability) error
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func Up(ctx context.Context, l *logger.Logger, storage storage) (err error) {
	roomAvailabilities := []*booking.RoomAvailability{
		{
			HotelID: "reddison",
			RoomID:  "lux",
			Date:    date(2024, 2, 26),
			Quota:   2,
		},
		{
			HotelID: "reddison",
			RoomID:  "lux",
			Date:    date(2024, 2, 27),
			Quota:   4,
		},
		{
			HotelID: "reddison",
			RoomID:  "lux",
			Date:    date(2024, 2, 28),
			Quota:   1,
		},
	}

	ctx, err = storage.BeginTransaction(ctx, "")
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	ctx = booking.NewContextWithIdempotencyKey(ctx, "migration")

	defer func() {
		if p := recover(); p != nil {
			if err = storage.RollbackTransaction(ctx); err != nil {
				l.LogErrorf("Could not rollback migration transaction after panic %v", p)
			}

			l.LogInfo("Migration transaction has been roll backed after panic")

			panic(p)
		}

		if err != nil {
			if err = storage.RollbackTransaction(ctx); err != nil {
				l.LogErrorf("Could not rollback migration transaction after error %v", err.Error())
			}

			l.LogInfo("Migration transaction has been roll backed after error")

			return
		}

		if err = storage.CommitTransaction(ctx); err != nil {
			l.LogErrorf("Could not commit migration transaction, err %v", err.Error())
		}

		l.LogInfo("Migration transaction has been committed")
	}()

	if err = storage.SaveRoomAvailabilities(ctx, roomAvailabilities); err != nil {
		return fmt.Errorf("save room availabilities to storage: %w", err)
	}

	return nil
}
