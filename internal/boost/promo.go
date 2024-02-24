package boost

import (
	"context"
	"fmt"
	"time"

	"github.com/avstrong/booking/internal/booking"
)

type storage interface {
	GetAvailablePromo(ctx context.Context, from time.Time) ([]string, error)
}

type Manager struct {
	storage storage
}

func New(storage storage) *Manager {
	return &Manager{storage: storage}
}

type PromoCode struct {
	Code               string
	DiscountPercentage float64
	ValidThrough       time.Time
}

func (p *PromoCode) Apply(order *booking.Order) error {
	if time.Now().UTC().After(p.ValidThrough) {
		return fmt.Errorf("promo code %s expired: %w", p.Code, ErrPromoCodeExpired)
	}

	for i := range order.Places {
		order.Places[i].Price -= order.Places[i].Price * p.DiscountPercentage / 100 //nolint:gomnd // for test reason
	}

	return nil
}

type LoyaltyDiscount struct {
	CustomerID     string
	DiscountAmount float64
	ValidThrough   time.Time
}

func (l *LoyaltyDiscount) Apply(order *booking.Order) error {
	// Проверить уровень лояльности клиента и применить скидку
	order.Price -= l.DiscountAmount

	return nil
}

func (m *Manager) Strategies(ctx context.Context) ([]booking.BoostStrategy, error) {
	now := time.Now().UTC()

	res, err := m.storage.GetAvailablePromo(ctx, time.Now())
	if err != nil {
		return nil, fmt.Errorf("get available promo from storage starting from %v: %w", now, err)
	}

	// Some logic

	_ = res

	promoCode := PromoCode{
		Code:               "blackFriday",
		DiscountPercentage: 35,                                        //nolint:gomnd // for test reason
		ValidThrough:       time.Now().UTC().Add(time.Hour * 24 * 30), //nolint:gomnd // for test reason
	}

	loyaltyDiscount := LoyaltyDiscount{
		CustomerID:     "test@test.com",
		DiscountAmount: 300,                                       //nolint:gomnd // for test reason
		ValidThrough:   time.Now().UTC().Add(time.Hour * 24 * 30), //nolint:gomnd // for test reason
	}

	return []booking.BoostStrategy{
		&promoCode,
		&loyaltyDiscount,
	}, nil
}
