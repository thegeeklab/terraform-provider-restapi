package restobject

import (
	"context"
	"net/http"
	"testing"

	"terraform-provider-restapi/internal/restapi/restclient"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{})

	httpmock.RegisterResponder(
		client.Options.ReadMethod,
		"https://restapi.local/fail",
		httpmock.NewJsonResponderOrPanic(http.StatusForbidden, nil),
	)
	httpmock.RegisterResponder(
		client.Options.ReadMethod,
		"https://restapi.local/missing",
		httpmock.NewJsonResponderOrPanic(http.StatusNotFound, nil),
	)

	tests := []struct {
		name    string
		opts    *ObjectOptions
		want    testAPIObject
		wantErr error
	}{
		{
			name:    "valid id",
			opts:    &ObjectOptions{ID: "minimal", GetPath: "/get"},
			want:    newTestObject(t, testingDataObjects["minimal"]),
			wantErr: nil,
		},
		{
			name:    "empty id",
			opts:    &ObjectOptions{ID: "", GetPath: "/get"},
			want:    testAPIObject{},
			wantErr: ErrReadObject,
		},
		{
			name:    "error request",
			opts:    &ObjectOptions{ID: "fail", GetPath: "/fail"},
			want:    testAPIObject{},
			wantErr: restclient.ErrUnexpectedResponseCode,
		},
		{
			name:    "not found request",
			opts:    &ObjectOptions{ID: "missing", GetPath: "/missing"},
			want:    testAPIObject{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		httpmock.RegisterResponder(
			client.Options.ReadMethod,
			"https://restapi.local/get",
			httpmock.NewJsonResponderOrPanic(http.StatusOK, tt.want),
		)

		t.Run(tt.name, func(t *testing.T) {
			ro, _ := New(client, tt.opts)

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
