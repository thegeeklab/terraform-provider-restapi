package restobject

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-restapi/internal/restapi"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrCreateObject = errors.New("failed to create object")

// Create uses the RestObject's client to send a POST request to create the
// object on the remote API. It handles setting the object's ID from the
// response and syncing the object's state.
func (ro *RestObject) Create(ctx context.Context) error {
	opts := ro.Options

	// Failsafe: The constructor should prevent this situation, but protect here also.
	// If no id is set, and the API does not respond with the id of whatever gets created,
	// we have no way to know what the object's id will be. Abandon this attempt.
	if opts.ID == "" && !ro.client.Options.WriteReturnsObject && !ro.client.Options.CreateReturnsObject {
		// Users must set write_returns_object to true or include an id in the object's data
		return fmt.Errorf("%w: %s", ErrCreateObject, "no id and client not configured to read response")
	}

	data, err := restapi.GetRequestData(opts.Data, nil)
	if err != nil {
		return err
	}

	postPath := opts.PostPath

	if opts.QueryString != "" {
		tflog.Debug(ctx, fmt.Sprintf("add query string '%s'", opts.QueryString))

		postPath = fmt.Sprintf("%s?%s", opts.PostPath, opts.QueryString)
	}

	resultString, _, err := ro.client.SendRequest(
		ctx, opts.CreateMethod, strings.ReplaceAll(postPath, "{id}", opts.ID), data)
	if err != nil {
		return err
	}

	// We will need to sync state as well as get the object's ID.
	if ro.client.Options.WriteReturnsObject || ro.client.Options.CreateReturnsObject {
		tflog.Debug(ctx, fmt.Sprintf("parse POST response: write_returns_object=%t, create_returns_object=%t",
			ro.client.Options.WriteReturnsObject, ro.client.Options.CreateReturnsObject,
		))

		err = ro.setData(ctx, resultString)

		// Yet another failsafe. In case something terrible went wrong internally,
		// bail out so the user at least knows that the ID did not get set.
		if opts.ID == "" {
			return fmt.Errorf("%w: %s", ErrCreateObject,
				"validation failed: no id but object may have been created: this should never happen")
		}
	} else {
		tflog.Debug(ctx, fmt.Sprintf("request created object from API: write_returns_object=%t, create_returns_object=%t",
			ro.client.Options.WriteReturnsObject, ro.client.Options.CreateReturnsObject))

		err = ro.Read(ctx)
	}

	opts.CreateResponseRaw = opts.APIResponseRaw

	return err
}
