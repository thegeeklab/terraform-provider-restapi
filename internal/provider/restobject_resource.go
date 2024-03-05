package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-restapi/internal/restapi/restclient"
	"terraform-provider-restapi/internal/restapi/restobject"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &RestobjectResource{}
	_ resource.ResourceWithImportState = &RestobjectResource{}
)

func NewRestobjectResource() resource.Resource {
	return &RestobjectResource{}
}

// APIObject is the state holding struct for a restapi_object resource.
type RestobjectResource struct {
	client *restclient.RestClient
}

type RestobjectResourceModel struct {
	Path       types.String `tfsdk:"path"`
	PostPath   types.String `tfsdk:"create_path"`
	GetPath    types.String `tfsdk:"read_path"`
	PutPath    types.String `tfsdk:"update_path"`
	DeletePath types.String `tfsdk:"destroy_path"`

	CreateMethod types.String `tfsdk:"create_method"`
	ReadMethod   types.String `tfsdk:"read_method"`
	UpdateMethod types.String `tfsdk:"update_method"`
	DeleteMethod types.String `tfsdk:"destroy_method"`

	QueryString types.String `tfsdk:"query_string"`
	ReadSearch  types.Object `tfsdk:"read_search"`

	ID          types.String `tfsdk:"id"`
	IDAttribute types.String `tfsdk:"id_attribute"`
	ObjectID    types.String `tfsdk:"object_id"`

	Data              types.String `tfsdk:"data"`
	UpdateData        types.String `tfsdk:"update_data"`
	DestroyData       types.String `tfsdk:"destroy_data"`
	APIResponse       types.Map    `tfsdk:"api_response"`
	APIResponseRaw    types.String `tfsdk:"api_response_raw"`
	CreateResponseRaw types.String `tfsdk:"create_response_raw"`
}

type ReadSearch struct {
	SearchKey   types.String `tfsdk:"search_key"`
	SearchValue types.String `tfsdk:"search_value"`
	ResultKey   types.String `tfsdk:"result_key"`
	QueryString types.String `tfsdk:"query_string"`
}

func (r *RestobjectResource) Metadata(
	_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_object"
}

func (r *RestobjectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Consider data sensitive if env variables is set to true.
	// isDataSensitive, _ := strconv.ParseBool(GetEnvOrDefault("API_DATA_IS_SENSITIVE", "false"))
	isDataSensitive := false

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Example resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Internal resource ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Optional: true,
			},
			"read_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be read (`GET`) on the API server. The string `{id}` is replaced by the terraform ID of " +
					"the object.",
				Optional: true,
			},
			"update_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be updated (`PUT`) on the API server. The string `{id}` is replaced by the terraform ID of " +
					"the object.",
				Optional: true,
			},
			"destroy_path": schema.StringAttribute{
				Description: "Defaults to `path/{id}`. The API path that specifies where objects of this type " +
					"can be deleted (`DELETE`) on the API server. The string `{id}` is replaced by the terraform ID of " +
					"the object.",
				Optional: true,
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `create_method` defined in the provider configuration. " +
					"Allows per-datasource override of `create_method` (see `create_method` provider documentation)",
				Optional: true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `read_method` defined in the provider configuration. " +
					"Allows per-datasource override of `read_method` (see `read_method` provider documentation)",
				Optional: true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `update_method` defined in the provider configuration. " +
					"Allows per-datasource override of `update_method` (see `update_method` provider documentation)",
				Optional: true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `destroy_method` defined in the provider configuration. " +
					"Allows per-datasource override of `destroy_method` (see `destroy_method` provider documentation)",
				Optional: true,
			},
			"id_attribute": schema.StringAttribute{
				Description: "Defaults to `id_attribute` defined in the provider configuration. " +
					"Allows per-datasource override of `id_attribute` (see `id_attribute` provider documentation)",
				Optional: true,
			},
			"object_id": schema.StringAttribute{
				Description: "Defaults to the autogenerated `id` gathered during normal operations and `id_attribute`. " +
					"Allows to set the ID manually. This is used in conjunction with the `*_path` attributes.",
				Optional: true,
			},
			"data": schema.StringAttribute{
				Description: "JSON object managed by the provider that holds information from the API response.",
				Required:    true,
				Sensitive:   isDataSensitive,
			},
			"read_search": schema.SingleNestedAttribute{
				Description: "Custom search for `read_path`.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"search_key": schema.StringAttribute{
						Description: "Key to identify a specific data record in the data array. " +
							"This should be a unique identifier e.g. `name`. Similar to `results_key`, " +
							"the value can have the format `path/to/key` to search for a nested object.",
						Optional: true,
					},
					"search_value": schema.StringAttribute{
						Description: "Value to compare with the value of `search_key` to determine whether " +
							"the correct object has been found. Example: If `search_key=name` and `search_value=foo`, " +
							"the record in the data array with the matching attribute `name=foo` is used.",
						Optional: true,
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
					"Currently the value is the golang fmt representation of the value. Simple primitives are set as expected, " +
					"but complex types like arrays and maps contain golang formatting.",
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
				Optional:    true,
				Description: "JSON object that is passed to update requests.",
				Sensitive:   isDataSensitive,
			},
			"destroy_data": schema.StringAttribute{
				Optional:    true,
				Description: "JSON object that is passed to destroy requests.",
				Sensitive:   isDataSensitive,
			},
		},
	}
}

