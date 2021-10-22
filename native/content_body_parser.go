package native

import (
	"errors"
	"strings"

	uuidParser "github.com/google/uuid"
	"github.com/jmoiron/jsonq"
)

// ContentBodyParser parses the body of native content
type ContentBodyParser interface {
	getUUID(body map[string]interface{}) (string, error)
}

type contentBodyParser struct {
	uuidJSONPaths []string
}

// NewContentBodyParser returns a new instace of a ContentBodyParser
func NewContentBodyParser(uuidJSONPaths []string) ContentBodyParser {
	return &contentBodyParser{uuidJSONPaths}
}

func (p contentBodyParser) getUUID(body map[string]interface{}) (string, error) {
	jq := jsonq.NewQuery(body)
	for _, uuidPath := range p.uuidJSONPaths {
		uuid, jsonPathErr := jq.String(strings.Split(uuidPath, ".")...)
		_, uuidParsingError := uuidParser.Parse(uuid)
		if jsonPathErr == nil && uuidParsingError == nil {
			return uuid, nil
		}
	}
	return "", errors.New("UUID not found")
}
