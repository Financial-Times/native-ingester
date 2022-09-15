package queue

import (
	"testing"

	"github.com/Financial-Times/go-logger/v2"

	"github.com/Financial-Times/kafka-client-go/v3"
	"github.com/stretchr/testify/assert"
)

const expectedTID = "tid_test"
const expectedOriginSystemID = "http://cmdb.ft.com/systems/cct"
const expectedTimestamp = "2017-02-16T12:56:16Z"
const expectedHash = "27f79e6d884acdd642d1758c4fd30d43074f8384d552d1ebb1959345"

var someMsgHeaders = map[string]string{
	"X-Request-Id":      expectedTID,
	"Origin-System-Id":  expectedOriginSystemID,
	"Message-Timestamp": expectedTimestamp,
	"Native-Hash":       expectedHash,
}

var aMsg = kafka.FTMessage{
	Headers: someMsgHeaders,
	Body:    `{"foo":"bar"}`,
}

var aMsgWithBadBody = kafka.FTMessage{
	Headers: someMsgHeaders,
	Body:    `I'm not JSON`,
}

var aMsgWithoutTimestamp = kafka.FTMessage{
	Headers: map[string]string{},
	Body:    `{"foo":"bar"}`,
}

func TestGetTransactionID(t *testing.T) {
	pe := publicationEvent{aMsg}
	actualTID := pe.transactionID()
	assert.Equal(t, expectedTID, actualTID, "The transaction ID shoud be the same of a consumer message")
}

func TestGetOriginSystemID(t *testing.T) {
	pe := publicationEvent{aMsg}
	actualOriginSystemID := pe.originSystemID()
	assert.Equal(t, expectedOriginSystemID, actualOriginSystemID, "The Origin-System-Id shoud be the same of a consumer message")
}

func TestGetNativeMessageSuccessfully(t *testing.T) {
	log := logger.NewUnstructuredLogger()
	pe := publicationEvent{aMsg}
	_, err := pe.nativeMessage(log)

	assert.NoError(t, err, "It should not return an error")
}

func TestGetNativeMessageFailBecauseBadBody(t *testing.T) {
	log := logger.NewUnstructuredLogger()
	pe := publicationEvent{aMsgWithBadBody}
	_, err := pe.nativeMessage(log)

	assert.EqualError(t, err, "invalid character 'I' looking for beginning of value", "It should return an error")
}

func TestGetCNativeNativeMessageFailBecauseMissingTimstamp(t *testing.T) {
	log := logger.NewUnstructuredLogger()
	pe := publicationEvent{aMsgWithoutTimestamp}
	_, err := pe.nativeMessage(log)

	assert.EqualError(t, err, "publish event does not contain timestamp", "It should return an error")
}

func TestGetProducerMessage(t *testing.T) {
	pe := publicationEvent{aMsg}
	actualProducerMsg := pe.producerMsg()

	assert.Equal(t, aMsg.Body, actualProducerMsg.Body, "It should have the same body of the consumer message")
	assert.Equal(t, aMsg.Headers, actualProducerMsg.Headers, "It should have the same headers of the consumer message")
}
