package config

import (
	"errors"
	"sort"
	"strings"
	"testing"
)

func toString(c *Configuration) string {
	if c == nil {
		return ""
	}
	var str string
	var keys []string
	for key := range c.Config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		origCollection := c.Config[key]
		str += key
		for _, val := range origCollection {
			str += val.ContentType + val.Collection
		}
	}
	return str
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		c    *Configuration
		err  error
	}{
		{
			"config Ok",
			&Configuration{
				Config: map[string][]OriginSystemConfig{
					"http://cmdb.ft.com/systems/cct": {
						{ContentType: ".*",
							Collection: "universal-content",
						},
					},
				},
			},
			nil,
		},
		{
			"Empty ContentType",
			&Configuration{
				Config: map[string][]OriginSystemConfig{
					"http://cmdb.ft.com/systems/cct": {
						{ContentType: "",
							Collection: "universal-content",
						},
					},
				},
			},
			errors.New("contentType value is mandatory"),
		},
		{
			"Empty Collection",
			&Configuration{
				Config: map[string][]OriginSystemConfig{
					"http://cmdb.ft.com/systems/cct": {
						{ContentType: "-",
							Collection: "",
						},
					},
				},
			},
			errors.New("collection value is mandatory"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.c.validateConfig()
			if err != tt.err && err.Error() != tt.err.Error() {
				t.Errorf("Configuration.validateConfig() error = %v, wantErr %v", err, tt.err)
				return
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	tests := []struct {
		name     string
		confText string
		wantC    *Configuration
		wantErr  bool
	}{
		{
			"Test1",
			`{
				"http://cmdb.ft.com/systems/cct": [
						{
							"content_type": ".*",
							"collection": "universal-content"
						}
					],
			   "http://cmdb.ft.com/systems/next-video-editor": [
						{
							"content_type": "application/json",
							"collection": "video"
						},
						{
							"content_type": "^(application/)*(vnd.ft-upp-audio\\+json).*$",
							"collection": "audio"
						}
					]	
			}`,
			&Configuration{
				Config: map[string][]OriginSystemConfig{
					"http://cmdb.ft.com/systems/cct": {
						{ContentType: ".*",
							Collection: "universal-content",
						},
					},
					"http://cmdb.ft.com/systems/next-video-editor": {
						{ContentType: "application/json",
							Collection: "video",
						},
						{ContentType: "^(application/)*(vnd.ft-upp-audio\\+json).*$",
							Collection: "audio",
						},
					},
				},
			},
			false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotC, err := ReadConfigFromReader(strings.NewReader(tt.confText))
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if toString(gotC) != toString(tt.wantC) {
				t.Errorf("ReadConfig() = %v, want %v", toString(gotC), toString(tt.wantC))
			}
		})
	}
}

func TestConfiguration_GetCollection(t *testing.T) {
	type args struct {
		originID    string
		contentType string
	}
	c := &Configuration{
		Config: map[string][]OriginSystemConfig{
			"http://cmdb.ft.com/systems/next-video-editor": {
				//{ContentType: "^(application/json).*$",
				{ContentType: "application/json",
					Collection: "video",
				},
				{ContentType: "^(application/)*(vnd.ft-upp-audio\\+json).*$",
					Collection: "audio",
				},
			},
			"http://cmdb.ft.com/systems/cct": {
				{
					ContentType: "^(application/)*(vnd.ft-upp-page).*$",
					Collection:  "pages",
				},
				{
					ContentType: "^(application/)*(vnd.ft-upp-content-relation\\+json).*$",
					Collection:  "content-relation",
				},
				{ContentType: ".*",
					Collection: "universal-content",
				},
			},
			"http://cmdb.ft.com/systems/spark": {
				{
					ContentType: "^(application/)*(vnd.ft-upp-content-relation\\+json).*$",
					Collection:  "content-relation",
				},
				{
					ContentType: ".*",
					Collection:  "universal-content",
				},
			},
			"http://cmdb.ft.com/systems/spark-lists": {
				{ContentType: "^(application/)*(vnd.ft-upp-list\\+json).*$",
					Collection: "universal-content",
				},
				{
					ContentType: "^(application/)*(vnd.ft-upp-page).*$",
					Collection:  "pages",
				},
			},
			"http://cmdb.ft.com/systems/cle": {
				{
					ContentType: "^(application/)*(vnd.ft-upp-live-event).*$",
					Collection:  "universal-content",
				},
			},
			"http://cmdb.ft.com/systems/spark-clips": {
				{
					ContentType: "^(application/)*(vnd.ft-upp-clip\\+json).*$",
					Collection:  "universal-content",
				},
				{
					ContentType: "^(application/)*(vnd.ft-upp-clip-set\\+json).*$",
					Collection:  "universal-content",
				},
			},
		},
	}
	err := c.validateConfig()
	if err != nil {
		t.Errorf("Configuration.GetCollection() error = %v", err)
		return
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"video wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"anytype"},
			"",
			true,
		},
		{
			"video OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json"},
			"video",
			false,
		},
		{
			"video OK long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json; utf8"},
			"video",
			false,
		},
		{
			"audio OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json"},
			"audio",
			false,
		},
		{
			"audio long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json;UTF8"},
			"audio",
			false,
		},
		{
			"audio wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio-json"},
			"",
			true,
		},
		{
			"wrong origin",
			args{"http://cmdb.ft.com/systems/next",
				"application/vnd.ft-upp-audio+json"},
			"",
			true,
		},
		{
			"cct article OK",
			args{"http://cmdb.ft.com/systems/cct",
				"application/vnd.ft-upp-article+json"},
			"universal-content",
			false,
		},
		{
			"cct list OK",
			args{"http://cmdb.ft.com/systems/cct",
				"application/vnd.ft-upp-list+json"},
			"universal-content",
			false,
		},
		{
			"cct page OK",
			args{
				originID:    "http://cmdb.ft.com/systems/cct",
				contentType: "application/vnd.ft-upp-page+json",
			},
			"pages",
			false,
		},
		{
			"spark article OK",
			args{"http://cmdb.ft.com/systems/spark",
				"application/vnd.ft-upp-article+json"},
			"universal-content",
			false,
		},
		{
			"spark list OK",
			args{"http://cmdb.ft.com/systems/spark",
				"application/vnd.ft-upp-list+json"},
			"universal-content",
			false,
		},
		{
			"spark page OK",
			args{
				originID:    "http://cmdb.ft.com/systems/spark",
				contentType: "application/vnd-ft.upp-page+json",
			},
			"universal-content",
			false,
		},
		{
			"spark-lists list OK",
			args{"http://cmdb.ft.com/systems/spark-lists",
				"application/vnd.ft-upp-list+json"},
			"universal-content",
			false,
		},
		{
			"spark-lists wrong CT",
			args{"http://cmdb.ft.com/systems/spark-lists",
				"application/vnd.ft-upp-article+json"},
			"",
			true,
		},
		{
			"spark-lists page OK",
			args{
				originID:    "http://cmdb.ft.com/systems/spark-lists",
				contentType: "application/vnd.ft-upp-page+json",
			},
			"pages",
			false,
		},
		{
			"Community live event OK",
			args{"http://cmdb.ft.com/systems/cle",
				"application/vnd.ft-upp-live-event+json"},
			"universal-content",
			false,
		},
		{
			"Community live event wrong CT",
			args{"http://cmdb.ft.com/systems/cle",
				"application/vnd.ft-upp-list+json"},
			"",
			true,
		},
		{
			"Community live event wrong origin",
			args{
				originID:    "http://cmdb.ft.com/systems/community-event",
				contentType: "application/vnd-ft.upp-live-event+json",
			},
			"",
			true,
		},
		{
			"Clips OK",
			args{
				originID:    "http://cmdb.ft.com/systems/spark-clips",
				contentType: "application/vnd.ft-upp-clip+json",
			},
			"universal-content",
			false,
		},
		{
			"Clips wrong origin",
			args{
				originID:    "http://cmdb.ft.com/systems/community-event",
				contentType: "application/vnd.ft-upp-clip+json",
			},
			"",
			true,
		},
		{
			"Clips wrong CT",
			args{
				originID:    "http://cmdb.ft.com/systems/spark-clips",
				contentType: "application/vnd.ft-upp-other+json",
			},
			"",
			true,
		},
		{
			"Clipset OK",
			args{
				originID:    "http://cmdb.ft.com/systems/spark-clips",
				contentType: "application/vnd.ft-upp-clip-set+json",
			},
			"universal-content",
			false,
		},
		{
			"Clipset wrong origin",
			args{
				originID:    "http://cmdb.ft.com/systems/community-event",
				contentType: "application/vnd.ft-upp-clip-set+json",
			},
			"",
			true,
		},
		{
			"ContentRelation OK",
			args{"http://cmdb.ft.com/systems/spark",
				"application/vnd.ft-upp-content-relation+json"},
			"content-relation",
			false,
		},
		{
			"ContentRelation OK",
			args{"http://cmdb.ft.com/systems/cct",
				"application/vnd.ft-upp-content-relation+json"},
			"content-relation",
			false,
		},
		{
			"ContentRelation wrong origin",
			args{"http://cmdb.ft.com/systems/community-event",
				"application/vnd.ft-upp-content-relation+json"},
			"",
			true,
		},
		{
			"Custom Code Components OK spark",
			args{
				originID:    "http://cmdb.ft.com/systems/spark",
				contentType: "application/vnd.ft-upp-custom-code-component+json",
			},
			"universal-content",
			false,
		},
		{
			"Custom Code Components OK cct",
			args{
				originID:    "http://cmdb.ft.com/systems/cct",
				contentType: "application/vnd.ft-upp-custom-code-component+json",
			},
			"universal-content",
			false,
		},
		{
			"Custom Code Components wrong origin",
			args{
				originID:    "http://cmdb.ft.com/systems/community-event",
				contentType: "application/vnd.ft-upp-custom-code-component+json",
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := c.GetCollection(tt.args.originID, tt.args.contentType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.GetCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Configuration.GetCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigurationMetadata_GetCollection(t *testing.T) {
	type args struct {
		originID    string
		contentType string
		publication []interface{}
	}
	c := &Configuration{
		Config: map[string][]OriginSystemConfig{
			"http://cmdb.ft.com/systems/ft-pink-annotations": {
				{ContentType: ".*",
					Collection: "pac-metadata",
				},
			},
			"http://cmdb.ft.com/systems/next-video-editor": {
				{ContentType: "application/json",
					Collection: "video-metadata",
				},
			},
			"http://cmdb.ft.com/systems/cct": {
				{ContentType: ".*",
					Publication: []string{"8e6c705e-1132-42a2-8db0-c295e29e8658", "19d50190-8656-4e91-8d34-82e646ada9c9"},
					Collection:  "external-metadata",
				},
			},
			"http://cmdb.ft.com/systems/spark": {
				{ContentType: ".*",
					Publication: []string{"8e6c705e-1132-42a2-8db0-c295e29e8658", "19d50190-8656-4e91-8d34-82e646ada9c9"},
					Collection:  "external-metadata",
				},
			},
		},
	}
	err := c.validateConfig()
	if err != nil {
		t.Errorf("Configuration.GetCollection() error = %v", err)
		return
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"ft pink json",
			args{"http://cmdb.ft.com/systems/ft-pink-annotations",
				"application/json",
				nil},
			"pac-metadata",
			false,
		},
		{
			"ft pink null CT",
			args{"http://cmdb.ft.com/systems/ft-pink-annotations",
				"",
				nil},
			"pac-metadata",
			false,
		},
		{
			"ft pink",
			args{"http://cmdb.ft.com/systems/ft-pink-annotations",
				"anytype",
				nil},
			"pac-metadata",
			false,
		},
		{
			"video wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"anytype",
				nil},
			"",
			true,
		},
		{
			"video OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json",
				nil},
			"video-metadata",
			false,
		},
		{
			"video OK long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/json; utf8",
				nil},
			"video-metadata",
			false,
		},
		{
			"audio OK",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json",
				nil},
			"",
			true,
		},
		{
			"audio long CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio+json;UTF8",
				nil},
			"",
			true,
		},
		{
			"audio wrong CT",
			args{"http://cmdb.ft.com/systems/next-video-editor",
				"application/vnd.ft-upp-audio-json",
				nil},
			"",
			true,
		},
		{
			"wrong origin",
			args{"http://cmdb.ft.com/systems/next",
				"application/vnd.ft-upp-audio+json",
				nil},
			"",
			true,
		},
		{
			"sustainable views ok",
			args{"http://cmdb.ft.com/systems/cct",
				"",
				[]interface{}{"8e6c705e-1132-42a2-8db0-c295e29e8658"}},
			"external-metadata",
			false,
		},
		{
			"sustainable views wrong publication",
			args{"http://cmdb.ft.com/systems/cct",
				"",
				[]interface{}{"8e6c705e-1132-42a2-8db0-c295e29e8659"}},
			"",
			true,
		},
		{
			"FTA ok",
			args{"http://cmdb.ft.com/systems/spark",
				"",
				[]interface{}{"19d50190-8656-4e91-8d34-82e646ada9c9"}},
			"external-metadata",
			false,
		},
		{
			"FTA ok with cct origin",
			args{"http://cmdb.ft.com/systems/cct",
				"",
				[]interface{}{"19d50190-8656-4e91-8d34-82e646ada9c9"}},
			"external-metadata",
			false,
		},
		{
			"fta wrong publication",
			args{"http://cmdb.ft.com/systems/spark",
				"",
				[]interface{}{"116c705e-1132-42a2-8db0-c295e29e8611"}},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := c.GetCollection(tt.args.originID, tt.args.contentType, tt.args.publication)
			if (err != nil) != tt.wantErr {
				t.Errorf("Configuration.GetCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Configuration.GetCollection() = %v, want %v", got, tt.want)
			}
		})
	}
}
