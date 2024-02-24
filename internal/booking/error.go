package booking

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrIdempotencyKey = errors.New("idempotency key not found")
	ErrNextID         = errors.New("get next id from generator")
	ErrLogic          = errors.New("logic error")
	ErrRecordNotFound = errors.New("record not found")
)

type AvailabilityError struct {
	errors []string
}

func NewAvailabilityError() *AvailabilityError {
	//nolint:exhaustruct
	return &AvailabilityError{}
}

func IsAvailabilityError(err error) *AvailabilityError {
	if err == nil {
		return nil
	}

	var availabilityError *AvailabilityError

	if errors.As(err, &availabilityError) {
		return availabilityError
	}

	return nil
}

func (e *AvailabilityError) AddUnavailableRoom(hotelID, roomID string, dates []time.Time) {
	e.errors = append(e.errors, fmt.Sprintf("room '%v' is unavalable in hotel '%v' on following dates %+v", roomID, hotelID, dates))
}

func (e *AvailabilityError) Error() string {
	return fmt.Sprintf("%+v", e.errors)
}

func (e *AvailabilityError) Fields() []string {
	return e.errors
}

func (e *AvailabilityError) UnavailableRoomsCount() int {
	return len(e.errors)
}

type InputError struct {
	fields map[string][]string
}

func newInputError() *InputError {
	return &InputError{
		fields: make(map[string][]string),
	}
}

func IsInputError(err error) *InputError {
	if err == nil {
		return nil
	}

	var inputError *InputError

	if errors.As(err, &inputError) {
		return inputError
	}

	return nil
}

func (ie *InputError) fieldsCount() int {
	return len(ie.fields)
}

func (ie *InputError) addError(field, msg string) {
	ie.fields[field] = append(ie.fields[field], msg)
}

func (ie *InputError) Error() string {
	return fmt.Sprintf("%+v", ie.fields)
}

func (ie *InputError) Fields() map[string][]string {
	return ie.fields
}
