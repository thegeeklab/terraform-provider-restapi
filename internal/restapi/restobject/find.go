package restobject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/thegeeklab/terraform-provider-restapi/internal/utils"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	ErrFindObject   = errors.New("failed to find object")
	ErrFindResponse = errors.New("failed to read response from find object")
)

// Find searches the REST API for an object matching the given search criteria.
// It issues a GET request to the API path, optionally adding the queryString.
// It parses the JSON response, extracting the result array at resultKey.
// It loops through the array looking for an object where searchKey equals searchValue.
// If found, it returns that object as the APIResponse.
// It also extracts the ID attribute into the RestObject options.
func (ro *RestObject) Find(
	ctx context.Context, queryString, searchKey, searchValue, resultKey string,
) (APIResponse, error) {
	var (
		resp      APIResponse
		dataArray []any
		result    any
		ok        bool
	)

	opts := ro.Options
	searchPath := opts.Path

	// Issue a GET to the base path and expect results to come back
	if queryString != "" {
		tflog.Debug(ctx, fmt.Sprintf("add query string '%s'", queryString))
		searchPath = fmt.Sprintf("%s?%s", searchPath, queryString)
	}

	tflog.Debug(ctx, fmt.Sprintf("call api with path '%s'", searchPath))

	resultString, _, err := ro.client.SendRequest(ctx, ro.client.Options.ReadMethod, searchPath, "")
	if err != nil {
		return resp, err
	}

	// Parse it seeking JSON data
	tflog.Debug(ctx, "parse received response")

	err = json.Unmarshal([]byte(resultString), &result)
	if err != nil {
		return resp, err
	}

	dataArray, err = getDataArray(result, resultKey)
	if err != nil {
		return resp, err
	}

	// Loop through all of the results seeking the specific record
	for _, item := range dataArray {
		var hash APIResponse

		if hash, ok = item.(map[string]any); !ok {
			return resp, fmt.Errorf("%w: data not a map of key value pairs", ErrFindResponse)
		}

		tflog.Debug(ctx, fmt.Sprintf("examining %v", hash))
		tflog.Debug(ctx, fmt.Sprintf("comparing '%s' to value of '%s'", searchValue, searchKey))

		tmp, err := utils.GetStringAtKey(hash, searchKey)
		if err != nil {
			return resp, (fmt.Errorf("%w: %w: failed to get value of '%s' in results array at '%s'",
				ErrFindResponse, err, searchKey, resultKey))
		}

		// Record found
		if tmp == searchValue {
			resp = hash

			opts.ID, err = utils.GetStringAtKey(hash, opts.IDAttribute)
			if err != nil {
				return resp, fmt.Errorf("%w: %w: no id_attribute '%s' in the record",
					ErrFindResponse, err, opts.IDAttribute)
			}

			tflog.Debug(ctx, fmt.Sprintf("found id '%s'", opts.ID))

			// But there is no id attribute
			if opts.ID == "" {
				return resp, fmt.Errorf("%w: attribute '%s' not in object for '%s'='%s', or empty value",
					ErrFindResponse, opts.IDAttribute, searchKey, searchValue)
			}

			break
		}
	}

	if opts.ID == "" {
		return resp, fmt.Errorf("%w: no object with '%s' = '%s' at %s",
			ErrFindObject, searchKey, searchValue, searchPath)
	}

	return resp, nil
}

// getDataArray extracts the data array from the find result.
// If resultKey is specified, it looks for that key in the result map.
// Otherwise, it expects the result to be a data array directly.
// Returns the data array and any error.
func getDataArray(result any, resultKey string) ([]any, error) {
	var (
		data []any
		tmp  any
		ok   bool
		err  error
	)

	if resultKey == "" {
		if data, ok = result.([]any); !ok {
			return data, fmt.Errorf("%w: epexted array but got '%s'",
				ErrFindResponse, reflect.TypeOf(result))
		}

		return data, nil
	}

	// First verify the data we got back is a hash
	if _, ok = result.(map[string]any); !ok {
		return data, fmt.Errorf("%w: cannot search for result_key '%s': not a hash",
			ErrFindResponse, resultKey)
	}

	tmp, err = utils.GetObjectAtKey(result.(map[string]any), resultKey)
	if err != nil {
		return data, fmt.Errorf("%w: %w: result_key not found", ErrFindResponse, err)
	}

	if data, ok = tmp.([]any); !ok {
		return data, fmt.Errorf("%w: result_key '%s': data not an array but '%T'",
			ErrFindResponse, resultKey, tmp)
	}

	return data, nil
}
