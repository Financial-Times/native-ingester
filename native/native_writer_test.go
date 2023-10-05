package native

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/go-logger/v2"

	"github.com/Financial-Times/native-ingester/config"
	"github.com/Financial-Times/service-status-go/httphandlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	cctOriginSystemID              = "http://cmdb.ft.com/systems/cct"
	publishRef                     = "tid_test-pub-ref"
	aUUID                          = "572d0acc-3f12-4e70-8830-8092c1042a52"
	aTimestamp                     = "2017-02-16T12:56:16Z"
	aHash                          = "27f79e6d884acdd642d1758c4fd30d43074f8384d552d1ebb1959345"
	aContentType                   = "application/json; version=1.0"
	withNativeHashHeader           = true
	withoutNativeHashHeader        = false
	messageTypeContentPublished    = "cms-content-published"
	universalContentCollectionName = "universal-content"
)

var strCollectionsOriginIdsMap string
var audioStrCollectionsOriginIdsMap string
var sparkCollectionsOriginIdsMap string
var aContentBody map[string]interface{}

func init() {
	aContentBody = map[string]interface{}{
		"publishReference": publishRef,
		"lastModified":     aTimestamp,
	}
	audioStrCollectionsOriginIdsMap =
		`{
		   "http://cmdb.ft.com/systems/next-video-editor": [
			   {
				   "content_type": "^(application/json).*$",
				   "collection": "video"
			   },
			   {
				   "content_type": "^(application/)*(vnd.ft-upp-audio\\+json).*$",
				   "collection": "universal-content"
			   }
		   ]
   }`
	sparkCollectionsOriginIdsMap =
		`{
			"http://cmdb.ft.com/systems/spark": [
				{
					"content_type": ".*",
					"collection": "universal-content"
				}
			]
   }`
	strCollectionsOriginIdsMap = `{
		"http://cmdb.ft.com/systems/cct": [
			{
				"content_type": "(application/json).*",
				"collection": "universal-content"
			}
		]
	}`
}

func setupMockNativeWriterService(t *testing.T, status int, hasHash bool, method string, collection string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, method, req.Method)
		assert.Equal(t, "/"+collection+"/"+aUUID, req.URL.Path)
		assert.Equal(t, publishRef, req.Header.Get(transactionIDHeader))
		assert.Equal(t, aContentType, req.Header.Get(contentTypeHeader))
		if hasHash {
			assert.Equal(t, aHash, req.Header.Get(nativeHashHeader))
		}
	}))
}

func setupMockNativeWriterGTG(t *testing.T, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if status != 200 {
			w.WriteHeader(status)
		}
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, httphandlers.GTGPath, req.URL.Path)
	}))
}

func getConfig(str string) (*config.Configuration, error) {
	ior := strings.NewReader(str)
	return config.ReadConfigFromReader(ior)
}

func TestGetCollectionShort(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("", *testCollectionsOriginIdsMap, p, log)

	actualCollection, err := w.GetCollection(cctOriginSystemID, aContentType, nil)
	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, universalContentCollectionName, actualCollection, "It should return the universal-content collection")

	_, err = w.GetCollection("Origin-Id-that-do-not-exist", aContentType, nil)
	assert.EqualError(t, err, "origin system not found", "It should return a collection not found error")
	p.AssertExpectations(t)
}

