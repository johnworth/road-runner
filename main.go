// road-runner
//
// Executes jobs based on a JSON blob serialized to a file.
// Each step of the job runs inside a Docker container. Job results are
// transferred back into iRODS with the porklock tool. Job status updates are
// posted to the **jobs.updates** topic in the **jobs** exchange.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/cyverse-de/configurate"
	"github.com/cyverse-de/dockerops"
	"github.com/cyverse-de/logcabin"
	"github.com/cyverse-de/messaging"
	"github.com/cyverse-de/model"
	"github.com/cyverse-de/road-runner/fs"
	"github.com/cyverse-de/version"
	"github.com/streadway/amqp"

	"github.com/spf13/viper"
)

var (
	job              *model.Job
	dckr             *dockerops.Docker
	client           *messaging.Client
	amqpExchangeName string
	amqpExchangeType string
)

var log = logrus.WithFields(logrus.Fields{
	"service": "road-runner",
	"art-id":  "road-runner",
	"group":   "org.cyverse",
})

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	logcabin.Init("road-runner", "road-runner")
	sigquitter := make(chan bool)
	sighandler := InitSignalHandler()
	sighandler.Receive(
		sigquitter,
		func(sig os.Signal) {
			log.Info("Received signal:", sig)
			if dckr == nil {
				log.Warn("Docker client is nil, can't clean up. Probably don't need to.")
			}
			if job == nil {
				log.Warn("Info didn't get parsed from the job file, can't clean up. Probably don't need to.")
			}
			if dckr != nil && job != nil {
				cleanup(dckr, job)
			}
			if client != nil && job != nil {
				fail(client, job, fmt.Sprintf("Received signal %s", sig))
			}
			os.Exit(-1)
		},
		func() {
			log.Info("Signal handler is quitting")
		},
	)
	signal.Notify(
		sighandler.Signals,
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGSTOP,
		syscall.SIGQUIT,
	)
	var (
		showVersion = flag.Bool("version", false, "Print the version information")
		jobFile     = flag.String("job", "", "The path to the job description file")
		cfgPath     = flag.String("config", "", "The path to the config file")
		writeTo     = flag.String("write-to", "/opt/image-janitor", "The directory to copy job files to.")
		dockerURI   = flag.String("docker", "unix:///var/run/docker.sock", "The URI for connecting to docker.")
		err         error
		cfg         *viper.Viper
	)
	flag.Parse()
	if *showVersion {
		version.AppVersion()
		os.Exit(0)
	}
	if *cfgPath == "" {
		log.Fatal("--config must be set.")
	}
	log.Infof("Reading config from %s\n", *cfgPath)
	if _, err = os.Open(*cfgPath); err != nil {
		log.Fatal(*cfgPath)
	}
	cfg, err = configurate.Init(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Done reading config from %s\n", *cfgPath)
	if *jobFile == "" {
		log.Fatal("--job must be set.")
	}
	data, err := ioutil.ReadFile(*jobFile)
	if err != nil {
		log.Fatal(err)
	}
	job, err = model.NewFromData(cfg, data)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = os.Open(*writeTo); err != nil {
		log.Fatal(err)
	}
	if err = fs.CopyJobFile(fs.FS, job.InvocationID, *jobFile, *writeTo); err != nil {
		log.Fatal(err)
	}
	uri := cfg.GetString("amqp.uri")
	amqpExchangeName = cfg.GetString("amqp.exchange.name")
	amqpExchangeType = cfg.GetString("amqp.exchange.type")
	client, err = messaging.NewClient(uri, true)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	client.SetupPublishing(amqpExchangeName)
	dckr, err = dockerops.NewDocker(context.Background(), cfg, *dockerURI)
	if err != nil {
		fail(client, job, "Failed to connect to local docker socket")
		log.Fatal(err)
	}
	// The channel that the exit code will be passed along on.
	exit := make(chan messaging.StatusCode)
	// Could probably reuse the exit channel, but that's less explicit.
	finalExit := make(chan messaging.StatusCode)
	// Launch the go routine that will handle job exits by signal or timer.
	go Exit(exit, finalExit)
	go client.Listen()
	client.AddDeletableConsumer(
		amqpExchangeName,
		amqpExchangeType,
		messaging.StopQueueName(job.InvocationID),
		messaging.StopRequestKey(job.InvocationID),
		func(d amqp.Delivery) {
			d.Ack(false)
			running(client, job, "Received stop request")
			exit <- messaging.StatusKilled
		})
	go Run(dckr, client, job, exit)
	exitCode := <-finalExit
	if err = fs.DeleteJobFile(fs.FS, job.InvocationID, *writeTo); err != nil {
		log.Errorf("%+v", err)
	}
	os.Exit(int(exitCode))
}
