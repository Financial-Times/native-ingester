package resources

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthCheckWithoutProducer(t *testing.T) {
	log := logger.NewUnstructuredLogger()
	if testing.Short() {
		t.Skip("Skipping test as it requires a connection to Kafka.")
	}

	consumerConfig := kafka.ConsumerConfig{
		BrokersConnectionString: "localhost:29092",
		ConsumerGroup:           "testGroup",
		Options:                 kafka.DefaultConsumerOptions(),
	}

	kafkaTopic := []*kafka.Topic{
		kafka.NewTopic("testTopic", kafka.WithLagTolerance(int64(500))),
	}

	consumer, err := kafka.NewConsumer(consumerConfig, kafkaTopic, log)
	require.NoError(t, err)

	nw := new(mocks.WriterMock)
	hc := NewHealthCheck(consumer, nil, nw, "http://test-panic-guide.com", log)

	assert.Nil(t, hc.producer)
	assert.NotNil(t, hc.consumer)
	assert.NotNil(t, hc.writer)
	assert.NotNil(t, hc.panicGuide)
}

func TestNewHealthCheckWithProducer(t *testing.T) {
	log := logger.NewUnstructuredLogger()
	if testing.Short() {
		t.Skip("Skipping test as it requires a connection to Kafka.")
	}

	consumerConfig := kafka.ConsumerConfig{
		BrokersConnectionString: "localhost:29092",
		ConsumerGroup:           "testGroup",
		Options:                 kafka.DefaultConsumerOptions(),
	}

	kafkaTopic := []*kafka.Topic{
		kafka.NewTopic("testTopic", kafka.WithLagTolerance(int64(500))),
	}

	consumer, err := kafka.NewConsumer(consumerConfig, kafkaTopic, log)
	require.NoError(t, err)

	producerConfig := kafka.ProducerConfig{
		BrokersConnectionString: "localhost:29092",
		Topic:                   "testProducerTopic",
		Options:                 kafka.DefaultProducerOptions(),
	}

	producer, err := kafka.NewProducer(producerConfig)
	require.NoError(t, err)

	nw := new(mocks.WriterMock)
	hc := NewHealthCheck(consumer, producer, nw, "http://test-panic-guide.com", log)

	assert.NotNil(t, hc.producer)
	assert.NotNil(t, hc.consumer)
	assert.NotNil(t, hc.writer)
	assert.NotNil(t, hc.panicGuide)
}

func TestHappyHealthCheckWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":`, "Producer healthcheck should not appear")
}

func TestHappyHealthCheckWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyConsumerHealthCheckWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(errors.New("Screw you guys I'm going home!"))
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":false`, "Consumer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":`, "Producer healthcheck should not appear")
}

func TestUnhappyConsumerMonitorHealthCheckWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(errors.New("screw you guys I'm going home"))
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerMonitorCheck","ok":false`, "Consumer monitor healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":`, "Producer healthcheck should not appear")
}

func TestUnhappyConsumerHealthCheckWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(errors.New("screw you guys I'm going home"))
	c.On("MonitorCheck").Return(errors.New("screw you guys I'm going home"))
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":false`, "Consumer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyNativeWriterHealthCheckWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":false`, "Native writer healthcheck should be unhappy")
	assert.NotContains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":`, "Producer healthcheck should not appear")
}

func TestUnhappyNativeWriterHealthCheckWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":false`, "Native writer healthcheck should be unhappy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":true`, "Producer healthcheck should be happy")
}

func TestUnhappyProducerHealthCheck(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(errors.New("I'm not fat, I'm big-boned."))
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	req := httptest.NewRequest("GET", "http://example.com/__health", nil)
	w := httptest.NewRecorder()

	hc.Handler()(w, req)

	assert.Equal(t, 200, w.Code, "It should return HTTP 200 OK")

	assert.Contains(t, w.Body.String(), `"name":"ConsumerQueueReachable","ok":true`, "Consumer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"NativeWriterReachable","ok":true`, "Native writer healthcheck should be happy")
	assert.Contains(t, w.Body.String(), `"name":"ProducerQueueReachable","ok":false`, "Producer healthcheck should be unhappy")
}

func TestHappyGTGCheckWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.True(t, status.GoodToGo)
	assert.Empty(t, status.Message)
}

func TestHappyGTGCheckWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.True(t, status.GoodToGo)
	assert.Empty(t, status.Message)
}

func TestUnhappyConsumerGTGWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(errors.New("Screw you guys I'm going home!"))
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Screw you guys I'm going home!", status.Message)
}

func TestUnhappyConsumerGTGWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(errors.New("Screw you guys I'm going home!"))
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm a happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Screw you guys I'm going home!", status.Message)
}

func TestUnhappyNativeWriterGTGWithoutProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	hc := HealthCheck{
		consumer: c,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Oh, my God, they killed Kenny!", status.Message)
}

func TestUnhappyNativeWriterGTGWithProducer(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an unhappy writer", errors.New("Oh, my God, they killed Kenny!"))
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(nil)
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "Oh, my God, they killed Kenny!", status.Message)
}

func TestUnhappyGTGCheck(t *testing.T) {
	c := new(mocks.ConsumerMock)
	c.On("ConnectivityCheck").Return(nil)
	c.On("MonitorCheck").Return(nil)
	nw := new(mocks.WriterMock)
	nw.On("ConnectivityCheck").Return("I'm an happy writer", nil)
	p := new(mocks.ProducerMock)
	p.On("ConnectivityCheck").Return(errors.New("I'm not fat, I'm big-boned."))
	hc := HealthCheck{
		consumer: c,
		producer: p,
		writer:   nw,
		logger:   logger.NewUnstructuredLogger(),
	}

	status := hc.GTG()

	assert.False(t, status.GoodToGo)
	assert.Equal(t, "I'm not fat, I'm big-boned.", status.Message)
}
