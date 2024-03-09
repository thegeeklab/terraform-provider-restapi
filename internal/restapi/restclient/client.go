package restclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/time/rate"
)

var (
	ErrInvalidClientOptions   = errors.New("invalid client options")
	ErrUnexpectedResponseCode = errors.New("unexpected http response code")
	ErrHTTPRequest            = errors.New("http request failed")
)

type ClientOptions struct {
	Endpoint               string
	Insecure               bool
	Username               string
	Password               string
	Headers                map[string]string
	UseCookies             bool
	Timeout                int64
	IDAttribute            string
	CreateMethod           string
	ReadMethod             string
	UpdateMethod           string
	DestroyMethod          string
	CopyKeys               []string
	ResponseFilter         *ResponseFilter
	DriftDetection         bool
	WriteReturnsObject     bool
	CreateReturnsObject    bool
	XSSIPrefix             string
	RateLimit              float64
	TestPath               string
	OAuthClientCredentials *OAuthCredentials
	CertString             string
	KeyString              string
	CertFile               string
	KeyFile                string
}

type OAuthCredentials struct {
	ClientID       string
	ClientSecret   string
	TokenEndpoint  string
	EndpointParams url.Values
	Scopes         []string
}

type ResponseFilter struct {
	Keys    []string
	Include bool
}

// RestClient is a HTTP client with additional controlling fields.
type RestClient struct {
	HTTPClient *http.Client
	Options    *ClientOptions

	rateLimiter *rate.Limiter
	oauthConfig *clientcredentials.Config
}

