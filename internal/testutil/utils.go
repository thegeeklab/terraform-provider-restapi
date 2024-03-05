package testutil

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

// SetupRootLogger initializes a tflogtest root logger that writes to the given
// bytes.Buffer .It returns a context bound to that logger and the buffer that
// received the log output.
func SetupRootLogger() (context.Context, *bytes.Buffer) {
	var output bytes.Buffer

	return tflogtest.RootLogger(context.Background(), &output), &output
}

// HasLogMessage checks if the given test logger logs contain the given
// message text. It is intended to be used in test cases to verify expected
// log messages.
func HasLogMessage(t *testing.T, want string, logs *bytes.Buffer) bool {
	t.Helper()

	entries, err := tflogtest.MultilineJSONDecode(logs)
	if err != nil {
		t.Fatalf("log output parsing failed: %v", err)
	}

	for _, value := range entries {
		if strings.Contains(fmt.Sprintf("%v", value["@message"]), want) {
			return true
		}
	}

	return false
}
