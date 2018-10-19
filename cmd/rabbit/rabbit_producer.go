// Publisher.go
package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"log"
)

func setupConfig() {
	//Rabbit flag setup
	pflag.String("rabbitmq_url", "amqp://guest:guest@godinary.rabbitmq:5672//", "RabbitMQ DSN")
	pflag.String("rabbitmq_queue", "core_godinary", "Name of RabbitMQ queue to get images")
	pflag.String("image_url", "", "Image url to enqueue")

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
}

func main() {
	imageURL := viper.GetString("image_url")
	if imageURL == "" {
		log.Fatalf("No image URL passed in make args")
	}

	//Make a connection
	conn, err := amqp.Dial(viper.GetString("rabbitmq_url"))
	failOnError(err, "Could not consume messages from RabbitMQ")
	defer conn.Close()

	//Ccreate a channel
	ch, err := conn.Channel()
	failOnError(err, "Could not initiate storage session")
	defer ch.Close()

	//Declare a queue
	q, err := ch.QueueDeclare(
		viper.GetString("rabbitmq_queue"), // name of the queue
		false,                             // should the message be persistent? also queue will survive if the cluster gets reset
		false,                             // autodelete if there's no consumers (like queues that have anonymous names, often used with fanout exchange)
		false,                             // exclusive means I should get an error if any other consumer subsribes to this queue
		false,                             // no-wait means I don't want RabbitMQ to wait if there's a queue successfully setup
		nil,                               // arguments for more advanced configuration
	)
	failOnError(err, "Could not declare queue")

	ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(imageURL),
		})
	log.Printf("Message sended correctly: %s", imageURL)

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
