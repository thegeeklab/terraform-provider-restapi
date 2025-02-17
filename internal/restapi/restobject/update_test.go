package restobject

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestUpdate(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{CreateReturnsObject: true})

	tests := []struct {
		name    string
		key     string
		data    string
		wantErr error
	}{
		{
			name:    "update id",
			key:     "dummy",
			data:    testObjectData["minimal"],
			wantErr: nil,
		},
		{
			name:    "update empty id",
			key:     "",
			data:    testObjectData["minimal"],
			wantErr: ErrUpdateObject,
		},
	}
	for _, tt := range tests {
		want := newTestObject(t, tt.data)

		httpmock.RegisterResponder(
			client.Options.UpdateMethod,
			fmt.Sprintf("https://restapi.local/%s", tt.key),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, want),
		)

		httpmock.RegisterResponder(
			client.Options.ReadMethod,
			fmt.Sprintf("https://restapi.local/%s", tt.key),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, want),
		)

		t.Run(tt.name, func(t *testing.T) {
			ro, _ := New(client, &ObjectOptions{ID: tt.key})
			json.Unmarshal([]byte(tt.data), &ro.Options.APIResponse)
			ro.Options.APIResponse["thing"] = "spoon"

			err := ro.Update(t.Context())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, want, mapToTestObject(t, ro.Options.APIResponse))
		})
	}
}
