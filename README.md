```sh
go run main.go
```

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
             }
         ],
         "payer": {
             "email": "guest@mail.ru"
         }
     }'
```
