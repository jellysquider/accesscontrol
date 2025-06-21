package main

import (
	"log"
	"time"

	"github.com/doubleunion/accesscontrol/router"
	rpio "github.com/stianeikeland/go-rpio/v4"
)

func main() {
	err := rpio.Open()
	if err != nil {
		log.Fatalf("failed to initialize gpio: %+v", err)
	}
	defer rpio.Close()

	// Run updateIPAndRestart every minute in a separate thread
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			err := router.UpdateIPAndRestart()
			if err != nil {
				log.Printf("Error in updateIPAndRestart: %v", err)
			}
		}
	}()

	router.RunRouter()
}
