package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/config"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/native-ingester/queue"
	"github.com/Financial-Times/native-ingester/resources"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/gorilla/mux"
	cli "github.com/jawher/mow.cli"
)

func main() {
	app := cli.App("native-ingester", "A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue")

	port := app.String(cli.StringOpt{
		Name:   "port",
		Value:  "8080",
		Desc:   "Port to listen on",
		EnvVar: "PORT",
	})

	// Read queue configuration
	kafkaClusterArn := app.String(cli.StringOpt{
		Name:   "kafka-cluster-arn",
		Desc:   "MSK cluster ARN.",
		EnvVar: "KAFKA_CLUSTER_ARN",
	})
	kafkaAddress := app.String(cli.StringOpt{
		Name:   "kafka-address",
		Value:  "kafka:9092",
		Desc:   "MSK address to connect to the producer and consumer queues.",
		EnvVar: "KAFKA_ADDRESS",
	})
	consumerGroup := app.String(cli.StringOpt{
		Name:   "consumer-group",
		Value:  "",
		Desc:   "Group used to read the messages from the queue.",
		EnvVar: "CONSUMER_GROUP",
	})
	consumerTopic := app.String(cli.StringOpt{
		Name:   "consumer-topic",
		Value:  "",
		Desc:   "The topic to read the messages from.",
		EnvVar: "CONSUMER_TOPIC",
	})
	lagTolerance := app.Int(cli.IntOpt{
		Name:   "lag-tolerance",
		Value:  500,
		Desc:   "Configured kafka consumer lag tolerance.",
		EnvVar: "KAFKA_LAG_TOLERANCE",
	})
	producerTopic := app.String(cli.StringOpt{
		Name:   "producer-topic",
		Value:  "",
		Desc:   "The topic to write the messages to.",
		EnvVar: "PRODUCER_TOPIC",
	})
	// Native writer configuration
	nativeWriterAddress := app.String(cli.StringOpt{
		Name:   "native-writer-address",
		Value:  "",
		Desc:   "Address (URL) of service that writes persistently the native content",
		EnvVar: "NATIVE_RW_ADDRESS",
	})
	contentUUIDFields := app.Strings(cli.StringsOpt{
		Name:   "content-uuid-fields",
		Value:  []string{},
		Desc:   "List of JSONPaths that point to UUIDs in native content bodies. e.g. uuid,post.uuid,data.uuidv3",
		EnvVar: "NATIVE_CONTENT_UUID_FIELDS",
	})
	contentType := app.String(cli.StringOpt{
		Name:   "content-type",
		Value:  "",
		Desc:   "The type of the content (for logging purposes, e.g. \"Content\" or \"Annotations\") the application is able to handle.",
		EnvVar: "CONTENT_TYPE",
	})
	appName := app.String(cli.StringOpt{
		Name:   "appName",
		Value:  "native-ingester",
		Desc:   "The name of the application",
		EnvVar: "APP_NAME",
	})
	configFile := app.String(cli.StringOpt{
		Name:   "config",
		Value:  "",
		Desc:   "Config file (e.g. config.json)",
		EnvVar: "CONFIG",
	})
	panicGuideUrl := app.String(cli.StringOpt{
		Name:   "panic-guide",
		Value:  "",
		Desc:   "Panic Guide URL",
		EnvVar: "PANIC_GUIDE_URL",
	})
	logLevel := app.String(cli.StringOpt{
		Name:   "logLevel",
		Value:  "INFO",
		Desc:   "Log level for the service",
		EnvVar: "LOG_LEVEL",
	})

	app.Action = func() {
		logger := logger.NewUPPLogger(*appName, *logLevel)
		conf, err := config.ReadConfig(*configFile)
		if err != nil {
			logger.WithError(err).Fatal("Error reading the configuration")
		}

		if *panicGuideUrl == "" {
			logger.Fatal("panicGuideUrl is empty")
		}

		logger.Infof("[Startup] Using UUID paths configuration: %#v", *contentUUIDFields)
		bodyParser := native.NewContentBodyParser(*contentUUIDFields)
		writer := native.NewWriter(*nativeWriterAddress, *conf, bodyParser, logger)
		logger.Infof("[Startup] Using native writer configuration: %#v", writer)

		mh := queue.NewMessageHandler(writer, *contentType, logger)

		var messageProducer *kafka.Producer
		if *producerTopic != "" {
			producerConfig := kafka.ProducerConfig{
				ClusterArn:              kafkaClusterArn,
				BrokersConnectionString: *kafkaAddress,
				Topic:                   *producerTopic,
				Options:                 kafka.DefaultProducerOptions(),
			}
			messageProducer, err = kafka.NewProducer(producerConfig)
			if err != nil {
				logger.WithError(err).Fatal("Failed to create Kafka producer")
			}
			defer messageProducer.Close()

			logger.Infof("[Startup] Producer: %#v", messageProducer)
			mh.ForwardTo(messageProducer)
		}

		consumerConfig := kafka.ConsumerConfig{
			ClusterArn:              kafkaClusterArn,
			BrokersConnectionString: *kafkaAddress,
			ConsumerGroup:           *consumerGroup,
			Options:                 kafka.DefaultConsumerOptions(),
		}

		kafkaTopic := []*kafka.Topic{
			kafka.NewTopic(*consumerTopic, kafka.WithLagTolerance(int64(*lagTolerance))),
		}

		messageConsumer, err := kafka.NewConsumer(consumerConfig, kafkaTopic, logger)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create Kafka consumer")
		}

		logger.Infof("[Startup] Consumer: %#v", messageConsumer)

		go func() {
			err = enableHealthCheck(*port, messageConsumer, messageProducer, writer, *panicGuideUrl)
			if err != nil {
				logger.WithError(err).Fatal("Couldn't set up HTTP listener")
			}
		}()

		messageConsumer.Start(mh.HandleMessage)
		defer messageConsumer.Close()

		ch := make(chan os.Signal)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
	}

	err := app.Run(os.Args)
	if err != nil {
		println(err)
	}
}

func enableHealthCheck(port string, consumer *kafka.Consumer, producer *kafka.Producer, nw native.Writer, pg string) error {
	hc := resources.NewHealthCheck(consumer, producer, nw, pg)

	r := mux.NewRouter()
	r.HandleFunc("/__health", hc.Handler())
	r.HandleFunc(httphandlers.GTGPath, httphandlers.NewGoodToGoHandler(hc.GTG)).Methods("GET")
	r.HandleFunc(httphandlers.BuildInfoPath, httphandlers.BuildInfoHandler).Methods("GET")
	r.HandleFunc(httphandlers.PingPath, httphandlers.PingHandler).Methods("GET")

	http.Handle("/", r)
	return http.ListenAndServe(":"+port, nil)
}
