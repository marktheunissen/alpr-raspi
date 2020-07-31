package main

import (
	"fmt"
	"listen_event"
	"log"
	"runtime"
	"strings"
	"time"

	"config"

	"github.com/jpillora/backoff"
	"github.com/kr/beanstalk"
	"github.com/rjeczalik/notify"
)

func main() {
	fmt.Println("ARCH:", runtime.GOOS, "Event type:", listen_event.ListenEvent)

	// Setup a filesystem watch on the directory.
	directory := config.Opts.WatchDir
	log.Println("Watching dir:", directory)
	fsEvents := make(chan notify.EventInfo, 100000)
	if err := notify.Watch(directory, fsEvents, listen_event.ListenEvent); err != nil {
		log.Fatalln(err)
	}
	defer notify.Stop(fsEvents)

	// Beanstalkd parameters
	motionEventsTubeName := "motion_events"
	addr := "127.0.0.1:11300"
	bs_backoff := &backoff.Backoff{
		//These are the defaults
		Min:    100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	// Main loop. Used to reconnect in the event of disconnection.
	for {
		log.Println("Connecting to beanstalkd...")
		conn, err := beanstalk.Dial("tcp", addr)
		if err != nil {
			backoffTime := bs_backoff.Duration()
			log.Printf("[ERROR]: sleeping %s: %s", backoffTime, err)
			time.Sleep(backoffTime)
			continue
		}
		defer conn.Close()
		bs_backoff.Reset()
		motionEventsTube := &beanstalk.Tube{conn, motionEventsTubeName}
		log.Println("Connected, beanstalkd addr:", addr)
		log.Println("Motion events tube:", motionEventsTubeName)

	ReceiveLoop:
		for {
			event := <-fsEvents
			filePath := event.Path()
			if strings.HasSuffix(filePath, "jpg") && !strings.HasSuffix(filePath, "lastsnap.jpg") {
				// Eventbytes, priority, delay, time-to-run
				id, err := motionEventsTube.Put([]byte(filePath), 1, 0, 30*time.Second)
				if err != nil {
					log.Println("[ERROR]: Beanstalk:", err)
					break ReceiveLoop
				}

				log.Println("JobID:", id, "filePath:", filePath)
			}
		}
	}
}
