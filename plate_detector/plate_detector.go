package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"config"

	"github.com/jpillora/backoff"
	"github.com/kr/beanstalk"
	"github.com/openalpr/openalpr"
)

func main() {
	log.Println("Plate detector startup")

	// Beanstalkd parameters
	motionTubeName := "motion_events"
	detectionTubeName := "detection_events"
	addr := "127.0.0.1:11300"
	reserveTimeout := time.Duration(5 * time.Second)
	bs_backoff := &backoff.Backoff{
		//These are the defaults
		Min:    100 * time.Millisecond,
		Max:    15 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	// ALPR parameters
	alpr := openalpr.NewAlpr(config.Opts.Region, "", "/usr/local/share/openalpr/runtime_data/")
	defer alpr.Unload()
	if !alpr.IsLoaded() {
		log.Println("OpenAlpr failed to load!")
		return
	}
	alpr.SetTopN(3)
	log.Println("ALPR loaded:", alpr.IsLoaded(), "region:", config.Opts.Region, "TopN: 3", "version:", openalpr.GetVersion())

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
		log.Println("Connected, beanstalkd addr:", addr, "reserve timeout:", reserveTimeout)
		log.Println("Motion events tube:", motionTubeName, "detection events tube:", detectionTubeName)
		defer conn.Close()
		bs_backoff.Reset()
		motionEventsTubeSet := beanstalk.NewTubeSet(conn, motionTubeName)
		detectionTube := beanstalk.Tube{conn, detectionTubeName}

	ReceiveLoop:
		for {
			// Returns an error after the timeout expires without receiving a job.
			id, filename, err := motionEventsTubeSet.Reserve(reserveTimeout)

			if err != nil {
				connErr, ok := err.(beanstalk.ConnError)
				// There was an error but it wasn't a timeout.
				if !ok || connErr.Err != beanstalk.ErrTimeout {
					log.Printf("[ERROR]: reserving a job: %+v", err)
					conn.Close()
					break ReceiveLoop
				}
				// Timeouts are fine, just try reserve again.
				if connErr.Err == beanstalk.ErrTimeout {
					continue
				}
			}

			log.Println("JobID:", id, "file:", string(filename))
			detectionResult, err := alpr.RecognizeByFilePath(string(filename))
			if err != nil {
				// If the file doesn't exist it might have been deleted. Log error
				// and the job will be deleted below. Consider burying it, too.
				// If OpenALPR throws an error, what do we do? We could exit the
				// application and allow systemd to start us up again perhaps?
				log.Println("[ERROR]: ALPR:", err)
			}
			if len(detectionResult.Plates) > 0 {
				log.Printf("At least one plate match: %+v", detectionResult.Plates[0].BestPlate)
				jsonResult, _ := json.Marshal(detectionResult)

				detectionEventStr := fmt.Sprintf("{\"filename\": \"%s\", \"event\": %s}", filename, jsonResult)
				log.Printf(detectionEventStr)

				// Eventbytes, priority, delay, time-to-run
				detectionEventId, err := detectionTube.Put([]byte(detectionEventStr), 1, 0, 30*time.Second)
				if err != nil {
					log.Println("[ERROR]: Beanstalk:", err)
					// If beanstalk goes away, we can't delete the job either so just continue.
					continue
				}
				log.Println("Added new event to", detectionTubeName, "id:", detectionEventId)
			} else {
				log.Println("No plate found, deleting file")
				err := os.Remove(string(filename))
				if err != nil {
					log.Println("[ERROR]: Unable to delete the file:", err)
				}
			}

			err = conn.Delete(id)
			if err != nil {
				// Maybe beanstalkd connection went away, reconnect will be attempted
				// on the next iteration.
				log.Println("[ERROR]: deleting job:", err)
			}
		}
	}
}
