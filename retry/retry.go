package main

import (
	"errors"
	"log"
	"time"
)

type exponentialRetry struct {
	numOfRetry        int
	exponentialFactor int
	delay             time.Duration
	fallback          func()
}

func (r *exponentialRetry) execute(action func() (error)) error {

	for i := range r.numOfRetry {
		fixed := i + 1
		if err := action(); err != nil {
			log.Println("fail at attempt:", fixed)
			if fixed == r.numOfRetry {
				if r.fallback != nil {
					r.fallback()
				}
				return err
			}
			<-time.Tick((r.delay * time.Duration(fixed)))
		} else {
			log.Println("success at attempt:", fixed)
			return nil
		}
	}
	return nil
}

func newExponentialRetry(
	numOfRetry int,
	exponentialFactor int,
	delay time.Duration,
	fallback func(),
) *exponentialRetry {

	return &exponentialRetry{
		numOfRetry: numOfRetry,
		exponentialFactor: exponentialFactor,
		delay: delay,
		fallback: fallback,
	}
}

func main() {
	r := newExponentialRetry(5, 2, time.Second, nil)

	f := func() (string, error) {
		return "", errors.New("erorr")
	}

	errorWrapper := func() error {
		_, err := f()
		return err
	}

	// simulate retry success.
	go func() {
		<- time.Tick(time.Second * 3)
		f = func() (string, error) {
			return "", nil
		}
	}()

	err := r.execute(errorWrapper)

	if err != nil {
		log.Fatal(err)	
	}
}