package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"img"
	"log"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/openalpr/openalpr"

	"database/sql"

	_ "github.com/lib/pq"

	"config"
	"utils"

	"github.com/jpillora/backoff"
	"github.com/kr/beanstalk"
)

type EventPayload struct {
	Filename    string               `json:"filename"`
	AlprResults openalpr.AlprResults `json:"event"`
}

func main() {
	log.Println("Uploader startup")

	// Parse from the command line, allows testing to work.
	config.Init(os.Args[1:])

	log.Println("Image library:", img.GetImageLib())

	// Amazon S3 parameters
	log.Println("S3 Bucket:", config.Opts.S3Bucket, "prefix:", config.Opts.S3Prefix, "region:", config.Opts.S3Region)
	s3uploader := s3manager.NewUploader(session.New(&aws.Config{Region: aws.String(config.Opts.S3Region)}))

	// Remote Postgres DB parameters
	remoteConnectStr := fmt.Sprintf("postgresql://postgres:%s@lpr-cloud.dvrcam.info:5432/postgres?sslmode=disable", config.Opts.RemotePostgresPass)
	remoteDB, err := sql.Open("postgres", remoteConnectStr)
	if err != nil {
		log.Println("[ERROR]:", err)
		os.Exit(1)
	}
	defer remoteDB.Close()
	err = remoteDB.Ping()
	if err != nil {
		log.Println("[ERROR]:", err)
		os.Exit(1)
	}

	// Local Postgres DB parameters
	localConnectStr := fmt.Sprintf("postgresql://lpr:%s@localhost:5432/lpr?sslmode=disable", config.Opts.LocalPostgresPass)
	localDB, err := sql.Open("postgres", localConnectStr)
	if err != nil {
		log.Println("[ERROR]:", err)
		os.Exit(1)
	}
	defer localDB.Close()
	err = localDB.Ping()
	if err != nil {
		log.Println("[ERROR]:", err)
		os.Exit(1)
	}

	// Beanstalkd parameters
	detectionTubeName := "detection_events"
	addr := "127.0.0.1:11300"
	reserveTimeout := time.Duration(5 * time.Second)
	bs_backoff := &backoff.Backoff{
		//These are the defaults
		Min:    100 * time.Millisecond,
		Max:    30 * time.Second,
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
		log.Println("Connected, beanstalkd addr:", addr, "reserve timeout:", reserveTimeout)
		log.Println("Detection events tube:", detectionTubeName)
		defer conn.Close()
		bs_backoff.Reset()
		detectionTubeSet := beanstalk.NewTubeSet(conn, detectionTubeName)

	ReceiveLoop:
		for {
			// Returns an error after the timeout expires without receiving a job.
			id, payloadBytes, err := detectionTubeSet.Reserve(reserveTimeout)

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

			log.Println("JobID:", id)

			// Unmarshal the payload containing the filename and detection event.
			var payload EventPayload
			err = json.Unmarshal(payloadBytes, &payload)
			if err != nil {
				log.Println("[ERROR]:", err)
			}

			timestamp, err := utils.ExtractTime(payload.Filename)
			if err != nil {
				// We're supposed to be able to extract the timestamp, but
				// continue and default to UTC now if we can't.
				log.Println("[ERROR] Timestamp:", err)
				now := time.Now().UTC()
				timestamp = &now
			}

			// Iterate over all the detected plates in the image.
			for _, plate := range payload.AlprResults.Plates {
				// First check we haven't just sent this plate out
				seenRecently, err := CheckRecent(localDB, plate.BestPlate, timestamp)
				if err != nil {
					// An error checking if the plate was seen recently: we will
					// just log an error and then continue to attempt to send the event.
					log.Println("[ERROR] LocalDB:", err)
				} else if seenRecently {
					continue
				}

				// Create a plate image and upload it to S3.
				plateImgUrl := "https://lpr-events.s3-eu-west-1.amazonaws.com/placeholder/error.jpg"
				plateBytes, err := img.CreatePlateImage(payload.Filename, plate.PlatePoints)
				_, plateName := path.Split(utils.GetPlateFilename(payload.Filename))
				if err != nil {
					log.Println("[ERROR] CreatePlateImage:", err)
				} else {
					log.Println("Created plateBytes for:", plateName)
					plateImgUrl, err = UploadFile(plateName, plateBytes, s3uploader)
					if err != nil {
						log.Println("[ERROR] UploadFile:", err)
					}
					log.Println("Upload success:", plateImgUrl)
				}

				// Create a frame thumbnail and upload it
				frameImgUrl := "https://lpr-events.s3-eu-west-1.amazonaws.com/placeholder/error.jpg"
				frameBytes, err := img.CreateFrameThumbnail(payload.Filename)
				_, frameName := path.Split(utils.GetFrameFilename(payload.Filename))
				if err != nil {
					log.Println("[ERROR] CreateFrameThumbnail:", err)
				} else {
					log.Println("Created frameBytes:", frameName)
					frameImgUrl, err = UploadFile(frameName, frameBytes, s3uploader)
					if err != nil {
						log.Println("[ERROR] UploadFile:", err)
					}
					log.Println("Upload success:", frameImgUrl)
				}

				// And send the event out. In the event of errors creating the images,
				// we still send the event.
				err = SendEvent(remoteDB, plate.BestPlate, plateImgUrl, frameImgUrl, timestamp)
				if err != nil {
					log.Println("[ERROR] SendEvent RemoteDB:", err)
				}
				log.Println("Event for plate:", plate.BestPlate, "sent to remote database")
			}

			err = conn.Delete(id)
			if err != nil {
				// Maybe beanstalkd connection went away, reconnect will be attempted
				// on the next iteration.
				log.Println("[ERROR]: Delete job:", err)
			}
		}
	}
}

