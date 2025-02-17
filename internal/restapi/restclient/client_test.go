package restclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestAPIClient(t *testing.T) {
	client := newMockClient(t, &ClientOptions{Timeout: 1})

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://restapi.local/ok",
		httpmock.NewStringResponder(http.StatusOK, "OK"),
	)

	httpmock.RegisterResponder(http.MethodGet, "/redirect", func(_ *http.Request) (*http.Response, error) {
		response := httpmock.NewStringResponse(http.StatusFound, "")

		response.Header.Set("Location", "https://restapi.local/ok")

		return response, nil
	})

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://restapi.local/slow",
		httpmock.NewStringResponder(http.StatusOK, "OK").Delay(time.Second*10),
	)

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr error
	}{
		{
			name:    "OK",
			url:     "/ok",
			want:    "OK",
			wantErr: nil,
		},
		{
			name:    "Redirect",
			url:     "/redirect",
			want:    "OK",
			wantErr: nil,
		},
		{
			name:    "Timeout",
			url:     "/slow",
			want:    "",
			wantErr: ErrHTTPRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, _, err := client.SendRequest(t.Context(), "GET", tt.url, "")
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)

				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, res, tt.want)
		})
	}
}

func TestAPIClientRateLimit(t *testing.T) {
	client := newMockClient(t, &ClientOptions{RateLimit: 1})

	httpmock.RegisterResponder(
		http.MethodGet,
		"https://restapi.local/ok",
		httpmock.NewStringResponder(http.StatusOK, "OK"),
	)

	t.Run("Rate limit", func(t *testing.T) {
		// Testing rate limited OK request
		startTime := time.Now().Unix()

		for i := 0; i < 4; i++ {
			client.SendRequest(t.Context(), "GET", "/ok", "")
		}

		duration := time.Now().Unix() - startTime
		if duration < 3 {
			t.Fatalf("requests not delayed")
		}
	})
}

func newMockClient(t *testing.T, opts *ClientOptions) *RestClient {
	t.Helper()

	if opts.Endpoint == "" {
		opts.Endpoint = "https://restapi.local/"
	}

	client, _ := New(t.Context(), opts)

	httpmock.ActivateNonDefault(client.HTTPClient)

	t.Cleanup(func() {
		httpmock.DeactivateAndReset()
	})

	return client
}
