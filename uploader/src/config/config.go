package config

import (
	"log"
	"os"
	"time"

	flags "github.com/jessevdk/go-flags"
)

// Options describes all the CLI flags that can be passed
type Options struct {
	S3Bucket              string `env:"UPLOADER_S3_BUCKET" required:"true" short:"a"`
	S3Prefix              string `env:"UPLOADER_S3_PREFIX" required:"true" short:"b"`
	S3Region              string `env:"UPLOADER_S3_REGION" default:"eu-west-1" short:"c"`
	RemotePostgresPass    string `env:"UPLOADER_REMOTE_POSTGRES_PASS" required:"true" short:"d"`
	LocalPostgresPass     string `env:"UPLOADER_LOCAL_POSTGRES_PASS" required:"true" short:"e"`
	Camera                string `env:"UPLOADER_CAMERA" default:"lpr-camera" short:"f"`
	Site                  string `env:"UPLOADER_SITE" default:"lpr-site" short:"g"`
	PlateImageHeight      int    `env:"UPLOADER_PLATE_IMAGE_HEIGHT" default:"60" short:"h"`
	PlateImageQuality     uint   `env:"UPLOADER_PLATE_IMAGE_QUALITY" default:"80" short:"i"`
	FrameImageHeight      int    `env:"UPLOADER_FRAME_IMAGE_HEIGHT" default:"200" short:"j"`
	FrameImageQuality     int    `env:"UPLOADER_FRAME_IMAGE_QUALITY" default:"70" short:"k"`
	EventIntervalTimeSecs int    `env:"UPLOADER_EVENT_INTERVAL_TIME" default:"15" short:"l"`
	FrameDir              string `env:"UPLOADER_FRAME_DIR" default:"./" short:"m"`
	PlateDir              string `env:"UPLOADER_PLATE_DIR" default:"./" short:"n"`
	AccessKey             string `env:"AWS_ACCESS_KEY_ID" required:"true" short:"o"`
	SecretKey             string `env:"AWS_SECRET_ACCESS_KEY" required:"true" short:"p"`
	EventIntervalTime     time.Duration
}

// Opts is the application config struct that we allow external access too
var Opts Options

func Init(args []string) {
	_, err := flags.ParseArgs(&Opts, args)
	if err != nil {
		log.Println(err)
		log.Println("Missing ENV vars containing configuration, try `. lpr.env`")
		os.Exit(1)
	}
	Opts.EventIntervalTime = time.Duration(Opts.EventIntervalTimeSecs) * time.Second
}