func (r *RestobjectResource) Configure(
	_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*restclient.RestClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *restapi.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	r.client = client
}

//nolint:dupl
func (r *RestobjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RestobjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	objectOpts, diags := toObjectOptions(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ro, err := restobject.New(r.client, objectOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())

		return
	}

	if err := ro.Create(ctx); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())

		return
	}

	resp.Diagnostics.Append(mapFields(ctx, ro.Options, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

//nolint:dupl
func (r *RestobjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RestobjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	objectOpts, diags := toObjectOptions(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ro, err := restobject.New(r.client, objectOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())

		return
	}

	if err := ro.Read(ctx); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())

		return
	}

	resp.Diagnostics.Append(mapFields(ctx, ro.Options, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

//nolint:dupl
func (r *RestobjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RestobjectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	objectOpts, diags := toObjectOptions(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ro, err := restobject.New(r.client, objectOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())

		return
	}

	if err := ro.Update(ctx); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())

		return
	}

	resp.Diagnostics.Append(mapFields(ctx, ro.Options, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

//nolint:dupl
func (r *RestobjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RestobjectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	objectOpts, diags := toObjectOptions(ctx, data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ro, err := restobject.New(r.client, objectOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create API client", err.Error())

		return
	}

	if err := ro.Delete(ctx); err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())

		return
	}

	resp.Diagnostics.Append(mapFields(ctx, ro.Options, &data)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *RestobjectResource) ImportState(
	ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

//nolint:gocyclo,gocognit
func toObjectOptions(ctx context.Context, data RestobjectResourceModel) (*restobject.ObjectOptions, diag.Diagnostics) {
	objectOpts := &restobject.ObjectOptions{}
	objectOpts.ReadSearch = &restobject.ReadSearch{}

	readSearch := &ReadSearch{}
	diags := make(diag.Diagnostics, 0)

	if !(data.Path.IsNull() || data.Path.IsUnknown()) {
		objectOpts.Path = data.Path.ValueString()
	}

	if !(data.PostPath.IsNull() || data.PostPath.IsUnknown()) {
		objectOpts.PostPath = data.PostPath.ValueString()
	}

	if !(data.GetPath.IsNull() || data.GetPath.IsUnknown()) {
		objectOpts.GetPath = data.GetPath.ValueString()
	}

	if !(data.PutPath.IsNull() || data.PutPath.IsUnknown()) {
		objectOpts.PutPath = data.PutPath.ValueString()
	}

	if !(data.DeletePath.IsNull() || data.DeletePath.IsUnknown()) {
		objectOpts.DeletePath = data.DeletePath.ValueString()
	}

	if !(data.CreateMethod.IsNull() || data.CreateMethod.IsUnknown()) {
		objectOpts.CreateMethod = data.CreateMethod.ValueString()
	}

	if !(data.ReadMethod.IsNull() || data.ReadMethod.IsUnknown()) {
		objectOpts.ReadMethod = data.ReadMethod.ValueString()
	}

	if !(data.UpdateMethod.IsNull() || data.UpdateMethod.IsUnknown()) {
		objectOpts.UpdateMethod = data.UpdateMethod.ValueString()
	}

	if !(data.DeleteMethod.IsNull() || data.DeleteMethod.IsUnknown()) {
		objectOpts.DeleteMethod = data.DeleteMethod.ValueString()
	}

	if !(data.QueryString.IsNull() || data.QueryString.IsUnknown()) {
		objectOpts.QueryString = data.QueryString.ValueString()
	}

	// Parse nested struct from read_search
	if !(data.ReadSearch.IsNull() || data.ReadSearch.IsUnknown()) {
		asOpts := basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}
		diags.Append(data.ReadSearch.As(ctx, readSearch, asOpts)...)
	}

	if !(readSearch.SearchKey.IsNull() || readSearch.SearchKey.IsUnknown()) {
		objectOpts.ReadSearch.SearchKey = readSearch.SearchKey.ValueString()
	}

	if !(readSearch.SearchValue.IsNull() || readSearch.SearchValue.IsUnknown()) {
		objectOpts.ReadSearch.SearchValue = readSearch.SearchValue.ValueString()
	}

	if !(readSearch.ResultKey.IsNull() || readSearch.ResultKey.IsUnknown()) {
		objectOpts.ReadSearch.ResultKey = readSearch.ResultKey.ValueString()
	}

	if !(readSearch.QueryString.IsNull() || readSearch.QueryString.IsUnknown()) {
		objectOpts.ReadSearch.QueryString = readSearch.QueryString.ValueString()
	}

	if !(data.ID.IsNull() || data.ID.IsUnknown()) {
		objectOpts.ID = data.ID.ValueString()
	}

	if !(data.ObjectID.IsNull() || data.ObjectID.IsUnknown()) {
		objectOpts.ID = data.ObjectID.ValueString()
	}

	if !(data.IDAttribute.IsNull() || data.IDAttribute.IsUnknown()) {
		objectOpts.IDAttribute = data.IDAttribute.ValueString()
	}

	if !(data.IDAttribute.IsNull() || data.IDAttribute.IsUnknown()) {
		objectOpts.IDAttribute = data.IDAttribute.ValueString()
	}

	if !(data.Data.IsNull() || data.Data.IsUnknown()) {
		err := json.Unmarshal([]byte(data.Data.ValueString()), &objectOpts.Data)
		if err != nil {
			diags.AddError("Can not parse attribute", fmt.Sprintf("%s: %v", err, data.Data))
		}
	}

	return objectOpts, diags
}

func mapFields(ctx context.Context, opts *restobject.ObjectOptions, model *RestobjectResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(opts.ID)

	data, err := json.Marshal(opts.Data)
	if err != nil {
		diags.AddError("Can not map fields", fmt.Sprintf("%s: %v", err, opts.Data))
	}

	model.Data = types.StringValue(string(data))

	resp := make(map[string]string)
	for k, v := range opts.APIResponse {
		resp[k] = fmt.Sprintf("%v", v)
	}

	model.APIResponse, diags = types.MapValueFrom(ctx, types.StringType, resp)
	model.APIResponseRaw = types.StringValue(opts.APIResponseRaw)
	model.CreateResponseRaw = types.StringValue(opts.CreateResponseRaw)

	return diags
}
