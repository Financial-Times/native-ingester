package resources

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/Financial-Times/service-status-go/gtg"
)

// HealthCheck implements the healthcheck for the native ingester
type HealthCheck struct {
	writer     native.Writer
	consumer   kafkaConsumer
	producer   kafkaProducer
	panicGuide string
	logger     *logger.UPPLogger
}

type kafkaConsumer interface {
	ConnectivityCheck() error
	MonitorCheck() error
}

type kafkaProducer interface {
	ConnectivityCheck() error
}

// NewHealthCheck return a new instance of a native ingester HealthCheck
func NewHealthCheck(consumer kafkaConsumer, producer kafkaProducer, writer native.Writer, panicGuide string, logger *logger.UPPLogger) *HealthCheck {
	return &HealthCheck{
		writer:     writer,
		consumer:   consumer,
		producer:   producer,
		panicGuide: panicGuide,
		logger:     logger,
	}
}

func (hc *HealthCheck) consumerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "consumer-queue",
		BusinessImpact:   "Native content or metadata will not reach this app, nor will they be stored in native store",
		Name:             "ConsumerQueueReachable",
		PanicGuide:       hc.panicGuide,
		Severity:         2,
		TechnicalSummary: "Consumer message queue is not reachable/healthy",
		Checker:          check(hc.consumer.ConnectivityCheck, hc.logger, "Consumer connectivity"),
	}
}

func (hc *HealthCheck) consumerMonitorCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "consumer-lag-check",
		BusinessImpact:   "Native content or metadata publishing is slowed down.",
		Name:             "ConsumerMonitorCheck",
		PanicGuide:       hc.panicGuide,
		Severity:         3,
		TechnicalSummary: kafka.LagTechnicalSummary,
		Checker:          check(hc.consumer.MonitorCheck, hc.logger, "Consumer status"),
	}
}

func (hc *HealthCheck) producerQueueCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "producer-queue",
		BusinessImpact:   "Content or metadata will not reach the end of the publishing pipeline",
		Name:             "ProducerQueueReachable",
		PanicGuide:       hc.panicGuide,
		Severity:         2,
		TechnicalSummary: "Producer message queue is not reachable/healthy",
		Checker:          check(hc.producer.ConnectivityCheck, hc.logger, "Producer connectivity"),
	}
}

func (hc *HealthCheck) nativeWriterCheck() fthealth.Check {
	return fthealth.Check{
		ID:               "native-writer",
		BusinessImpact:   "Content or metadata will not be written in the native store nor will they reach the end of the publishing pipeline",
		Name:             "NativeWriterReachable",
		PanicGuide:       "https://runbooks.in.ft.com/nativerw",
		Severity:         2,
		TechnicalSummary: "Native writer is not reachable/healthy",
		Checker:          hc.writer.ConnectivityCheck,
	}
}

func check(fn func() error, logger *logger.UPPLogger, component string) func() (string, error) {
	return func() (string, error) {
		if err := fn(); err != nil {
			logger.WithError(err).Errorf("%s healthcheck failed", component)
			return "", err
		}

		return "OK", nil
	}
}

// Handler returns the HTTP handler of the healthcheck
func (hc *HealthCheck) Handler() func(w http.ResponseWriter, req *http.Request) {
	checks := []fthealth.Check{hc.consumerQueueCheck(), hc.nativeWriterCheck(), hc.consumerMonitorCheck()}
	if hc.producer != nil {
		checks = append(checks, hc.producerQueueCheck())
	}

	healthCheck := fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  "native-ingester",
			Name:        "Native Ingester Healthcheck",
			Description: "It checks if kafka and native writer are available",
			Checks:      checks,
		},
		Timeout: 10 * time.Second,
	}

	return fthealth.Handler(healthCheck)
}

func (hc *HealthCheck) GTG() gtg.Status {
	consumerCheck := func() gtg.Status {
		return gtgCheck(hc.consumer.ConnectivityCheck, hc.logger, "Consumer")
	}

	writerCheck := func() gtg.Status {
		return writerGtgCheck(hc.writer.ConnectivityCheck)
	}

	if hc.producer != nil {
		producerCheck := func() gtg.Status {
			return gtgCheck(hc.producer.ConnectivityCheck, hc.logger, "Producer")
		}
		return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, producerCheck, writerCheck})()
	}

	return gtg.FailFastParallelCheck([]gtg.StatusChecker{consumerCheck, writerCheck})()
}

func gtgCheck(handler func() error, logger *logger.UPPLogger, component string) gtg.Status {
	if err := handler(); err != nil {
		logger.WithError(err).Errorf("%s GTG check failed", component)
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}

func writerGtgCheck(handler func() (string, error)) gtg.Status {
	if _, err := handler(); err != nil {
		return gtg.Status{GoodToGo: false, Message: err.Error()}
	}
	return gtg.Status{GoodToGo: true}
}
