package testutil

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func SetupRootLogger() (context.Context, *bytes.Buffer) {
	var output bytes.Buffer

	return tflogtest.RootLogger(context.Background(), &output), &output
}

func HasLogMessage(t *testing.T, want string, logs *bytes.Buffer) bool {
	t.Helper()

	entries, err := tflogtest.MultilineJSONDecode(logs)
	if err != nil {
		t.Fatalf("log outtput parsing failed: %v", err)
	}

	for _, value := range entries {
		if strings.Contains(fmt.Sprintf("%v", value["@message"]), want) {
			return true
		}
	}

	return false
}
