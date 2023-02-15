package mocks

import (
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/native"
	"github.com/stretchr/testify/mock"
)

type ProducerMock struct {
	mock.Mock
}

func (p *ProducerMock) ConnectivityCheck() error {
	args := p.Called()
	return args.Error(0)
}

func (p *ProducerMock) SendMessage(msg kafka.FTMessage) error {
	args := p.Called(msg)
	return args.Error(0)
}

func (p *ProducerMock) Close() error {
	p.Called()
	return nil
}

type ConsumerMock struct {
	mock.Mock
}

func (c *ConsumerMock) ConnectivityCheck() error {
	args := c.Called()
	return args.Error(0)
}

func (c *ConsumerMock) MonitorCheck() error {
	args := c.Called()
	return args.Error(0)
}

func (c *ConsumerMock) Start(messageHandler func(message kafka.FTMessage)) {
	c.Called(messageHandler)
}

func (c *ConsumerMock) Close() error {
	c.Called()
	return nil
}

type WriterMock struct {
	mock.Mock
}

func (w *WriterMock) GetCollection(originID string, contentType string) (string, error) {
	args := w.Called(originID, contentType)
	return args.String(0), args.Error(1)
}

func (w *WriterMock) WriteToCollection(msg native.NativeMessage, collection string) (string, string, error) {
	args := w.Called(msg, collection)
	return args.String(0), args.String(1), args.Error(2)
}

func (w *WriterMock) ConnectivityCheck() (string, error) {
	args := w.Called()
	return args.String(0), args.Error(1)
}
