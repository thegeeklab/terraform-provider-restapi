package restobject

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	client := newMockClient(t, &restclient.ClientOptions{})

	tests := []struct {
		name        string
		searchKey   string
		searchValue string
		want        testObject
		wantErr     error
	}{
		{
			name:        "find cat",
			searchKey:   "thing",
			searchValue: "cat",
			want:        newTestObject(t, testObjectData["pet"]),
			wantErr:     nil,
		},
		{
			name:        "find dog",
			searchKey:   "thing",
			searchValue: "dog",
			want:        newTestObject(t, testObjectData["pet"]),
			wantErr:     ErrFindObject,
		},
	}
	for _, tt := range tests {
		httpmock.RegisterResponder(
			client.Options.ReadMethod,
			fmt.Sprintf("https://restapi.local/%s", tt.searchValue),
			httpmock.NewJsonResponderOrPanic(http.StatusOK, []testObject{tt.want}),
		)

		t.Run(tt.name, func(t *testing.T) {
			ro, _ := New(client, &ObjectOptions{Path: tt.searchValue})

			got, err := ro.Find(context.Background(), "", tt.searchKey, tt.searchValue, "")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tt.want, mapToTestObject(t, got))
		})
	}
}
