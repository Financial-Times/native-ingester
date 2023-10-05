package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"slices"
)

type OriginSystemConfig struct {
	ContentType       string   `json:"content_type,binding:required"`
	Publication       []string `json:"publication"`
	Collection        string   `json:"collection,binding:required"`
	contentTypeRegexp *regexp.Regexp
}

// Configuration data
type Configuration struct {
	Config map[string][]OriginSystemConfig
}

func (c *Configuration) validateConfig() error {
	for oKey, origCollection := range c.Config {
		for ocKey, val := range origCollection {
			if val.ContentType == "" {
				return errors.New("contentType value is mandatory")
			}
			if val.Collection == "" {
				return errors.New("collection value is mandatory")
			}
			c.Config[oKey][ocKey].contentTypeRegexp = regexp.MustCompile(val.ContentType)
		}
	}
	return nil
}

func (c *Configuration) GetCollection(originID string, contentType string, publication []interface{}) (string, error) {
	collection := c.Config[originID]
	if len(collection) == 0 {
		return "", errors.New("origin system not found")
	}
	for _, val := range collection {
		if val.contentTypeRegexp.MatchString(contentType) && publicationMatch(publication, val) {
			return val.Collection, nil
		}
	}
	return "", errors.New("origin system, content type and publication not configured")
}

// ReadConfigFromReader reads config as a json stream from the given reader
func ReadConfigFromReader(r io.Reader) (c *Configuration, e error) {
	c = new(Configuration)
	decoder := json.NewDecoder(r)
	e = decoder.Decode(&c.Config)
	if e != nil {
		return nil, e
	}
	return c, c.validateConfig()
}

// ReadConfig reads config as a json file from the given path
func ReadConfig(confPath string) (c *Configuration, e error) {
	file, fErr := os.Open(confPath)
	if fErr != nil {
		return nil, fErr
	}
	defer file.Close()
	return ReadConfigFromReader(file)
}

func publicationMatch(publication []interface{}, config OriginSystemConfig) bool {
	if len(config.Publication) == 0 {
		return true
	}

	for _, pub := range publication {
		if slices.Contains(config.Publication, pub.(string)) {
			return true
		}
	}
	return false
}
