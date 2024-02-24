package booking

import "time"

type RoomAvailability struct {
	HotelID string    `json:"hotel_id"`
	RoomID  string    `json:"room_id"`
	Date    time.Time `json:"date"`
	Quota   int       `json:"quota"`
}

type GetAvailabilityInput struct {
	HotelID string
	RoomID  string
	From    time.Time
	To      time.Time
}

type Event struct {
	ID        int
	OrderID   int
	CreatedAt time.Time
}

type Place struct {
	HotelID string    `json:"hotel_id"`
	RoomID  string    `json:"room_id"`
	From    time.Time `json:"from"`
	To      time.Time `json:"to"`
	Price   float64
}

type Payer struct {
	Email string `json:"email"`
}

type BookInput struct {
	Payer           Payer   `json:"payer"`
	Places          []Place `json:"places"`
	BoostStrategies []BoostStrategy
}

type Order struct {
	ID        int       `json:"id"`
	Payer     Payer     `json:"payer"`
	Places    []Place   `json:"places"`
	CreatedAt time.Time `json:"created_at"`
	Price     float64   `json:"price"`
}
