package restobject

import (
	"context"
	"encoding/json"
	"testing"

	"terraform-provider-restapi/internal/restapi/restclient"

	"github.com/jarcoal/httpmock"
)

type testAPIObject struct {
	ID       string            `json:"id"`
	Revision int               `json:"revision,omitempty"`
	Thing    string            `json:"thing,omitempty"`
	IsCat    bool              `json:"isCat,omitempty"`
	Colors   []string          `json:"colors,omitempty"`
	Attrs    map[string]string `json:"attrs,omitempty"`
}

var testingDataObjects = map[string]string{
	"normal": `{
	  "id": "1",
	  "revision": 1,
	  "thing": "potato",
	  "isCat": false,
	  "Colors": [
		"orange",
		"white"
	  ],
	  "Attrs": {
		"size": "6 in",
		"weight": "10 oz"
	  }
	}`,
	"minimal": `{
      "id": "2",
      "thing": "fork"
    }`,
	"nocolor": `{
      "id": "3",
      "thing": "paper",
      "isCat": false,
      "attrs": {
        "height": "8.5 in",
        "width": "11 in"
      }
    }`,
	"noattr": `{
      "id": "4",
      "thing": "nothing",
      "isCat": false,
      "colors": [
        "none"
      ]
    }`,
	"pet": `{
      "id": "5",
      "thing": "cat",
      "isCat": true,
      "colors": [
        "orange",
        "white"
      ],
      "attrs": {
        "size": "1.5 ft",
        "weight": "15 lb"
      }
    }`,
}

func newTestObject(t *testing.T, input string) testAPIObject {
	t.Helper()

	var testObject testAPIObject

	if err := json.Unmarshal([]byte(input), &testObject); err != nil {
		t.Fatalf("failed to unmarshall json from '%s'", input)
	}

	return testObject
}

func mapToTestObject(t *testing.T, resp APIResponse) testAPIObject {
	t.Helper()

	// Convert map to json string
	jsonStr, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal json from '%s'", resp)
	}

	return newTestObject(t, string(jsonStr))
}

func newMockClient(t *testing.T, opts *restclient.ClientOptions) *restclient.RestClient {
	t.Helper()

	if opts.Endpoint == "" {
		opts.Endpoint = "https://restapi.local/"
	}

	client, _ := restclient.New(context.Background(), opts)

	httpmock.ActivateNonDefault(client.HTTPClient)

	t.Cleanup(func() {
		httpmock.DeactivateAndReset()
	})

	return client
}
