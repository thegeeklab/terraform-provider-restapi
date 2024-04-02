package provider

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-restapi/internal/restapi/restclient"
	"terraform-provider-restapi/internal/restapi/restobject"
	"terraform-provider-restapi/internal/utils"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &RestobjectDataSource{}

func NewRestobjectDataSource() datasource.DataSource {
	return &RestobjectDataSource{}
}

// RestobjectDataSource defines the data source implementation.
type RestobjectDataSource struct {
	client *restclient.RestClient
}

func (d *RestobjectDataSource) Metadata(
	_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (d *RestobjectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	// Consider data sensitive if env variables is set to true.
	isDataSensitive, _ := strconv.ParseBool(utils.GetEnvOrDefault("RESTAPI_SENSITIVE_DATA", "false"))

	resp.Schema = schema.Schema{
		Description: "Restapi object data source schema.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal resource ID.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The API path in addition to the base URL defined in the provider configuration, " +
					"which represents objects of this type on the API server.",
				Required: true,
			},
			"create_path": schema.StringAttribute{
				Description: "Defaults to `path`. The API path that specifies where objects of this type " +
					"are to be created (`POST`) on the API server. The string `{id}` is replaced by the Terraform ID " +
					"of the object if the data contains the attribute `id_attribute`.",
				Computed: true,
			},
			"read_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be read (`GET`) on the API server. The string `{id}` is replaced by the Terraform ID of " +
					"the object.",
				Optional: true,
			},
			"update_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be updated (`PUT`) on the API server. The string `{id}` is replaced by the Terraform ID of " +
					"the object.",
				Computed: true,
			},
			"destroy_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be deleted (`DELETE`) on the API server. The string `{id}` is replaced by the Terraform ID of " +
					"the object.",
				Computed: true,
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `create_method` defined in the provider configuration. " +
					"Allows override of `create_method` (see `create_method` provider documentation) per data source.",
				Computed: true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `read_method` defined in the provider configuration. " +
					"Allows override of `read_method` (see `read_method` provider documentation) per data source.",
				Optional: true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `update_method` defined in the provider configuration. " +
					"Allows override of `update_method` (see `update_method` provider documentation) per data source.",
				Computed: true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `destroy_method` defined in the provider configuration. " +
					"Allows override of `destroy_method` (see `destroy_method` provider documentation) per data source.",
				Computed: true,
			},
			"id_attribute": schema.StringAttribute{
				Description: "Defaults to `id_attribute` defined in the provider configuration. " +
					"Allows override of `id_attribute` (see `id_attribute` provider documentation) per data source.",
				Optional: true,
			},
			"object_id": schema.StringAttribute{
				Description: "Defaults to the auto-generated `id` gathered during normal operations and `id_attribute`. " +
					"Allows to set the ID manually. This is used in conjunction with the `*_path` attributes.",
				Computed: true,
			},
			"data": schema.StringAttribute{
				Description: "JSON object managed by the provider that holds information from the API response.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"read_search": schema.SingleNestedAttribute{
				Description: "Custom search for `read_path`.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"search_key": schema.StringAttribute{
						Description: "Key to identify a specific data record in the data array. " +
							"This should be a unique identifier e.g. `name`. Similar to `results_key`, " +
							"the value can have the format `path/to/key` to search for a nested object.",
						Required: true,
					},
					"search_value": schema.StringAttribute{
						Description: "Value to compare with the value of `search_key` to determine whether " +
							"the correct object has been found. Example: If `search_key=name` and `search_value=foo`, " +
							"the record in the data array with the matching attribute `name=foo` is used.",
						Required: true,
					},
					"result_key": schema.StringAttribute{
						Description: "Key to identify the data array with result objects in the API response. " +
							"The format is `path/to/key`. If this key is omitted, it is assumed that " +
							"the response data is already an array and should be used directly.",
						Optional: true,
					},
					"query_string": schema.StringAttribute{
						Description: "Defaults to `query_string`. Optional query string used for API read requests.",
						Optional:    true,
					},
				},
			},
			"query_string": schema.StringAttribute{
				Description: "Query string to be included in the path.",
				Optional:    true,
			},
			"api_response": schema.MapAttribute{
				ElementType: types.StringType,
				Description: "API response data. This map includes k/v pairs usable in other resources as readable objects. " +
					"Currently the value is the `golang fmt` representation of the value. Simple primitives are set as expected, " +
					"but complex types like arrays and maps contain `golang` formatting.",
				Computed:  true,
				Sensitive: isDataSensitive,
			},
			"api_response_raw": schema.StringAttribute{
				Description: "The raw body of the HTTP response from the last read of the object.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"create_response_raw": schema.StringAttribute{
				Description: "The raw body of the HTTP response from the object creation.",
				Computed:    true,
				Sensitive:   isDataSensitive,
			},
			"update_data": schema.StringAttribute{
				Computed:    true,
				Description: "JSON object that is passed to update requests.",
				Sensitive:   isDataSensitive,
			},
			"destroy_data": schema.StringAttribute{
				Computed:    true,
				Description: "JSON object that is passed to destroy requests.",
				Sensitive:   isDataSensitive,
			},
		},
	}
}

func (d *RestobjectDataSource) Configure(
	_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*restclient.RestClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *http.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	d.client = client
}

func (d *RestobjectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestobjectResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	objectOpts, diags := toObjectOptions(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ro, err := restobject.New(d.client, objectOpts)
	if err != nil {
		resp.Diagnostics.AddError("Can not create restobject", err.Error())

		return
	}

	if _, err = ro.Find(
		ctx, data.QueryString.ValueString(),
		ro.Options.ReadSearch.SearchKey,
		ro.Options.ReadSearch.SearchValue,
		ro.Options.ReadSearch.ResultKey,
	); err != nil {
		resp.Diagnostics.AddError("Can not find restobject", err.Error())

		return
	}

	err = ro.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Can not read restobject", err.Error())

		return
	}

	resp.Diagnostics.Append(mapFields(ctx, ro.Options, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}
