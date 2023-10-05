package queue

import (
	"fmt"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/native"
)

// MessageHandler handles messages consumed from a queue
type MessageHandler struct {
	writer      native.Writer
	producer    kafkaProducer
	forwards    bool
	contentType string
	logger      *logger.UPPLogger
}

type kafkaProducer interface {
	SendMessage(message kafka.FTMessage) error
	Close() error
}

// NewMessageHandler returns a new instance of MessageHandler
func NewMessageHandler(w native.Writer, contentType string, logger *logger.UPPLogger) *MessageHandler {
	return &MessageHandler{writer: w, contentType: contentType, logger: logger}
}

// HandleMessage implements the strategy for handling message from a queue
func (mh *MessageHandler) HandleMessage(msg kafka.FTMessage) {
	pubEvent := publicationEvent{msg}

	mh.logger.WithTransactionID(pubEvent.transactionID()).WithField("Content-Type", pubEvent.contentType()).Infof("Handling new message with headers: %v", pubEvent.Headers)

	logMonitoringEvent := mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType)

	writerMsg, err := pubEvent.nativeMessage(mh.logger)
	if err != nil {
		logMonitoringEvent.
			WithError(err).
			Error("Error unmarshalling content body from publication event. Ignoring message.")
		return
	}

	collection, err := mh.writer.GetCollection(pubEvent.originSystemID(), writerMsg.ContentType(), writerMsg.Publication())
	if err != nil {
		logMonitoringEvent.
			WithValidFlag(false).
			Warn(fmt.Sprintf("Skipping content because of not whitelisted combination (Origin-System-Id, Content-Type): (%s, %s)", pubEvent.originSystemID(), writerMsg.ContentType()))
		return
	}

	contentUUID, updatedContent, writerErr := mh.writer.WriteToCollection(writerMsg, collection)
	if writerErr != nil {
		logMonitoringEvent.
			WithError(writerErr).
			Error("Failed to write native content")
		return
	}

	if mh.forwards {

		if writerMsg.IsPartialContent() {
			pubEvent.Body = updatedContent
		}

		mh.logger.WithTransactionID(pubEvent.transactionID()).Info("Forwarding consumed message to different queue")
		forwardErr := mh.producer.SendMessage(pubEvent.producerMsg())
		if forwardErr != nil {
			logMonitoringEvent.
				WithUUID(contentUUID).
				WithError(forwardErr).
				Error("Failed to forward consumed message to a different queue")
			return
		}
		logMonitoringEvent.
			WithUUID(contentUUID).
			Info("Successfully ingested")
	}

}

// ForwardTo sets up the message producer to forward messages after writing in the native store
func (mh *MessageHandler) ForwardTo(p kafkaProducer) {
	mh.producer = p
	mh.forwards = true
}
