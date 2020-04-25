package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devnw/alog"
	"github.com/devnw/atomizer"
	amqp "github.com/devnw/conductors/amqp"
	_ "github.com/devnw/montecarlopi"
	"github.com/pkg/errors"
)

const (
	// CONNECTIONSTRING is the connection string for the message queue, in this case
	// this is specific to rabbit mq
	CONNECTIONSTRING string = "CONNECTIONSTRING"

	// QUEUE is the queue for atom messages to be passed across in the message queue
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
			alog.Println("Interrupt Received, Closing Atomizer Agent")
		}
	}()

	var err error

	//[]alog.Destination{
	// 	{
	// 		alog.ERROR | alog.CRIT | alog.FATAL,
	// 		alog.STD,
	// 		os.Stderr,
	// 	},
	// }

	if err = alog.Global(
		ctx,
		"ATOMIZER AGENT",
		alog.DEFAULTTIMEFORMAT,
		time.UTC,
		alog.DEFAULTBUFFER,
		alog.Standards()...,
	); err == nil {
		env := flag.Bool("e", false, "signals to the agent to use environment variables for configurations")
		c := flag.String("conn", "amqp://guest:guest@localhost:5672/", "connection string used for rabbit mq")
		q := flag.String("queue", "atomizer", "queue is the queue for atom messages to be passed across in the message queue")
		flag.Parse()

		if *env {
			*c, *q, err = envoverride()
		}

		if err == nil {

			// Create a copy of the conductor for the agent
			var conductor atomizer.Conductor
			if conductor, err = amqp.Connect(ctx, *c, *q); err == nil {

				// Register the conductor into the atomizer library after initializing the
				/// connection to the message queue
				err := atomizer.Register(conductor)
				if err != nil {
					panic(err)
				}

				if conductor != nil {
					events := make(chan interface{})
					alog.Printc(ctx, events)
					// Create a copy of the atomizer
					if mizer := atomizer.Atomize(ctx, events); mizer != nil {

						// Execute the processing on the atomizer
						if err = mizer.Exec(); err == nil {

							alog.Println("Online")

							// Block until the processing is interrupted
							mizer.Wait()
						} else {
							alog.Fatalln(err, "error while executing atomizer")
						}
					} else {
						alog.Fatalln(err, "atomizer instance returned nil")
					}
				} else {
					alog.Fatalln(err, "conductor was returned nil")
				}
			} else {
				alog.Fatalln(err, "error while initializing conductor")
			}
		} else {
			alog.Fatalln(err, "error while pulling environment variables")
		}

		time.Sleep(time.Millisecond * 50)

		// Get the alog wait method to work with the internal channels
		alog.Wait(true)
	} else {
		alog.Fatalln(nil, "unable to overwrite the global logger")
	}
}

// envoverride pulls the environment variables as defined in the constants
// section and overwrites the passed flag values
func envoverride() (c, q string, err error) {

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
