package door

import (
	rpio "github.com/stianeikeland/go-rpio/v4"
	"log"
	"sync"
	"time"
)

type Control struct {
	pin      rpio.Pin
	mutex    sync.Mutex
	refCount int
}

func New() Control {
	pin := rpio.Pin(21)
	pin.Output()
	pin.Low()
	return Control{pin: pin}
}

func (c *Control) UnlockForDuration(duration time.Duration, authorizedBy string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Unlock(authorizedBy)
	c.refCount++

	go func() {
		time.Sleep(duration)

		c.mutex.Lock()
		defer c.mutex.Unlock()

		c.refCount--
		if c.refCount == 0 {
			c.Lock(authorizedBy)
		}
	}()
}

func (c *Control) Unlock(authorizedBy string) {
	c.pin.High()
	log.Printf("door unlocked (%s)\n", authorizedBy)
}

func (c *Control) Lock(authorizedBy string) {
	c.pin.Low()
	log.Printf("door locked (%s)\n", authorizedBy)
}
