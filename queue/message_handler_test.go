package queue

import (
	"errors"
	"testing"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	cctOriginSystemID                  = "http://cmdb.ft.com/systems/cct"
	universalContentCollection         = "universal-content"
	contentType                        = "application/json; version=1.0"
	messageTypeHeader                  = "Message-Type"
	messageTypePartialContentPublished = "cms-partial-content-published"
)

var goodMsgHeaders = map[string]string{
	"Content-Type":      contentType,
	"X-Request-Id":      "tid_test",
	"Message-Timestamp": "2017-02-16T12:56:16Z",
	"Origin-System-Id":  cctOriginSystemID,
}

var goodMsg = kafka.FTMessage{
	Body:    "{}",
	Headers: goodMsgHeaders,
}

var badBodyMsg = kafka.FTMessage{
	Body:    "I am not JSON",
	Headers: goodMsgHeaders,
}

func init() {
	logger.InitDefaultLogger("native-ingester")
}

func TestWriteToNativeSuccessfullyWithoutForward(t *testing.T) {
	w := new(mocks.WriterMock)
	w.On("GetCollection", cctOriginSystemID, contentType).Return(universalContentCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), universalContentCollection).Return("", "", nil)

	p := new(mocks.ProducerMock)

	mh := NewMessageHandler(w, contentType)
	mh.producer = p
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeSuccessfullyWithForward(t *testing.T) {
	w := new(mocks.WriterMock)
	w.On("GetCollection", cctOriginSystemID, contentType).Return(universalContentCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), universalContentCollection).Return("", "", nil)

	p := new(mocks.ProducerMock)
	p.On("SendMessage", mock.AnythingOfType("kafka.FTMessage")).Return(nil)

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWritePartialContentToNativeSuccessfullyWithForward(t *testing.T) {
	w := new(mocks.WriterMock)
	updatedBody := "updated"
	goodMsgPartialUpdated := goodMsg
	goodMsgPartialUpdated.Headers[messageTypeHeader] = messageTypePartialContentPublished

	expectedMessage := kafka.FTMessage{
		Body:    updatedBody,
		Headers: goodMsgPartialUpdated.Headers,
	}

	w.On("GetCollection", cctOriginSystemID, contentType).Return(universalContentCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), universalContentCollection).Return("", updatedBody, nil)

	p := new(mocks.ProducerMock)
	p.On("SendMessage", mock.AnythingOfType("kafka.FTMessage")).Return(nil)

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsgPartialUpdated)

	p.AssertCalled(t, "SendMessage", expectedMessage)
	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailWithBadBodyMessage(t *testing.T) {
	w := new(mocks.WriterMock)

	p := new(mocks.ProducerMock)

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(badBodyMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailWithNotCollectionForOriginId(t *testing.T) {
	w := new(mocks.WriterMock)
	w.On("GetCollection", cctOriginSystemID, contentType).Return("", errors.New("Collection Not Found"))

	p := new(mocks.ProducerMock)

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestWriteToNativeFailBecauseOfWriter(t *testing.T) {
	w := new(mocks.WriterMock)
	w.On("GetCollection", cctOriginSystemID, contentType).Return(universalContentCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), universalContentCollection).Return("", "", errors.New("today I do not want to write"))

	p := new(mocks.ProducerMock)

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
}

func TestForwardFailBecauseOfProducer(t *testing.T) {
	hook := logger.NewTestHook("native-ingester")
	w := new(mocks.WriterMock)
	w.On("GetCollection", cctOriginSystemID, contentType).Return(universalContentCollection, nil)
	w.On("WriteToCollection", mock.AnythingOfType("native.NativeMessage"), universalContentCollection).Return("", "", nil)

	p := new(mocks.ProducerMock)
	p.On("SendMessage", mock.AnythingOfType("kafka.FTMessage")).Return(errors.New("Today, I am not writing on a queue."))

	mh := NewMessageHandler(w, contentType)
	mh.ForwardTo(p)
	mh.HandleMessage(goodMsg)

	w.AssertExpectations(t)
	p.AssertExpectations(t)
	assert.Equal(t, "error", hook.LastEntry().Level.String())
	assert.Equal(t, "Failed to forward consumed message to a different queue", hook.LastEntry().Message)
}
