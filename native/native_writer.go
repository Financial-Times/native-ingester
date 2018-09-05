package native

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/native-ingester/config"
	"github.com/Financial-Times/service-status-go/httphandlers"
)

const nativeHashHeader = "X-Native-Hash"
const contentTypeHeader = "Content-Type"
const transactionIDHeader = "X-Request-Id"

// Writer provides the functionalities to write in the native store
type Writer interface {
	GetCollection(originID string, contentType string) (string, error)
	WriteToCollection(msg NativeMessage, collection string) (string, error)
	ConnectivityCheck() (string, error)
}

type nativeWriter struct {
	address     string
	collections config.Configuration
	httpClient  http.Client
	bodyParser  ContentBodyParser
}

// NewWriter returns a new instance of a native writer
func NewWriter(address string, collectionsOriginIdsMap config.Configuration, parser ContentBodyParser) Writer {
	collections := collectionsOriginIdsMap
	return &nativeWriter{address, collections, http.Client{}, parser}
}

func (nw *nativeWriter) GetCollection(originID string, contentType string) (string, error) {
	return nw.collections.GetCollection(originID, contentType)
}

func (nw *nativeWriter) WriteToCollection(msg NativeMessage, collection string) (string, error) {
	contentUUID, err := nw.bodyParser.getUUID(msg.body)
	if err != nil {
		logger.NewEntry(msg.TransactionID()).WithError(err).Error("Error extracting uuid. Ignoring message.")
		return contentUUID, err
	}
	logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).Info("Start processing native publish event")
	cBodyAsJSON, err := json.Marshal(msg.body)

	if err != nil {
		logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).WithError(err).Error("Error marshalling message")
		return contentUUID, err
	}

	requestURL := nw.address + "/" + collection + "/" + contentUUID
	request, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(cBodyAsJSON))
	if err != nil {
		logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).WithError(err).Error("Error calling native writer. Ignoring message.")
		return contentUUID, err
	}

	for header, value := range msg.headers {
		request.Header.Set(header, value)
	}

	if request.Header.Get(contentTypeHeader) == "" {
		logger.NewEntry(msg.TransactionID()).
			WithUUID(contentUUID).
			Warn("Native-save request does not have content-type header, defaulting to application/json.")

		request.Header.Set("Content-Type", "application/json")

	}

	response, err := nw.httpClient.Do(request)

	if err != nil {
		logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).WithError(err).Error("Error calling native writer. Ignoring message.")
		return contentUUID, err
	}
	defer properClose(msg.TransactionID(), response)

	if isNot2XXStatusCode(response.StatusCode) {
		errMsg := "Native writer returned non-200 code"
		err := errors.New(errMsg)
		logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).WithError(err).Error(errMsg)
		return contentUUID, err
	}

	logger.NewEntry(msg.TransactionID()).WithUUID(contentUUID).Info("Successfully finished processing native publish event")
	return contentUUID, nil
}

func properClose(tid string, resp *http.Response) {
	_, err := io.Copy(ioutil.Discard, resp.Body)
	if err != nil {
		logger.NewEntry(tid).WithError(err).Warn("Couldn't read response body")
	}
	err = resp.Body.Close()
	if err != nil {
		logger.NewEntry(tid).WithError(err).Warn("Couldn't close response body")
	}
}

func isNot2XXStatusCode(statusCode int) bool {
	return statusCode < 200 || statusCode >= 300
}

func (nw nativeWriter) ConnectivityCheck() (string, error) {
	req, err := http.NewRequest("GET", nw.address+httphandlers.GTGPath, nil)
	if err != nil {
		return "Error in building request to check if the native writer is good to go", err
	}
	resp, err := nw.httpClient.Do(req)
	if err != nil {
		return "Native writer is not good to go.", err
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return "Native writer is not good to go.", fmt.Errorf("GTG HTTP status code is %v", resp.StatusCode)
	}
	return "Native writer is good to go.", nil
}

// NativeMessage is the message accepted by the native writer
type NativeMessage struct {
	body    map[string]interface{}
	headers map[string]string
}

// NewNativeMessage returns a new instance of a NativeMessage
func NewNativeMessage(contentBody string, timestamp string, transactionID string) (NativeMessage, error) {
	body := make(map[string]interface{})
	if err := json.Unmarshal([]byte(contentBody), &body); err != nil {
		return NativeMessage{}, err
	}

	body["lastModified"] = timestamp
	body["publishReference"] = transactionID

	msg := NativeMessage{body, make(map[string]string)}
	msg.headers[transactionIDHeader] = transactionID

	return msg, nil
}

//AddHashHeader adds the hash of the native content as a header
func (msg *NativeMessage) AddHashHeader(hash string) {
	msg.headers[nativeHashHeader] = hash
}

//AddContentTypeHeader adds the content type of the native content as a header
func (msg *NativeMessage) AddContentTypeHeader(hash string) {
	msg.headers[contentTypeHeader] = hash
}

func (msg *NativeMessage) TransactionID() string {
	return msg.headers[transactionIDHeader]
}

func (msg *NativeMessage) ContentType() string {
	return msg.headers[contentTypeHeader]
}
