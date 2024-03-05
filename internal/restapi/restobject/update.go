package restobject

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-restapi/internal/restapi"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var ErrUpdateObject = errors.New("failed to update object")

func (ro *RestObject) Update(ctx context.Context) error {
	opts := ro.Options

	if opts.ID == "" {
		return fmt.Errorf("%w: id not set", ErrUpdateObject)
	}

	data, err := restapi.GetRequestData(opts.Data, opts.UpdateData)
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
