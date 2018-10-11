package main

import (
	raven "github.com/getsentry/raven-go"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/trilopin/godinary/storage"
	//"godinary/godinary/interactors"
	"log"
	"os"
)

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
	log.Printf("Connecting to rabbitmq with URL %s ...", rabbitmqURL)
	connection, channel, err := openRabbitQueueChannel(rabbitmqURL, queueName)
	defer connection.Close()
	defer channel.Close()

	log.Printf("Connected correctly")
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
	log.Printf("Starting queue %s consume, waiting for messages...", queueName)

	storage := setupStorage()
	err = storage.Init()
	failOnError(err, "Could not initiate storage session")
	log.Printf("Storage initiated correctly")

	semaphore := make(chan struct{}, viper.GetInt("max_rabbit_requests"))

	for msg := range msgs {
		<-semaphore
		go func(semaphore chan<- struct{}) {
			message, err := cacheImage(msg.Body, storage, false)
			log.Printf(message)
			if err != nil {
				//Notify Sentry
				log.Printf(err)
			}
			semaphore <- struct{}{}
		}(semaphore)

	}

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func setupStorage() storage.Driver {
	var storage storage.Driver
	if viper.GetString("storage") == "gs" {
		GCEProject := viper.GetString("gce_project")
		GSBucket := viper.GetString("gs_bucket")
		GSCredentials := viper.GetString("gs_credentials")
		if GCEProject == "" {
			log.Fatalln("GoogleStorage project should be setted")
		}
		if GSBucket == "" {
			log.Fatalln("GoogleStorage bucket should be setted")
		}
		if GSCredentials == "" {
			log.Fatalln("GoogleStorage Credentials shold be setted")
		}
		storage := &storage.GoogleStorageDriver{
			BucketName:  GSBucket,
			ProjectName: GCEProject,
			Credentials: GSCredentials,
		}
	} else {
		FSBase := viper.GetString("fs_base")
		if FSBase == "" {
			log.Fatalln("filesystem base path should be setted")
		}
		storage = storage.NewFileDriver(FSBase)
	}
	return storage
}

func setupConfig() {
	//Rabbit flag setup
	pflag.String("rabbitmq_url", "amqp://guest:guest@localhost:5672/", "RabbitMQ DSN")
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
	flag.Int("max_rabbit_requests", 100, "Maximum number of simultaneous downloads")

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
		raven.CapturePanic(func() {
			// do all of the scary things here
		}, nil)
	}

	log.SetOutput(os.Stdout)
}

func main() {
	initRabbitCacheImages(viper.GetString("rabbitmq_queue"), viper.GetString("rabbitmq_url"))
}