func TestGetCollection(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")

	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")
	w := NewWriter("", *testCollectionsOriginIdsMap, p, log)

	tests := []struct {
		contentType   string
		expCollection string
		wantErr       bool
	}{
		{
			"application/json;version=1.0",
			"universal-content",
			false},
		{
			"application/json; version=1.0",
			"universal-content",
			false},
		{"application/json",
			"universal-content",
			false},
		{"wrong",
			"",
			true},
	}
	for _, tt := range tests {
		t.Run("Test", func(t *testing.T) {
			actualCollection, err := w.GetCollection(cctOriginSystemID, tt.contentType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetVideoCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expCollection != actualCollection {
				t.Errorf("TestGetVideoCollection() = %v, want %v", actualCollection, tt.expCollection)
			}

		})
	}
}

func TestGetAllCollection(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")

	p := new(ContentBodyParserMock)
	str := `{
			"http://cmdb.ft.com/systems/cct": [
				{
					"content_type": ".*",
					"collection": "universal-content"
				}
			]
	}`
	testCollectionsOriginIdsMap, err := getConfig(str)
	assert.NoError(t, err, "It should not return an error")
	w := NewWriter("", *testCollectionsOriginIdsMap, p, log)

	tests := []struct {
		contentType   string
		expCollection string
		wantErr       bool
	}{
		{
			"application/json;version=1.0",
			"universal-content",
			false},
		{
			"application/json; version=1.0",
			"universal-content",
			false},
		{"application/json",
			"universal-content",
			false},
		{"wrong",
			"universal-content",
			false},
	}
	for _, tt := range tests {
		t.Run("Test", func(t *testing.T) {
			actualCollection, err := w.GetCollection(cctOriginSystemID, tt.contentType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetVideoCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expCollection != actualCollection {
				t.Errorf("TestGetVideoCollection() = %v, want %v", actualCollection, tt.expCollection)
			}

		})
	}
}
func TestGetVideoCollection(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")

	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(audioStrCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")
	w := NewWriter("", *testCollectionsOriginIdsMap, p, log)
	o := "http://cmdb.ft.com/systems/next-video-editor"

	tests := []struct {
		contentType   string
		expCollection string
		wantErr       bool
	}{
		{
			"application/vnd.ft-upp-audio+json;version=1.0",
			"universal-content",
			false},
		{
			"application/vnd.ft-upp-audio+json",
			"universal-content",
			false},
		{
			"application/vnd.ft-upp-audio+json;",
			"universal-content",
			false},
		{
			"application/vnd.ft-upp-audio+json;version=1.0",
			"universal-content",
			false},
		{
			"vnd.ft-upp-audio+json",
			"universal-content",
			false},
		{"application/json",
			"video",
			false},
		{"wrong",
			"",
			true},
	}
	for _, tt := range tests {
		t.Run("Test", func(t *testing.T) {
			actualCollection, err := w.GetCollection(o, tt.contentType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestGetVideoCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expCollection != actualCollection {
				t.Errorf("TestGetVideoCollection() = %v, want %v", actualCollection, tt.expCollection)
			}

		})
	}
}

func TestWriteMessageToCollectionWithSuccess(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")
	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withoutNativeHashHeader, "POST", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	msg.AddContentTypeHeader(aContentType)
	assert.NoError(t, err, "It should not return an error by creating a message")

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	contentUUID, _, err := w.WriteToCollection(msg, universalContentCollectionName)

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWritePartialMessageToCollectionWithSuccess(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(sparkCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")
	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withoutNativeHashHeader, "PATCH", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypePartialContentPublished)
	msg.AddContentTypeHeader(aContentType)
	assert.NoError(t, err, "It should not return an error by creating a message")

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	contentUUID, _, err := w.WriteToCollection(msg, universalContentCollectionName)

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteMessageWithHashToCollectionWithSuccess(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader, "POST", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	contentUUID, _, err := w.WriteToCollection(msg, universalContentCollectionName)

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteMessageToCollectionWithContentTypeSuccess(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader, "POST", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	contentUUID, _, err := w.WriteToCollection(msg, universalContentCollectionName)

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, aUUID, contentUUID)
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfMissingUUID(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return("", errors.New("UUID not found"))
	nws := setupMockNativeWriterService(t, 200, withNativeHashHeader, "POST", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	_, _, err = w.WriteToCollection(msg, universalContentCollectionName)

	assert.EqualError(t, err, "UUID not found", "It should return a  UUID not found error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceInternalError(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)
	nws := setupMockNativeWriterService(t, 500, withoutNativeHashHeader, "POST", universalContentCollectionName)
	defer nws.Close()

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	_, _, err = w.WriteToCollection(msg, universalContentCollectionName)

	assert.EqualError(t, err, "Native writer returned non-200 code", "It should return a non-200 HTTP status error")
	p.AssertExpectations(t)
}

func TestWriteContentBodyToCollectionFailBecauseOfNativeRWServiceNotAvailable(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	p.On("getUUID", aContentBody).Return(aUUID, nil)

	msg, err := NewNativeMessage("{}", aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should not return an error by creating a message")
	msg.AddHashHeader(aHash)
	msg.AddContentTypeHeader(aContentType)

	w := NewWriter("http://an-address.com", *testCollectionsOriginIdsMap, p, log)
	_, _, err = w.WriteToCollection(msg, universalContentCollectionName)

	assert.Error(t, err, "It should return an error")
	p.AssertExpectations(t)
}

func TestConnectivityCheckSuccess(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 200)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	msg, err := w.ConnectivityCheck()

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "Native writer is good to go.", msg, "It should return a positive message")
}

func TestConnectivityCheckSuccessWithoutHostHeader(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 200)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	msg, err := w.ConnectivityCheck()

	assert.NoError(t, err, "It should not return an error")
	assert.Equal(t, "Native writer is good to go.", msg, "It should return a positive message")
}

func TestConnectivityCheckFailNotGTG(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	nws := setupMockNativeWriterGTG(t, 503)

	w := NewWriter(nws.URL, *testCollectionsOriginIdsMap, p, log)
	msg, err := w.ConnectivityCheck()

	assert.EqualError(t, err, "GTG HTTP status code is 503", "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailNativeRWServiceNotAvailable(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("http://an-address.com", *testCollectionsOriginIdsMap, p, log)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Native writer is not good to go.", msg, "It should return a negative message")
}

func TestConnectivityCheckFailToBuildRequest(t *testing.T) {
	log := logger.NewUPPLogger("native_writer_test", "DEBUG")
	p := new(ContentBodyParserMock)
	testCollectionsOriginIdsMap, err := getConfig(strCollectionsOriginIdsMap)
	assert.NoError(t, err, "It should not return an error")

	w := NewWriter("http://foo.com  and some spaces", *testCollectionsOriginIdsMap, p, log)
	msg, err := w.ConnectivityCheck()

	assert.Error(t, err, "It should return an error")
	assert.Equal(t, "Error in building request to check if the native writer is good to go", msg, "It should return a negative message")
}

type ContentBodyParserMock struct {
	mock.Mock
}

func (p *ContentBodyParserMock) getUUID(body map[string]interface{}) (string, error) {
	args := p.Called(body)
	return args.String(0), args.Error(1)
}

func TestBuildNativeMessageSuccess(t *testing.T) {
	msg, err := NewNativeMessage(`{"foo":"bar"}`, aTimestamp, publishRef, messageTypeContentPublished)
	assert.NoError(t, err, "It should return an error in creating a new message")
	msg.AddHashHeader(aHash)

	assert.Equal(t, msg.body["foo"], "bar", "The body should contain the original attributes")
	assert.Equal(t, msg.body["lastModified"], aTimestamp, "The body should contain the additional timestamp")
	assert.Equal(t, msg.body["publishReference"], publishRef, "The body should contain the publish reference")
	assert.Equal(t, msg.headers[nativeHashHeader], aHash, "The message should contain the hash")
	assert.Equal(t, msg.TransactionID(), publishRef, "The message should contain the publish reference")
}

func TestBuildNativeMessageFailure(t *testing.T) {
	_, err := NewNativeMessage("__INVALID_BODY__", aTimestamp, publishRef, messageTypeContentPublished)
	assert.EqualError(t, err, "invalid character '_' looking for beginning of value", "It should return an error in creating a new message")
}
