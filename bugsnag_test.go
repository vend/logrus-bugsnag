package logrus_bugsnag

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bugsnag "github.com/bugsnag/bugsnag-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stackFrame struct {
	Method     string `json:"method"`
	File       string `json:"file"`
	LineNumber int    `json:"lineNumber"`
}

type exception struct {
	Message    string       `json:"message"`
	Stacktrace []stackFrame `json:"stacktrace"`
}

type event struct {
	Exceptions []exception      `json:"exceptions"`
	Metadata   bugsnag.MetaData `json:"metaData"`
}

type notice struct {
	Events []event `json:"events"`
}

func TestNoticeReceived(t *testing.T) {
	c := make(chan event, 1)
	expectedMessage := "foo"
	expectedMetadataLen := 3
	expectedFields := []string{"animal", "size", "omg"}
	expectedValues := []interface{}{"walrus", float64(9009), true}

	// create server to retrieve notification into a channel.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var notice notice
		data, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		err = json.Unmarshal(data, &notice)
		require.NoError(t, err)
		err = r.Body.Close()
		require.NoError(t, err)
		c <- notice.Events[0]
	}))
	defer ts.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer ts2.Close()

	bugsnag.Configure(bugsnag.Configuration{
		Endpoints:    bugsnag.Endpoints{Notify: ts.URL, Sessions: ts2.URL},
		ReleaseStage: "production",
		APIKey:       "12345678901234567890123456789012",
		Synchronous:  true,
	})

	// Add hook
	hook, err := NewBugsnagHook()
	require.NoError(t, err, "failed to create hook")
	log := logrus.New()
	log.Hooks.Add(hook)

	// Send log
	log.WithFields(logrus.Fields{
		"error":  errors.New(expectedMessage),
		"animal": "walrus",
		"size":   9009,
		"omg":    true,
	}).Error("Bugsnag will not see this string")

	select {
	case event := <-c:
		exception := event.Exceptions[0]
		assert.Equal(t, expectedMessage, exception.Message,
			fmt.Sprintf("Unexpected message received: got %q, expected %q", exception.Message, expectedMessage))

		assert.True(t, len(exception.Stacktrace) > 1, "Bugsnag error does not have a stack trace")
		metadata, ok := event.Metadata["metadata"]
		assert.True(t, ok, "Expected a Metadata field to be present in the bugsnag metadata")
		assert.Equal(t, expectedMetadataLen, len(metadata))

		for idx, field := range expectedFields {
			val, ok := metadata[field]
			assert.True(t, ok, fmt.Sprintf("Expected field %q not found", field))
			assert.Equal(t, expectedValues[idx], val,
				fmt.Sprintf("For field %q, found value %v, expected value %v", field, val, expectedValues[idx]))
		}

		topFrameMethod := exception.Stacktrace[0].Method
		assert.Equal(t, "TestNoticeReceived", topFrameMethod,
			fmt.Sprintf("Unexpected method on top of call stack: got %q, expected TestNoticeReceived", topFrameMethod))

	case <-time.After(time.Second):
		t.Fatal("Timed out; no notice received by Bugsnag API")
	}

	// will generate a different stacktrace compared to log.WithFields().Error()
	log.Errorf("Another error")

	select {
	case event := <-c:
		topFrame := event.Exceptions[0].Stacktrace[0]
		if topFrame.Method != "TestNoticeReceived" {
			t.Errorf("Unexpected method on top of call stack: got %q, expected %q", topFrame.Method,
				"TestNoticeReceived")
		}

	case <-time.After(time.Second):
		t.Error("Timed out; no notice received by Bugsnag API")
	}
}
