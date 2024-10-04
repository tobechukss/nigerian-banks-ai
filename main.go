package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	setUpRoutes()
	port := os.Getenv("PORT")
	host := os.Getenv("HOST")
	portString := fmt.Sprint(host, ":", port)
	fmt.Println("Listening on Port", portString)
	err := http.ListenAndServe(portString, nil)
	if err != nil {
		log.Fatal("Server level error", err)
	}
}
