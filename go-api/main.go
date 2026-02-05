package main

import (
	"log"

	httpapi "github.com/andrewreder/agent-poc/go-api/http-api"
)

func main() {
	r, err := httpapi.NewRouter()
	if err != nil {
		log.Fatalf("failed to initialize HTTP router: %v", err)
	}
	r.Run(":8080")
}
