package queue

import (
	"fmt"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/kafka"
	"github.com/Financial-Times/native-ingester/native"
)

// MessageHandler handles messages consumed from a queue
type MessageHandler struct {
	writer      native.Writer
	producer    kafka.Producer
	forwards    bool
	contentType string
	logger      *logger.UPPLogger
}

// NewMessageHandler returns a new instance of MessageHandler
func NewMessageHandler(w native.Writer, contentType string, logger *logger.UPPLogger) *MessageHandler {
	return &MessageHandler{writer: w, contentType: contentType, logger: logger}
}

// HandleMessage implements the strategy for handling message from a queue
func (mh *MessageHandler) HandleMessage(msg kafka.FTMessage) error {
	pubEvent := publicationEvent{msg}

	mh.logger.WithTransactionID(pubEvent.transactionID()).WithField("Content-Type", pubEvent.contentType()).Infof("Handling new message with headers: %v", pubEvent.Headers)

	writerMsg, err := pubEvent.nativeMessage(mh.logger)
	if err != nil {
		mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType).
			WithError(err).
			Error("Error unmarshalling content body from publication event. Ignoring message.")
		return err
	}

	collection, err := mh.writer.GetCollection(pubEvent.originSystemID(), writerMsg.ContentType())
	if err != nil {
		mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType).
			WithValidFlag(false).
			Warn(fmt.Sprintf("Skipping content because of not whitelisted combination (Origin-System-Id, Content-Type): (%s, %s)", pubEvent.originSystemID(), writerMsg.ContentType()))
		return nil
	}

	contentUUID, updatedContent, writerErr := mh.writer.WriteToCollection(writerMsg, collection)
	if writerErr != nil {
		mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType).
			WithError(writerErr).
			Error("Failed to write native content")
		return err
	}

	if mh.forwards {

		if writerMsg.IsPartialContent() {
			pubEvent.Body = updatedContent
		}

		mh.logger.WithTransactionID(pubEvent.transactionID()).Info("Forwarding consumed message to different queue")
		forwardErr := mh.producer.SendMessage(pubEvent.producerMsg())
		if forwardErr != nil {
			mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType).
				WithUUID(contentUUID).
				WithError(forwardErr).
				Error("Failed to forward consumed message to a different queue")
			return forwardErr
		}
		mh.logger.WithMonitoringEvent("Ingest", pubEvent.transactionID(), mh.contentType).
			WithUUID(contentUUID).
			Info("Successfully ingested")
	}

	return nil
}

// ForwardTo sets up the message producer to forward messages after writing in the native store
func (mh *MessageHandler) ForwardTo(p kafka.Producer) {
	mh.producer = p
	mh.forwards = true
}
