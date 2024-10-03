package restobject

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/thegeeklab/terraform-provider-restapi/internal/utils"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrUpdateObject = errors.New("failed to update object")

// Update updates the RestObject by sending a PUT request to the API.
// It returns an error if the ID is not set, if there is an error building the
// request data, or if there is an error sending the request.
//
// If write_returns_object is true, it will parse the response and update the
// RestObject. Otherwise it will re-read the object from the API after the update.
func (ro *RestObject) Update(ctx context.Context) error {
	opts := ro.Options

	if opts.ID == "" {
		return fmt.Errorf("%w: id not set", ErrUpdateObject)
	}

	data, err := utils.GetRequestData(opts.Data, opts.UpdateData)
	if err != nil {
		return err
	}

	resultString, _, err := ro.client.SendRequest(
		ctx, opts.UpdateMethod, strings.ReplaceAll(opts.PutPath, "{id}", opts.ID), data)
	if err != nil {
		return err
	}

	if ro.client.Options.WriteReturnsObject {
		tflog.Debug(ctx, fmt.Sprintf("parse PUT response: write_returns_object=%t",
			ro.client.Options.WriteReturnsObject,
		))

		err = ro.setData(ctx, resultString)
	} else {
		tflog.Debug(ctx, fmt.Sprintf("request updated object from API: write_returns_object=%t",
			ro.client.Options.WriteReturnsObject))

		err = ro.Read(ctx)
	}

	return err
}