// CheckRecent checks whether we've seen this plate recently
func CheckRecent(db *sql.DB, plate string, timestamp *time.Time) (bool, error) {
	upsertQuery := "INSERT INTO last_seen (plate, time) VALUES (($1), ($2)) ON CONFLICT (plate) DO UPDATE SET time = ($2)"
	var last_seen time.Time

	err := db.QueryRow("SELECT time FROM last_seen WHERE plate = $1", plate).Scan(&last_seen)
	switch {
	case err == sql.ErrNoRows:
		log.Println("Plate:", plate, "not seen recently (no rows), inserting timestamp:", timestamp)
		_, err = db.Exec(upsertQuery, plate, timestamp)
		if err != nil {
			return false, err
		}
		return false, nil

	case err != nil:
		return false, err

	default:
		timeago := Round(time.Since(last_seen), time.Millisecond)
		if timeago > config.Opts.EventIntervalTime {
			log.Println("Plate:", plate, "not seen recently, marker set at", timeago, "ago, over", config.Opts.EventIntervalTime, "ago, updating timestamp:", timestamp)
			_, err = db.Exec(upsertQuery, plate, timestamp)
			if err != nil {
				return false, err
			}
			return false, nil
		} else {
			log.Println("Plate:", plate, "seen recently", timeago, "ago, updating timestamp:", timestamp)
			_, err = db.Exec(upsertQuery, plate, timestamp)
			if err != nil {
				return true, err
			}
			return true, nil
		}
	}
}

// SendEvent sends the event data to the remote Postgres database
func SendEvent(db *sql.DB, plate string, plate_image string, frame_image string, timestamp *time.Time) error {
	query := "INSERT INTO events (time, camera, plate, plate_image, frame_image, site) " +
		"VALUES (($1), ($2), ($3), ($4), ($5), ($6))"
	_, err := db.Exec(query, timestamp, config.Opts.Camera, plate, plate_image, frame_image, config.Opts.Site)
	if err != nil {
		return err
	}
	return nil
}

// UploadFile sends the give image bytes to Amazon S3.
func UploadFile(fileName string, fileBytes *bytes.Buffer, s3uploader *s3manager.Uploader) (string, error) {
	class := s3.ObjectStorageClassReducedRedundancy
	contentType := "image/jpeg"

	// Upload the file to S3 using the S3 Manager
	uploadRes, err := s3uploader.Upload(&s3manager.UploadInput{
		Bucket:       aws.String(config.Opts.S3Bucket),
		Key:          aws.String(path.Join(config.Opts.S3Prefix, fileName)),
		Body:         fileBytes,
		ContentType:  &contentType,
		StorageClass: &class,
	})
	if err != nil {
		return "", err
	}

	return uploadRes.Location, nil
}

// Round out a duration for printing
func Round(d, r time.Duration) time.Duration {
	if r <= 0 {
		return d
	}
	neg := d < 0
	if neg {
		d = -d
	}
	if m := d % r; m+m < r {
		d = d - m
	} else {
		d = d + r - m
	}
	if neg {
		return -d
	}
	return d
}
