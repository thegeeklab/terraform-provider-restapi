package restobject

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"terraform-provider-restapi/internal/restapi/restclient"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{})

	httpmock.RegisterResponder(
		client.Options.CreateMethod,
		"https://restapi.local",
		httpmock.NewStringResponder(http.StatusOK, "OK"),
	)

	tests := []struct {
		name    string
		key     string
		want    testAPIObject
		wantErr error
	}{
		{
			name:    "create id",
			key:     "minimal",
			want:    newTestObject(t, testingDataObjects["minimal"]),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		httpmock.RegisterResponder(
			client.Options.ReadMethod,
			fmt.Sprintf("https://restapi.local/%s", tt.key),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, tt.want),
		)

		t.Run(tt.name, func(t *testing.T) {
			ro, _ := New(client, &ObjectOptions{ID: tt.key})

			err := ro.Read(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tt.want, mapToTestObject(t, ro.Options.APIResponse))
		})
	}
}
