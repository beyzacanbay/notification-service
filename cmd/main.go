package main

import (
	"log"

	"github.com/beyzacanbay/notification-service/internal/server"
)

func main() {
	app := server.NewServer()

	log.Fatal(app.Listen(":8081"))
}
