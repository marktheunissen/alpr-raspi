# LPR on the Raspberry Pi

| :warning: **This project is not maintained or supported in any way** |
| --- |

A set of services written in Go for running [OpenALPR](https://github.com/openalpr/openalpr) on the Raspberry Pi. These are written in such a way that they add missing functionality to the open source version of OpenALPR (e.g. multiple detection threads).

The application is broken up into microservices, so that a cluster of Raspberry Pis could be effeciently used to increase the detection rate on a single camera. For example, you might want more detection threads running and so dedicate more CPU to detection.

## Local development Mac OSX

    postgres -D /usr/local/var/postgres
    beanstalkd -V
    cd plate_detector && . lpr.env && go run plate_detector.go
    cd uploader && . lpr.env && make run
    cd watcher && . lpr.env && go run watcher.go
    cd watcher/testdata
    ./create_jpgs

## Remote DB

Use `scripts/schema.sql` to create the schema for the main PostGres DB.

## Motion detection

LPR-Raspi relies on the excellent [Motion](https://motion-project.github.io/) project, which does motion detection on video feeds.

The folder `/motion_docker` allows you to run Linux motion daemon on OSX. There are `make` commands in there that build and run the container.

## Streaming

It's possible to create a MJPG stream of a local video for testing purposes. First grab sample video and resize it:

    ffmpeg -i sample.avi -vcodec libx264 -preset slow -profile high -vf scale=768:432 432p/sample.avi

From VCL create an MJPEG stream to Raspi:

    vlc \
    --loop \
    -vvv \
    sample.avi \
    --sout "#transcode{vcodec=MJPG,fps=5}:std{access=http{mime=multipart/x-mixed-replace;boundary=--7b3cc56e5f51db803f790dad720ed50a},mux=mpjpeg,dst=10.0.0.5:8082/go.mjpg,delay=0}"

And to Motion on localhost or Docker:

    vlc \
    --loop \
    -vvv \
    sample.avi \
    --sout "#transcode{vcodec=MJPG,fps=5}:std{access=http{mime=multipart/x-mixed-replace;boundary=--7b3cc56e5f51db803f790dad720ed50a},mux=mpjpeg,dst=127.0.0.1:8082/go.mjpg,delay=0}"

Then in browser:

    http://10.0.0.5:8082/go.mjpg

You can try specify bitrate / FPS, but prefer just using fps.

    vb=10000,fps=10
    vb=5000,fps=5
    vb=2000,fps=2

## Plate Detector

The `plate_detector` app runs `openalpr.RecognizeByFilePath` and detects plates in the images using OpenALPR.

## Uploader

The Uploader service picks events off the beanstalk queue, creates crops and thumbnails of the JPG event image using GraphicsMagick, and uploads the results to the central PostGres DB in the cloud, and Amazon S3.

### Local development

For GraphicsMagick integration on a Mac:

    brew install homebrew/versions/giflib5
    export CGO_LDFLAGS="-L/usr/local/opt/giflib5/lib"
    export CGO_CPPFLAGS="-I/usr/local/opt/giflib5/include"
    go get github.com/rainycape/magick

Local DB for the deduplication:

    brew install postgres
    postgres -D /usr/local/var/postgres
    createdb `whoami`
    psql

Install the schema as per `uploader/schema/uploader.sql`

## Beanstalk

Beanstalk is the backbone queue (message bus) that the services use to pass messages.

- https://godoc.org/github.com/kr/beanstalk
- http://kr.github.io/beanstalkd/
- https://github.com/kr/beanstalkd

To start on OSX in foreground:

    beanstalkd -V

    # Get some stats
    echo -e "stats\r\n" | nc localhost 11300
    echo -e "stats-tube detection_events\r\n" | nc localhost 11300
    echo -e "stats-tube motion_events\r\n" | nc localhost 11300

On Raspi (systemd):

    systemctl start beanstalkd

## ALPR License Plate Recognition

The plate recognition is performed using OpenALPR integration:

- https://groups.google.com/forum/#!forum/openalpr
- https://github.com/openalpr/openalpr
- http://doc.openalpr.com/

Installing ALPR on OSX:

    brew tap homebrew/science
    brew install openalpr

Running OpenALPR:

    alpr -c eu -j 01-20160609180828-02.jpg | jq '.'

Compiling ALPR on Raspi can be done using the [lpr-deps](https://github.com/marktheunissen/lpr-deps) repo.
