package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/atomizer"
	"github.com/benjivesterby/atomizer/conductors"
	_ "github.com/benjivesterby/montecarlopi"
	"github.com/pkg/errors"
)

const (
	// CONNECTIONSTRING is the connection string for the message queue, in this case
	// this is specific to rabbit mq
	CONNECTIONSTRING string = "CONNECTIONSTRING"

	// QUEUE is the queue for atom messages to be passed accross in the message queue
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
		q := flag.String("queue", "atomizer", "queue is the queue for atom messages to be passed accross in the message queue")
		flag.Parse()

		if *env {
			*c, *q, err = envoverride()
		}

		if err == nil {

			// Create a copy of the conductor for the agent
			var conductor atomizer.Conductor
			if conductor, err = conductors.Connect(*c, *q); err == nil {

				// Register the conductor into the atomizer library after initializing the
				/// connection to the message queue
				atomizer.Register(ctx, conductor.ID(), conductor)

				if conductor != nil {

					// Create a copy of the atomizer
					if mizer := atomizer.Atomize(ctx); mizer != nil {

						alog.Printc(ctx, stoichan(ctx, mizer.Events(0)))
						alog.Errorc(ctx, etoichan(ctx, mizer.Errors(0)))

						// Execute the processing on the atomizer
						if err = mizer.Exec(); err == nil {

							alog.Println("Online")

							// Block until the processing is interrupted
							mizer.Wait()

							alog.Println("Received Cleanup Complete")
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
		// TODO: Get the alog wait method to work with the internal channels
		alog.Wait()
	} else {
		alog.Fatalln(nil, "unable to overwrite the global logger")
	}
}

func stoichan(ctx context.Context, values <-chan string) <-chan interface{} {
	out := make(chan interface{})

	go func(ctx context.Context, values <-chan string, out chan<- interface{}) {
		for {
			select {
			case <-ctx.Done():
				close(out)
				return
			case out <- <-values:
			}
		}
	}(ctx, values, out)

	return out
}

func etoichan(ctx context.Context, values <-chan error) <-chan interface{} {
	out := make(chan interface{})

	go func(ctx context.Context, values <-chan error, out chan<- interface{}) {
		for {
			select {
			case <-ctx.Done():
				close(out)
				return
			case out <- <-values:
			}
		}
	}(ctx, values, out)

	return out
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
