# Notification Service


curl -X POST http://localhost:8081/api/v1/notifications -H "Content-Type: application/json" -d '{"channel":"sms","recipient":"+905551234567","content":"Merhaba!","priority":0}'                             


1000 lik batch icinde gonderilemeyen notificationlar olursa napacagiz? bunu handle edelim

curl -X POST http://localhost:8081/api/v1/notifications/batch \
-H "Content-Type: application/json" \
-d '{
    "notifications": [
    {"channel":"sms","recipient":"+905551234567","content":"Hello 1"},
    {"channel":"email","recipient":"test@test.com","content":"Hello 2"},
    {"channel":"push","recipient":"device-token-123","content":"Hello 3","priority":0}
    ]
}'