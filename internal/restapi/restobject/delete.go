package restobject

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"terraform-provider-restapi/internal/restapi"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func (ro *RestObject) Delete(ctx context.Context) error {
	var err error

	opts := ro.Options

	if opts.ID == "" {
		tflog.Warn(ctx, "attempt to delete an object that has no id set: assuming this is ok")

		return nil
	}

	deletePath := opts.DeletePath

	if opts.QueryString != "" {
		tflog.Debug(ctx, fmt.Sprintf("add query string '%s'", opts.QueryString))

		deletePath = fmt.Sprintf("%s?%s", opts.DeletePath, opts.QueryString)
	}

	data, err := restapi.GetRequestData(opts.DestroyData, nil)
	if err != nil {
		return err
	}

	_, status, err := ro.client.SendRequest(
		ctx, opts.DeleteMethod, strings.ReplaceAll(deletePath, "{id}", opts.ID), data)
	if err != nil && status != http.StatusNotFound {
		return err
	}

	return nil
}
