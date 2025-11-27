package restobject

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"
	"github.com/thegeeklab/terraform-provider-restapi/internal/testutils"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestDelete(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{})

	tests := []struct {
		name       string
		key        string
		statusCode int
		wantErr    error
		wantLog    string
	}{
		{
			name:       "delete id",
			key:        "valid",
			statusCode: http.StatusOK,
			wantErr:    nil,
			wantLog:    "",
		},
		{
			name:       "empty id",
			key:        "",
			statusCode: http.StatusOK,
			wantErr:    nil,
			wantLog:    "attempt to delete an object that has no id set",
		},
		{
			name:       "delete already deleted (404)",
			key:        "not-found",
			statusCode: http.StatusNotFound,
			wantErr:    nil,
			wantLog:    "",
		},
		{
			name:       "delete already deleted (410 Gone)",
			key:        "gone",
			statusCode: http.StatusGone,
			wantErr:    nil,
			wantLog:    "",
		},
	}
	for _, tt := range tests {
		httpmock.RegisterResponder(client.Options.DestroyMethod, fmt.Sprintf("https://restapi.local/%s", tt.key),
			httpmock.NewStringResponder(tt.statusCode, "OK"),
		)

		t.Run(tt.name, func(t *testing.T) {
			ctx, output := testutils.SetupRootLogger()
			ro, _ := New(client, &ObjectOptions{ID: tt.key})

			err := ro.Delete(ctx)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			if tt.wantLog != "" {
				assert.Truef(t, testutils.HasLogMessage(t, tt.wantLog, output), "expected log message not found: %s", tt.wantLog)
			}

			assert.NoError(t, err)
		})
	}
}
