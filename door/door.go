package door

import (
	"fmt"
	rpio "github.com/stianeikeland/go-rpio/v4"
	"log"
	"sync"
	"time"
)

type Door struct {
	pin      rpio.Pin
	mutex    sync.Mutex
	refCount int
}

func New() Door {
	pin := rpio.Pin(21)
	pin.Output()
	pin.Low()
	return Door{pin: pin}
}

const maxDuration = time.Second * 30

func (c *Door) UnlockForDuration(duration time.Duration, authorizedBy string) error {
	if duration > maxDuration {
		return fmt.Errorf("duration (%.0f) is longer than maximum allowed (%.0f)", duration.Seconds(), maxDuration.Seconds())
	}

	if duration <= 0 {
		return fmt.Errorf("duration (%.0f) must be greater than 0", duration.Seconds())
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.unlock(authorizedBy)
	c.refCount++

	go func() {
		time.Sleep(duration)

		c.mutex.Lock()
		defer c.mutex.Unlock()

		c.refCount--
		if c.refCount == 0 {
			c.lock(authorizedBy)
		}
	}()

	return nil
}

func (c *Door) unlock(authorizedBy string) {
	c.pin.High()
	log.Printf("door unlocked (%s)\n", authorizedBy)
}

func (c *Door) lock(authorizedBy string) {
	c.pin.Low()
	log.Printf("door locked (%s)\n", authorizedBy)
}
