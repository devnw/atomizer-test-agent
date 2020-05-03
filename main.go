// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devnw/alog"
	"github.com/devnw/amqp"
	"github.com/devnw/atomizer"
	_ "github.com/devnw/montecarlopi"
	"github.com/pkg/errors"
)

const (
	// CONNECTIONSTRING is the connection string for the message queue,
	// in this case this is specific to rabbitmq
	CONNECTIONSTRING string = "CONNECTIONSTRING"

	// QUEUE is the queue for atom messages to be passed across in the
	// message queue
	QUEUE string = "QUEUE"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	// Setup interrupt monitoring for the agent
	go func() {
		defer cancel()

		select {
		case <-ctx.Done():
			return
		case <-sigs:
			alog.Println("Closing Atomizer Agent")
			os.Exit(1)
		}
	}()

	err := alog.Global(
		ctx,
		"ATOMIZER AGENT",
		alog.DEFAULTTIMEFORMAT,
		time.UTC,
		alog.DEFAULTBUFFER,
		alog.Standards()...,
	)

	if err != nil {
		fmt.Println("unable to overwrite the global logger")
	}

	env := flag.Bool("e", false, "signals to the agent to use environment variables for configurations")
	c := flag.String("conn", "amqp://guest:guest@localhost:5672/", "connection string used for rabbit mq")
	q := flag.String("queue", "atomizer", "queue is the queue for atom messages to be passed across in the message queue")
	flag.Parse()

	if *env {
		*c, *q, err = environment()
	}

	if err != nil {
		fmt.Println("error while pulling environment variables | " + err.Error())
		os.Exit(1)
	}

	// Create the amqp conductor for the agent
	conductor, err := amqp.Connect(ctx, *c, *q)
	if err != nil || conductor == nil {
		fmt.Println("error while initializing amqp | " + err.Error())
		os.Exit(1)
	}

	// Register the conductor into the atomizer library after initializing the
	/// connection to the message queue
	err = atomizer.Register(conductor)
	if err != nil {
		fmt.Println("error registering amqp conductor | " + err.Error())
		os.Exit(1)
	}

	// setup the alog event subscription
	events := make(chan interface{})
	defer close(events)

	alog.Printc(ctx, events)

	// Create a copy of the atomizer
	a := atomizer.Atomize(ctx, events)
	if a == nil {
		fmt.Println("atomizer instance returned nil")
		os.Exit(1)
	}

	// Execute the processing on the atomizer
	err = a.Exec()
	if err != nil {
		fmt.Println("error while executing atomizer | " + err.Error())
		os.Exit(1)
	}

	alog.Println("Online")

	// Block until the processing is interrupted
	a.Wait()

	time.Sleep(time.Millisecond * 50)

	// Get the alog wait method to work with the internal channels
	alog.Wait(true)
}

// environment pulls the environment variables as defined in the constants
// section and overwrites the passed flag values
func environment() (c, q string, err error) {

	if c = os.Getenv(CONNECTIONSTRING); len(c) > 0 {
		if q = os.Getenv(QUEUE); len(q) > 0 {
		} else {
			err = errors.Errorf("environment variable %s is empty", QUEUE)
		}
	} else {
		err = errors.Errorf("environment variable %s is empty", CONNECTIONSTRING)
	}

	return c, q, err
}
