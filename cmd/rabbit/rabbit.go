package main

import (
	raven "github.com/getsentry/raven-go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"godinary/interactors"
	"godinary/storage"
	"log"
	"os"
)

var logger *log.Logger

func openRabbitQueueChannel(rabbitmqURL string, queue string) (*amqp.Connection, *amqp.Channel, error) {
	//Make a connection
	conn, err := amqp.Dial(rabbitmqURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	//Create a channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	//Declare a queue
	_, err = ch.QueueDeclare(
		queue, // name of the queue
		false, // should the message be persistent? also queue will survive if the cluster gets reset
		false, // autodelete if there's no consumers (like queues that have anonymous names, often used with fanout exchange)
		false, // exclusive means I should get an error if any other consumer subsribes to this queue
		false, // no-wait means I don't want RabbitMQ to wait if there's a queue successfully setup
		nil,   // arguments for more advanced configuration
	)

	failOnError(err, "Failed to declare queue in RabbitMQ")

	return conn, ch, nil
}

func initRabbitCacheImages(queueName string, rabbitmqURL string) {

	logger.Printf("Connecting to rabbitmq with URL %s ...", rabbitmqURL)
	connection, channel, err := openRabbitQueueChannel(rabbitmqURL, queueName)
	defer connection.Close()
	defer channel.Close()

	logger.Printf("Connected correctly")
	msgs, err := channel.Consume(
		queueName, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	failOnError(err, "Could not consume messages from RabbitMQ")
	logger.Printf("Starting queue %s consume, waiting for messages...", queueName)

	storageDriver := setupStorage()
	err = storageDriver.Init()
	failOnError(err, "Could not initiate storage session")
	logger.Printf("Storage initiated correctly, awaiting image urls...")

	semaphore := make(chan struct{}, viper.GetInt("max_rabbit_requests"))
	async := false
	if viper.GetString("async_storage") == "true" {
		async = true
	}
	for msg := range msgs {
		semaphore <- struct{}{}
		go func(image_url string) {
			raven.CapturePanic(func() {
				interactors.DownloadAndCacheImage(image_url, storageDriver, async, logger)
			}, nil)
			<-semaphore
		}(string(msg.Body[:]))
	}

}

func failOnError(err error, msg string) {
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		logger.Panicf("%s: %s", msg, err)
	}
}

func setupStorage() storage.Driver {
	var storageDriver storage.Driver
	if viper.GetString("storage") == "gs" {
		GCEProject := viper.GetString("gce_project")
		GSBucket := viper.GetString("gs_bucket")
		GSCredentials := viper.GetString("gs_credentials")
		if GCEProject == "" {
			logger.Panicln("GoogleStorage project should be setted")
		}
		if GSBucket == "" {
			logger.Panicln("GoogleStorage bucket should be setted")
		}
		if GSCredentials == "" {
			logger.Panicln("GoogleStorage Credentials shold be setted")
		}
		storageDriver = &storage.GoogleStorageDriver{
			BucketName:  GSBucket,
			ProjectName: GCEProject,
			Credentials: GSCredentials,
		}
	} else {
		FSBase := viper.GetString("fs_base")
		if FSBase == "" {
			logger.Panicln("filesystem base path should be setted")
		}
		storageDriver = storage.NewFileDriver(FSBase)
	}
	return storageDriver
}

func setupConfig() {
	//Rabbit flag setup
	pflag.String("rabbitmq_url", "amqp://guest:guest@godinary.rabbitmq:5672//", "RabbitMQ DSN")
	pflag.String("rabbitmq_queue", "core_godinary", "Name of RabbitMQ queue to get images")
	//Decide wich type of storage we will use
	pflag.String("storage", "fs", "Storage type: 'gs' for google storage or 'fs' for filesystem")
	//Google Cloud Storage flag setup)
	pflag.String("gce_project", "", "GS option: Sentry DSN for error tracking")
	pflag.String("gs_bucket", "", "GS option: Bucket name")
	pflag.String("gs_credentials", "", "GS option: Path to service account file with Google Storage credentials")
	//Local Storage flag setup TODO:Create relative path for fs_base
	pflag.String("fs_base", "", "FS option: Base dir for filesystem storage")
	//Sentry
	pflag.String("sentry_url", "", "Sentry DSN for error tracking")
	pflag.String("release", "", "Release hash to notify sentry")
	//Max downloads
	pflag.Int("max_rabbit_requests", 100, "Maximum number of simultaneous downloads")
	//Async storage to googlecloud
	pflag.String("async_storage", "true", "Storage Option, if 'true' ,storage will be asynchronous")
	//Number threads

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	// env setup
	viper.AutomaticEnv()
	viper.SetEnvPrefix("godinary")
	pflag.VisitAll(func(f *pflag.Flag) {
		viper.BindEnv(f.Name)
	})
}

func init() {

	setupConfig()

	if viper.GetString("sentry_url") != "" {
		raven.SetDSN(viper.GetString("sentry_url"))
		if viper.GetString("release") != "" {
			raven.SetRelease(viper.GetString("release"))
		}
	}

	logger = log.New(os.Stdout, "rabbitmq_worker: ", log.Lshortfile|log.LstdFlags)
	logger.SetOutput(os.Stdout)
}

func main() {
	initRabbitCacheImages(viper.GetString("rabbitmq_queue"), viper.GetString("rabbitmq_url"))
}
