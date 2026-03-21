# Notification Service


curl -X POST http://localhost:8081/api/v1/notifications -H "Content-Type: application/json" -d '{"channel":"sms","recipient":"+905551234567","content":"Merhaba!","priority":0}'                             