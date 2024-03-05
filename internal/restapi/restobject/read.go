package restobject

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrReadObject = errors.New("failed to read object")

// Read retrieves the RestObject from the API based on the configured ID and options.
// It handles errors and unset IDs. It can also search the response and return a matched object.
func (ro *RestObject) Read(ctx context.Context) error {
	opts := ro.Options

	if opts.ID == "" {
		return fmt.Errorf("%w: id not set", ErrReadObject)
	}

	getPath := opts.GetPath

	if opts.QueryString != "" {
		tflog.Debug(ctx, fmt.Sprintf("add query string '%s'", opts.QueryString))
		getPath = fmt.Sprintf("%s?%s", opts.GetPath, opts.QueryString)
	}

	result, status, err := ro.client.SendRequest(ctx, opts.ReadMethod, strings.ReplaceAll(getPath, "{id}", opts.ID), "")
	if err != nil {
		if status == http.StatusNotFound {
			tflog.Error(ctx, fmt.Sprintf("%s: failed to refresh state for '%s' at path '%s': removing from state",
				err, opts.ID, opts.GetPath))

			opts.ID = ""

			return nil
		}

		return err
	}

	if opts.ReadSearch.SearchKey != "" && opts.ReadSearch.SearchValue != "" {
		queryString := opts.ReadSearch.QueryString
		resultKey := opts.ReadSearch.ResultKey
		searchKey := opts.ReadSearch.SearchKey
		searchValue := opts.ReadSearch.SearchValue

		if opts.QueryString != "" {
			tflog.Debug(ctx, fmt.Sprintf("add query string '%s'", opts.QueryString))
			queryString = fmt.Sprintf("%s&%s", queryString, opts.QueryString)
		}

		objFound, err := ro.Find(ctx, queryString, searchKey, searchValue, resultKey)
		if err != nil {
			opts.ID = ""

			return nil //nolint:nilerr
		}

		objFoundString, err := json.Marshal(objFound)
		if err != nil {
			return err
		}

		result = string(objFoundString)
	}

	return ro.setData(ctx, result)
}
