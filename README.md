# Booking System - Alpha Version

Welcome to the alpha version of our Booking System. This system allows users to book rooms in hotels, offering a simple yet comprehensive way to manage bookings, availability. This document provides instructions on how to get started, utilize the API, and conduct tests to ensure everything is functioning as expected.

## Features

- **Book Rooms**: Users can book available rooms in hotels for specific dates.
- **Boost Strategies**: Apply special boost strategies for discounts or promotions during booking.
- **Idempotency**: Ensures that bookings are processed only once to avoid duplicate bookings.

## Getting Started

### Prerequisites

1. Ensure you have Go installed on your system to run or build the application.

2. Postman or cURL for testing the API.

### Running the Application

To run the application directly:

```sh
go run main.go
```

## Testing the API

You can test the API functionalities using cURL commands or Postman. Below are examples of how to use cURL to interact with the API.

### Create a Booking

To create a booking, you need to include the Idempotency-Key header to ensure idempotency.

```sh
curl -X POST http://localhost:8092/api/orders/v1 \
     -H "Content-Type: application/json" \
     -H "Idempotency-Key: unique_key" \
     -d '{
         "places": [
             {
                "hotel_id": "reddison",
                "room_id": "lux",
                "from": "2024-02-26T00:00:00Z",
                "to": "2024-02-28T00:00:00Z"
             },
             {
                "hotel_id": "reddison",
                "room_id": "lux2",
                "from": "2024-03-28T00:00:00Z",
                "to": "2024-03-29T00:00:00Z"
            }
         ],
         "payer": {
             "email": "guest@mail.ru"
         }
     }'
```

## API Endpoints

- **POST /api/orders/v1**: Create a new booking.
  For testing using Postman, you can import the cURL commands as they are, or manually set up the requests in Postman with the same URLs, headers, and request bodies.

This README provides a starting point for your project documentation, ensuring that anyone getting started with your booking system has the necessary information to run, test, and understand the basic functionalities. As your project grows, consider expanding the documentation to cover new features and use cases.
