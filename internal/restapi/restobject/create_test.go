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

func TestCreate(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{CreateReturnsObject: true})

	tests := []struct {
		name    string
		key     string
		want    testObject
		wantErr error
	}{
		{
			name:    "create minimal",
			key:     "minimal",
			want:    newTestObject(t, testObjectData["minimal"]),
			wantErr: nil,
		},
		{
			name:    "invalid input",
			key:     "",
			want:    testObject{},
			wantErr: ErrCreateObject,
		},
	}

	for _, tt := range tests {
		httpmock.RegisterResponder(
			client.Options.CreateMethod,
			fmt.Sprintf("https://restapi.local/%s", tt.key),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, tt.want),
		)

		t.Run(tt.name, func(t *testing.T) {
			ro, _ := New(client, &ObjectOptions{ID: tt.key, Path: fmt.Sprintf("/%s", tt.key)})

			err := ro.Create(context.Background())
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tt.want, mapToTestObject(t, ro.Options.APIResponse))
		})
	}
}
