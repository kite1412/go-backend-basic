package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

const (
	closed = iota
	open
	halfopen
)

type settings struct {
	maxCall  int
	timeout  time.Duration
	failures int
	fallback func()
}

type circuitBreaker struct {
	log        *log.Logger
	state     int
	callCount  int
	errCount   int
	settings   settings
	m          sync.Mutex
	transition chan time.Duration
}

func (c *circuitBreaker) stateTransition() {
	for d := range c.transition {
		<- time.NewTicker(d).C
		c.log.Println("entering half-open state")
		c.m.Lock()
		c.errCount = 0
		c.state = halfopen
		c.m.Unlock()
	}
}

func (c *circuitBreaker) execute(action func() (any, error)) {
	c.m.Lock()
	defer c.m.Unlock()
	if c.state == closed {
		_, err := action()
		if err != nil {
			c.errCount++
			if c.errCount == c.settings.failures {
				c.state = open
				c.transition <- c.settings.timeout
			}
		} else {
			c.errCount = 0
		}
	} else if c.state == halfopen {
		if c.callCount < c.settings.maxCall {
			c.callCount++
			_, err := action()
			if err != nil {
				c.log.Println("back to open after half-open state")
				c.callCount = 0
				c.state = open
				c.transition <- c.settings.timeout
				return
			}
		}
		if c.callCount == c.settings.maxCall {
			c.log.Println("state restored")
			c.state = closed
		}
	} else {
		c.log.Println("circuit breaker in open state, not accepting any request at the moment.")
		if c.settings.fallback != nil {
			c.log.Println("resort to fallback")
			c.settings.fallback()
		}
	}
}

func newCircuitBreaker(settings settings) *circuitBreaker {
	s := &settings

	if s.failures == 0 {
		s.failures = 1
	}
	
	if s.maxCall == 0 {
		s.maxCall = 1
	}
	 
	if s.timeout == 0 {
		s.timeout = time.Second * 4
	}
	
	cb := &circuitBreaker{
		log: log.New(log.Writer(), "<CB>", log.Lshortfile),
		state: closed,
		settings: settings,
		transition: make(chan time.Duration),
	}

	go cb.stateTransition()

	return cb
}

func main() {
	cb := newCircuitBreaker(settings{})

	// simulate restoration.
	for n := range 20 {
		if n % 5 == 0 {
			cb.execute(func() (any, error) {
				cb.log.Println("func called")
				return "", nil
			})
		} else {
			cb.execute(func() (any, error) {
				cb.log.Println("error called")
				return "", errors.New("error")
			})
		}
		time.Sleep(time.Second)
	}

	// simulate error on half-open.
	// for range 20 {
	// 	cb.execute(func() (any, error) {
	// 		cb.log.Println("error called")
	// 		return "", errors.New("error")
	// 	})
	// 	time.Sleep(time.Second)
	// }
}