// New creates a new RestClient instance.
// It takes a context and ClientOptions, and returns a RestClient instance and error.
// It initializes the RestClient with the provided options, creating the HTTP client,
// OAuth client if configured, rate limiter, etc.
// It returns any errors encountered while initializing the client.
func New(ctx context.Context, opts *ClientOptions) (*RestClient, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("%w: endpoint not set", ErrInvalidClientOptions)
	}

	// Sanetize default
	if opts.IDAttribute == "" {
		opts.IDAttribute = "id"
	}

	// Remove any trailing slashes since we will append
	// to this URL with our own root-prefixed location
	opts.Endpoint = strings.TrimSuffix(opts.Endpoint, "/")

	if opts.CreateMethod == "" {
		opts.CreateMethod = "POST"
	}

	if opts.ReadMethod == "" {
		opts.ReadMethod = "GET"
	}

	if opts.UpdateMethod == "" {
		opts.UpdateMethod = "PUT"
	}

	if opts.DestroyMethod == "" {
		opts.DestroyMethod = "DELETE"
	}

	if opts.OAuthClientCredentials == nil {
		opts.OAuthClientCredentials = &OAuthCredentials{}
	}

	tlsConfig := &tls.Config{
		// Disable TLS verification if requested
		//nolint:gosec
		InsecureSkipVerify: opts.Insecure,
	}

	if opts.CertString != "" && opts.KeyString != "" {
		cert, err := tls.X509KeyPair([]byte(opts.CertString), []byte(opts.KeyString))
		if err != nil {
			return nil, err
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if opts.CertFile != "" && opts.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
		if err != nil {
			return nil, err
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}

	var cookieJar http.CookieJar

	if opts.UseCookies {
		cookieJar, _ = cookiejar.New(nil)
	}

	rateLimit := rate.Limit(opts.RateLimit)
	bucketSize := int(math.Max(math.Round(opts.RateLimit), 1))
	tflog.Info(ctx, fmt.Sprintf("limit: %f bucket: %d", opts.RateLimit, bucketSize))
	rateLimiter := rate.NewLimiter(rateLimit, bucketSize)

	rc := RestClient{
		HTTPClient: &http.Client{
			Timeout:   time.Second * time.Duration(opts.Timeout),
			Transport: tr,
			Jar:       cookieJar,
		},
		Options: opts,

		rateLimiter: rateLimiter,
	}

	if opts.OAuthClientCredentials.ClientID != "" &&
		opts.OAuthClientCredentials.ClientSecret != "" &&
		opts.OAuthClientCredentials.TokenEndpoint != "" {
		rc.oauthConfig = &clientcredentials.Config{
			ClientID:       opts.OAuthClientCredentials.ClientID,
			ClientSecret:   opts.OAuthClientCredentials.ClientSecret,
			TokenURL:       opts.OAuthClientCredentials.TokenEndpoint,
			Scopes:         opts.OAuthClientCredentials.Scopes,
			EndpointParams: opts.OAuthClientCredentials.EndpointParams,
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("api_client.go: Constructed client:\n%s", rc.ToString()))

	return &rc, nil
}

// SendRequest sends an HTTP request to the configured API endpoint.
// It handles constructing the request, adding headers and authentication,
// rate limiting, logging, and error handling.
func (rc *RestClient) SendRequest(ctx context.Context, method, path, data string) (string, int, error) {
	var (
		req *http.Request
		err error
	)

	opts := rc.Options
	url := fmt.Sprintf("%s/%s", strings.TrimRight(opts.Endpoint, "/"), strings.TrimLeft(path, "/"))

	tflog.Debug(ctx, fmt.Sprintf("method='%s', path='%s', full url (derived)='%s', data='%s'", method, path, url, data))

	if data == "" {
		req, err = http.NewRequest(method, url, nil)
	} else {
		buffer := bytes.NewBuffer([]byte(data))
		req, err = http.NewRequest(method, url, buffer)

		// Default of application/json, but allow headers array to overwrite later
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	if err != nil {
		return "", 0, err
	}

	tflog.Debug(ctx, fmt.Sprintf("send http request to %s", req.URL))

	// Allow for tokens or other pre-created secrets
	for n, v := range opts.Headers {
		req.Header.Set(n, v)
	}

	if rc.oauthConfig != nil {
		ctxx := context.WithValue(ctx, oauth2.HTTPClient, rc.HTTPClient)
		tokenSource := rc.oauthConfig.TokenSource(ctxx)

		token, err := tokenSource.Token()
		if err != nil {
			return "", 0, err
		}

		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	}

	if opts.Username != "" && opts.Password != "" {
		// ... and fall back to basic authentication if configured
		req.SetBasicAuth(opts.Username, opts.Password)
	}

	tflog.Debug(ctx, fmt.Sprintf("request headers: %v", req.Header))
	tflog.Debug(ctx, fmt.Sprintf("request body: %v", data))

	if rc.rateLimiter != nil {
		// Rate limiting
		tflog.Debug(ctx, "wait for rate limit availability")

		_ = rc.rateLimiter.Wait(ctx)
	}

	resp, err := rc.HTTPClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("%w: %s", ErrHTTPRequest, err.Error())
	}
	defer resp.Body.Close()

	tflog.Debug(ctx, fmt.Sprintf("response code: %d", resp.StatusCode))
	tflog.Debug(ctx, fmt.Sprintf("response header: %v", resp.Header))

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}

	body := strings.TrimPrefix(string(bodyBytes), opts.XSSIPrefix)
	tflog.Debug(ctx, fmt.Sprintf("response body: %s", body))

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return body, resp.StatusCode, fmt.Errorf("%w: http %d: %s", ErrUnexpectedResponseCode, resp.StatusCode, body)
	}

	return body, resp.StatusCode, nil
}

// ToString returns a string representation of the RestClient options.
func (rc *RestClient) ToString() string {
	var buffer bytes.Buffer

	opts := rc.Options

	buffer.WriteString(fmt.Sprintf("uri: %s\n", opts.Endpoint))
	buffer.WriteString(fmt.Sprintf("insecure: %t\n", opts.Insecure))
	buffer.WriteString(fmt.Sprintf("username: %s\n", opts.Username))
	buffer.WriteString(fmt.Sprintf("password: %s\n", opts.Password))
	buffer.WriteString(fmt.Sprintf("id_attribute: %s\n", opts.IDAttribute))
	buffer.WriteString(fmt.Sprintf("write_returns_object: %t\n", opts.WriteReturnsObject))
	buffer.WriteString(fmt.Sprintf("create_returns_object: %t\n", opts.CreateReturnsObject))
	buffer.WriteString("headers:\n")

	for k, v := range opts.Headers {
		buffer.WriteString(fmt.Sprintf(" %s: %s\n", k, v))
	}

	for _, n := range opts.CopyKeys {
		buffer.WriteString(fmt.Sprintf("  %s", n))
	}

	return buffer.String()
}
