package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"log"
	"os"
)

func openRabbitQueueChannel(rabbitmqURL string, queue string) (*amqp.Connection, *amqp.Channel, error) {
	//Make a connection
	conn, err := amqp.Dial(rabbitmqURL)
	if failOnError(err, "Failed to connect to RabbitMQ") {
		return nil, nil, err
	}

	//Create a channel
	ch, err := conn.Channel()
	if failOnError(err, "Failed to open a channel") {
		return nil, nil, err
	}
	//Declare a queue
	_, err = ch.QueueDeclare(
		queue, // name of the queue
		false, // should the message be persistent? also queue will survive if the cluster gets reset
		false, // autodelete if there's no consumers (like queues that have anonymous names, often used with fanout exchange)
		false, // exclusive means I should get an error if any other consumer subsribes to this queue
		false, // no-wait means I don't want RabbitMQ to wait if there's a queue successfully setup
		nil,   // arguments for more advanced configuration
	)

	if failOnError(err, "Failed to declare queue in RabbitMQ") {
		return nil, nil, err
	}

	return conn, ch, nil
}

func initRabbitCacheImages(queueName string, rabbitmqURL string) {
	log.Printf("Connecting to rabbitmq with URL %s ...", rabbitmqURL)
	connection, channel, err := openRabbitQueueChannel(rabbitmqURL, queueName)
	defer connection.Close()
	defer channel.Close()
	if err != nil {
		return
	}
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

	if failOnError(err, "Could not consume messages from RabbitMQ") {
		return
	}
	log.Printf("Starting queue %s consume, waiting for messages...", queueName)
	for msg := range msgs {
		log.Printf("received message %s", msg.Body)
	}

}

func failOnError(err error, msg string) bool {
	if err != nil {
		log.Printf("%s: %s", msg, err)
		return true
	}
	return false
}

func setupConfig() {
	// flags setup
	pflag.String("rabbitmq_url", "amqp://guest:guest@localhost:5672/", "RabbitMQ DSN")
	pflag.String("rabbitmq_queue", "core_godinary", "Name of RabbitMQ queue to get images")

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

	log.SetOutput(os.Stdout)
}

func main() {
	initRabbitCacheImages(viper.GetString("rabbitmq_queue"), viper.GetString("rabbitmq_url"))
}
