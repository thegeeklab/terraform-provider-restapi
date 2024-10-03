package restobject

import (
	"context"
	"net/http"
	"testing"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"

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
		name      string
		opts      *ObjectOptions
		data      []testObject
		want      testObject
		wantIndex int
		wantErr   error
	}{
		{
			name:    "valid object",
			opts:    &ObjectOptions{ID: "minimal", GetPath: "/get"},
			want:    newTestObject(t, testObjectData["minimal"]),
			wantErr: nil,
		},
		{
			name: "valid array",
			opts: &ObjectOptions{
				ID:         "minimal",
				GetPath:    "/get",
				Path:       "/get",
				ReadSearch: &ReadSearch{SearchKey: "id", SearchValue: "2"},
			},
			data:    newTestObjectList(t, testObjectData["minimal"]),
			want:    newTestObject(t, testObjectData["minimal"]),
			wantErr: nil,
		},
		{
			name:    "empty id",
			opts:    &ObjectOptions{ID: "", GetPath: "/get"},
			want:    testObject{},
			wantErr: ErrReadObject,
		},
		{
			name:    "error request",
			opts:    &ObjectOptions{ID: "fail", GetPath: "/fail"},
			want:    testObject{},
			wantErr: restclient.ErrUnexpectedResponseCode,
		},
		{
			name:    "not found request",
			opts:    &ObjectOptions{ID: "missing", GetPath: "/missing"},
			want:    testObject{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		resp := httpmock.NewJsonResponderOrPanic(http.StatusOK, tt.want)

		if tt.data != nil {
			resp = httpmock.NewJsonResponderOrPanic(http.StatusOK, tt.data)
		}

		httpmock.RegisterResponder(
			client.Options.ReadMethod,
			"https://restapi.local/get",
			resp,
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
