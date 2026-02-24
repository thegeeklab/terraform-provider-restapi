package restobject

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"
	"github.com/thegeeklab/terraform-provider-restapi/internal/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrInvalidObjectOptions = errors.New("invalid object options")

type (
	// APIPayload is a map that contains arbitrary JSON-serializable data
	// representing the payload of an API request.
	APIPayload map[string]any
	// APIResponse is a map that contains arbitrary response data.
	APIResponse map[string]any
)

// APIObject is the state holding struct for a restapi_object resource.
type RestObject struct {
	client  *restclient.RestClient
	Options *ObjectOptions
}

type ObjectOptions struct {
	Path         string
	PostPath     string
	GetPath      string
	PutPath      string
	DeletePath   string
	CreateMethod string
	ReadMethod   string
	UpdateMethod string
	DeleteMethod string
	QueryString  string
	ReadSearch   *ReadSearch
	ID           string
	IDAttribute  string

	// Set internally
	Data              APIPayload  // Data as managed by the user
	UpdateData        APIPayload  // Update data as managed by the user
	DestroyData       APIPayload  // Destroy data as managed by the user
	APIResponse       APIResponse // Data as available from the API
	APIResponseRaw    string
	CreateResponseRaw string
}

type ReadSearch struct {
	SearchKey   string
	SearchValue string
	ResultKey   string
	QueryString string
}

// New creates a new RestObject instance with the given client and options.
// It sets default values for the object options if they are not provided,
// using values from the client options. It also tries to set the object ID
// from the provided data if an ID is not already set in the options.
func New(client *restclient.RestClient, opts *ObjectOptions) (*RestObject, error) {
	URLSuffixID := "{id}"
	ro := &RestObject{}

	if opts.IDAttribute == "" {
		opts.IDAttribute = client.Options.IDAttribute
	}

	if opts.CreateMethod == "" {
		opts.CreateMethod = client.Options.CreateMethod
	}

	if opts.ReadMethod == "" {
		opts.ReadMethod = client.Options.ReadMethod
	}

	if opts.UpdateMethod == "" {
		opts.UpdateMethod = client.Options.UpdateMethod
	}

	if opts.DeleteMethod == "" {
		opts.DeleteMethod = client.Options.DestroyMethod
	}

	if opts.PostPath == "" {
		opts.PostPath = opts.Path
	}

	if opts.GetPath == "" {
		opts.GetPath = filepath.Join(opts.Path, URLSuffixID)
	}

	if opts.PutPath == "" {
		opts.PutPath = filepath.Join(opts.Path, URLSuffixID)
	}

	if opts.DeletePath == "" {
		opts.DeletePath = filepath.Join(opts.Path, URLSuffixID)
	}

	if opts.ReadSearch == nil {
		opts.ReadSearch = &ReadSearch{}
	}

	// Opportunistically set the object's ID if it is provided in the data.
	// If it is not set, we will get it later in synchronize_state.
	if opts.Data != nil && opts.ID == "" {
		var tmp string

		tmp, err := utils.GetStringAtKey(opts.Data, opts.IDAttribute)
		if err == nil {
			opts.ID = tmp
		} else if !client.Options.WriteReturnsObject && !client.Options.CreateReturnsObject && opts.Path == "" {
			// If the id is not set and we cannot obtain it later, error out to be safe.
			return ro, fmt.Errorf("%w: object can not be managed: id_attribute '%s' not found in object "+
				"and write_returns_object or create_returns_object not set", ErrInvalidObjectOptions, opts.IDAttribute)
		}
	}

	ro.client = client
	ro.Options = opts

	return ro, nil
}

// ToString returns a string representation of the RestObject options.
func (ro *RestObject) ToString() string {
	var buffer bytes.Buffer

	opts := ro.Options

	fmt.Fprintf(&buffer, "id: %s\n", opts.ID)
	fmt.Fprintf(&buffer, "get_path: %s\n", opts.GetPath)
	fmt.Fprintf(&buffer, "post_path: %s\n", opts.PostPath)
	fmt.Fprintf(&buffer, "put_path: %s\n", opts.PutPath)
	fmt.Fprintf(&buffer, "delete_path: %s\n", opts.DeletePath)
	fmt.Fprintf(&buffer, "query_string: %s\n", opts.QueryString)
	fmt.Fprintf(&buffer, "create_method: %s\n", opts.CreateMethod)
	fmt.Fprintf(&buffer, "read_method: %s\n", opts.ReadMethod)
	fmt.Fprintf(&buffer, "update_method: %s\n", opts.UpdateMethod)
	fmt.Fprintf(&buffer, "destroy_method: %s\n", opts.DeleteMethod)
	fmt.Fprintf(&buffer, "read_search: %s\n", spew.Sdump(opts.ReadSearch))
	fmt.Fprintf(&buffer, "data: %s\n", spew.Sdump(opts.Data))
	fmt.Fprintf(&buffer, "update_data: %s\n", spew.Sdump(opts.UpdateData))
	fmt.Fprintf(&buffer, "destroy_data: %s\n", spew.Sdump(opts.DestroyData))
	fmt.Fprintf(&buffer, "api_response: %s\n", spew.Sdump(opts.APIResponse))

	return buffer.String()
}

// setData updates the RestObject's data from the provided API response.
// It extracts the ID if not already set, copies configured keys from the
// API response to the data, and stores the raw API response.
func (ro *RestObject) setData(ctx context.Context, state string) error {
	var err error

	opts := ro.Options

	tflog.Debug(ctx, fmt.Sprintf("update api object data: '%s'", state))

	// Store filtered response body.
	opts.APIResponse, opts.APIResponseRaw, err = utils.FilterJSONString(
		state, ro.client.Options.ResponseFilter.Keys, ro.client.Options.ResponseFilter.Include)
	if err != nil {
		return err
	}

	// A usable ID was not passed (in constructor or here),
	// so we have to guess what it is from the data structure.
	if opts.ID == "" {
		val, err := utils.GetStringAtKey(opts.APIResponse, opts.IDAttribute)
		if err != nil {
			return fmt.Errorf("error extracting id from data element: %w", err)
		}

		opts.ID = val
	} else {
		tflog.Debug(ctx, fmt.Sprintf("not updating id as already set: '%s'", opts.ID))
	}

	if opts.Data == nil {
		tflog.Debug(ctx, fmt.Sprintf("final object after data sync: %+v", ro.ToString()))

		return nil
	}

	if len(ro.client.Options.CopyKeys) > 0 {
		// Any keys that come from the data we want to copy are done here
		for _, key := range ro.client.Options.CopyKeys {
			tflog.Debug(ctx, fmt.Sprintf("copy key '%s' from api_response (%v) to data (%v)",
				key, opts.APIResponse[key], opts.Data[key]))

			opts.Data[key] = opts.APIResponse[key]
		}
	} else if ro.client.Options.DriftDetection {
		for key, value := range utils.IntersectMaps(opts.Data, opts.APIResponse) {
			tflog.Debug(ctx, fmt.Sprintf("copy key '%s' from api_response (%v) to data (%v)",
				key, value, opts.Data[key]))

			opts.Data[key] = value
		}
	}

	tflog.Debug(ctx, fmt.Sprintf("final object after data sync: %+v", ro.ToString()))

	return err
}
