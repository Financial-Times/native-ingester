package queue

import (
	"errors"
	"strings"

	"github.com/Financial-Times/go-logger/v2"
	"github.com/Financial-Times/kafka-client-go/v4"
	"github.com/Financial-Times/native-ingester/native"
)

type publicationEvent struct {
	kafka.FTMessage
}

func (pe *publicationEvent) transactionID() string {
	return pe.Headers["X-Request-Id"]
}

func (pe *publicationEvent) originSystemID() string {
	return strings.TrimSpace(pe.Headers["Origin-System-Id"])
}

func (pe *publicationEvent) contentType() string {
	return strings.TrimSpace(pe.Headers["Content-Type"])
}

func (pe *publicationEvent) messageType() string {
	return strings.TrimSpace(pe.Headers["Message-Type"])
}

// nativeMessage given a kafka message, extracts useful headers and body to adds them into a new NativeMessage struct.
func (pe *publicationEvent) nativeMessage(log *logger.UPPLogger) (native.NativeMessage, error) {
	timestamp, found := pe.Headers["Message-Timestamp"]
	if !found {
		return native.NativeMessage{}, errors.New("publish event does not contain timestamp")
	}

	msg, err := native.NewNativeMessage(pe.Body, timestamp, pe.transactionID(), pe.messageType())

	if err != nil {
		return native.NativeMessage{}, err
	}

	nativeHash, found := pe.Headers["Native-Hash"]
	if found {
		msg.AddHashHeader(nativeHash)
	}

	contentType, found := pe.Headers["Content-Type"]
	if found {
		msg.AddContentTypeHeader(contentType)
	}
	originSystemID, found := pe.Headers["Origin-System-Id"]
	if found {
		msg.AddOriginSystemIDHeader(originSystemID)
	}
	schemaVersion, found := pe.Headers["X-Schema-Version"]
	if found {
		msg.AddSchemaVersion(schemaVersion)
	}
	contentRevision, found := pe.Headers["X-Content-Revision"]
	if found {
		msg.AddContentRevision(contentRevision)
	}
	log.WithTransactionID(pe.transactionID()).
		WithField("Content-Type", msg.ContentType()).
		WithField("Origin-System-Id", msg.OriginSystemID()).
		WithField("X-Content-Revision", msg.ContentRevision()).
		WithField("X-Schema-Version", msg.SchemaVersion()).
		Infof("Constructed new NativeMessage")

	return msg, nil
}

func (pe *publicationEvent) producerMsg() kafka.FTMessage {
	return kafka.FTMessage{
		Headers: pe.Headers,
		Body:    pe.Body,
	}
}
