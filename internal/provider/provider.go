package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/thegeeklab/terraform-provider-restapi/internal/restapi/restclient"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure RestapiProvider satisfies various provider interfaces.
var _ provider.Provider = &RestapiProvider{}

// RestapiProvider defines the provider implementation.
type RestapiProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type RestapiProviderModel struct {
	Endpoint               types.String  `tfsdk:"endpoint"`
	Insecure               types.Bool    `tfsdk:"insecure"`
	Username               types.String  `tfsdk:"username"`
	Password               types.String  `tfsdk:"password"`
	Headers                types.Map     `tfsdk:"headers"`
	UseCookies             types.Bool    `tfsdk:"use_cookies"`
	Timeout                types.Int64   `tfsdk:"timeout"`
	IDAttribute            types.String  `tfsdk:"id_attribute"`
	CreateMethod           types.String  `tfsdk:"create_method"`
	ReadMethod             types.String  `tfsdk:"read_method"`
	UpdateMethod           types.String  `tfsdk:"update_method"`
	DestroyMethod          types.String  `tfsdk:"destroy_method"`
	CopyKeys               types.List    `tfsdk:"copy_keys"`
	ResponseFilter         types.Object  `tfsdk:"response_filter"`
	DriftDetection         types.Bool    `tfsdk:"drift_detection"`
	WriteReturnsObject     types.Bool    `tfsdk:"write_returns_object"`
	CreateReturnsObject    types.Bool    `tfsdk:"create_returns_object"`
	XSSIPrefix             types.String  `tfsdk:"xssi_prefix"`
	RateLimit              types.Float64 `tfsdk:"rate_limit"`
	TestPath               types.String  `tfsdk:"test_path"`
	OAuthClientCredentials types.Object  `tfsdk:"oauth_client_credentials"`
	CertString             types.String  `tfsdk:"cert_string"`
	KeyString              types.String  `tfsdk:"key_string"`
	CertFile               types.String  `tfsdk:"cert_file"`
	KeyFile                types.String  `tfsdk:"key_file"`
}

type OAuthClientCredentials struct {
	ClientID       types.String `tfsdk:"client_id"`
	ClientSecret   types.String `tfsdk:"client_secret"`
	TokenEndpoint  types.String `tfsdk:"token_endpoint"`
	EndpointParams types.Map    `tfsdk:"endpoint_params"`
	Scopes         types.List   `tfsdk:"scopes"`
}

type ResponseFilter struct {
	Keys    types.List `tfsdk:"keys"`
	Include types.Bool `tfsdk:"include"`
}

func (p *RestapiProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "restapi"
	resp.Version = p.version
}

func (p *RestapiProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:    true,
				Description: "Endpoint address of the REST API. This is used as base URL for all requests.",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "When using HTTPS, this disables TLS verification of the host.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "When set, will use this username for basic authentication to the API.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "When set, will use this password for basic authentication to the API.",
			},
			"headers": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				MarkdownDescription: "A mapping of header names and values to be set on all outgoing requests. This is useful " +
					"if you want to use a script via the `external` provider or provide an approved token " +
					"or change the default Content-Type from `application/json`. If username` and `password` " +
					"are set and Authorization is one of the headers defined here, the basic authentication data will " +
					"take precedence.",
			},
			"use_cookies": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable cookie jar to persist session.",
			},
			"timeout": schema.Int64Attribute{
				Optional:    true,
				Description: "When set, will cause requests taking longer than this time (in seconds) to be aborted.",
			},
			"id_attribute": schema.StringAttribute{
				Optional: true,
				Description: "If this option is set, it is used for editing REST objects. " +
					"For example, if the ID is set to `name`, changes to the API object are made to " +
					"`http://example.com/api/<value_of_name>`. This value can also be a path to the ID attribute " +
					"delimited by '/' if it is several levels deep in the data, e.g. `attributes/id` in the case " +
					"of an object `{ \"attributes\": { \"id\": 1234 }, \"config\": { \"name\": \"foo\", \"something\": \"bar\"}}`.",
			},
			"create_method": schema.StringAttribute{
				Description: "Defaults to `POST`. The HTTP method used to CREATE objects of this type on the API server.",
				Optional:    true,
			},
			"read_method": schema.StringAttribute{
				Description: "Defaults to `GET`. The HTTP method used to READ objects of this type on the API server.",
				Optional:    true,
			},
			"update_method": schema.StringAttribute{
				Description: "Defaults to `PUT`. The HTTP method used to UPDATE objects of this type on the API server.",
				Optional:    true,
			},
			"destroy_method": schema.StringAttribute{
				Description: "Defaults to `DELETE`. The HTTP method used to DELETE objects of this type on the API server.",
				Optional:    true,
			},
			"copy_keys": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Keys to copy from the API response to the `data` attribute. " +
					"This is useful if internal API information also needs to be provided for updates, " +
					"e.g. the revision of the object. Deactivates `drift_detection` implicitly.",
			},
			"response_filter": schema.SingleNestedAttribute{
				Description: "Filter configuration for the API response.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"keys": schema.ListAttribute{
						ElementType: types.StringType,
						Description: "List of top-level keys (nested keys are not supported) to be used for filtering.",
						Required:    true,
					},
					"include": schema.BoolAttribute{
						Description: "By default, the given `keys` are excluded from the API response. " +
							"This flag can be set to `true` if the `keys` should be used as include filter instead.",
						Optional: true,
					},
				},
			},
			"drift_detection": schema.BoolAttribute{
				Optional: true,
				Description: "Automatic detection of data drifts in the state. If activated, " +
					"all keys in the `data` attribute are searched for in the `api_response`. If the attributes " +
					"are found and the values differ from the current information in the `data` attribute, " +
					"these are updated automatically. Defaults to `true`.",
			},
			"write_returns_object": schema.BoolAttribute{
				Optional: true,
				Description: "Enable it if the API returns the created object on all write operations (`POST`, `PUT`). " +
					"The returned object is used by the provider to refresh internal data structures.",
			},
			"create_returns_object": schema.BoolAttribute{
				Optional: true,
				Description: "Enable it if the API returns the created object on creation operations only (`POST`). " +
					"The returned object is used by the provider to refresh internal data structures.",
			},
			"xssi_prefix": schema.StringAttribute{
				Optional:    true,
				Description: "Trim the XSSI prefix from response string, if present, before parsing.",
			},
			"rate_limit": schema.Float64Attribute{
				Optional:    true,
				Description: "Limits the number of requests per second sent to the API.",
			},
			"test_path": schema.StringAttribute{
				Optional: true,
				Description: "If this option is set, the provider will send a `read_method` request to this path " +
					"after instantiation and require a `200 OK` response before proceeding. " +
					"This is useful if your API provides a no-op endpoint that can signal whether this provider " +
					"is configured correctly. The response data is ignored.",
			},
			"oauth_client_credentials": schema.SingleNestedAttribute{
				Optional:    true,
				Description: "Configuration for OAuth client credential flow.",
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						Description: "Client ID.",
						Sensitive:   true,
						Required:    true,
					},
					"client_secret": schema.StringAttribute{
						Description: "Client secret.",
						Sensitive:   true,
						Required:    true,
					},
					"token_endpoint": schema.StringAttribute{
						Description: "Token endpoint.",
						Required:    true,
					},
					"scopes": schema.ListAttribute{
						ElementType: types.StringType,
						Optional:    true,
						Description: "Scopes",
					},
					"endpoint_params": schema.MapAttribute{
						Optional:    true,
						Description: "Additional key/values to pass to the OAuth client library as `EndpointParams`",
						ElementType: types.ListType{
							ElemType: types.StringType,
						},
					},
				},
			},
			"cert_string": schema.StringAttribute{
				Optional:    true,
				Description: "Client certificate string used for mTLS authentication.",
			},
			"key_string": schema.StringAttribute{
				Optional: true,
				Description: "Client certificate key string used for mTLS authentication. " +
					"Note that this mechanism simply delegates to `tls.LoadX509KeyPair` which does not support " +
					"passphrase protected private keys. The most robust security protection available for the " +
					"`key_file` is restrictive file system permissions.",
			},
			"cert_file": schema.StringAttribute{
				Optional:    true,
				Description: "Client certificate file used for mTLS authentication.",
			},
			"key_file": schema.StringAttribute{
				Optional: true,
				Description: "Client certificate key file used for mTLS authentication. " +
					"Note that this mechanism simply delegates to `tls.LoadX509KeyPair` which does not support " +
					"passphrase protected private keys. The most robust security protection available for the " +
					"`key_file` is restrictive file system permissions.",
			},
		},
	}
}

//nolint:gocyclo,gocognit,maintidx
func (p *RestapiProvider) Configure(
	ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse,
) {
	var data RestapiProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	respFilter := &ResponseFilter{}

	clientOpts := &restclient.ClientOptions{}
	clientOpts.ResponseFilter = &restclient.ResponseFilter{}

	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		clientOpts.Endpoint = data.Endpoint.ValueString()
	}

	if !data.Insecure.IsNull() && !data.Insecure.IsUnknown() {
		clientOpts.Insecure = data.Insecure.ValueBool()
	}

	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		clientOpts.Username = data.Username.ValueString()
	}

	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		clientOpts.Password = data.Password.ValueString()
	}

	if !data.Headers.IsNull() && !data.Headers.IsUnknown() {
		resp.Diagnostics.Append(data.Headers.ElementsAs(ctx, &clientOpts.Headers, false)...)
	}

	if !data.UseCookies.IsNull() && !data.UseCookies.IsUnknown() {
		clientOpts.UseCookies = data.UseCookies.ValueBool()
	}

	if !data.Timeout.IsNull() && !data.Timeout.IsUnknown() {
		clientOpts.Timeout = data.Timeout.ValueInt64()
	}

	if !data.IDAttribute.IsNull() && !data.IDAttribute.IsUnknown() {
		clientOpts.IDAttribute = data.IDAttribute.ValueString()
	}

	if !data.CreateMethod.IsNull() && !data.CreateMethod.IsUnknown() {
		clientOpts.CreateMethod = data.CreateMethod.ValueString()
	}

	if !data.ReadMethod.IsNull() && !data.ReadMethod.IsUnknown() {
		clientOpts.ReadMethod = data.ReadMethod.ValueString()
	}

	if !data.UpdateMethod.IsNull() && !data.UpdateMethod.IsUnknown() {
		clientOpts.UpdateMethod = data.UpdateMethod.ValueString()
	}

	if !data.DestroyMethod.IsNull() && !data.DestroyMethod.IsUnknown() {
		clientOpts.DestroyMethod = data.DestroyMethod.ValueString()
	}

	if !data.CopyKeys.IsNull() && !data.CopyKeys.IsUnknown() {
		resp.Diagnostics.Append(data.CopyKeys.ElementsAs(ctx, &clientOpts.CopyKeys, false)...)
	}

	if !data.ResponseFilter.IsNull() && !data.ResponseFilter.IsUnknown() {
		asOpts := basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}
		resp.Diagnostics.Append(data.ResponseFilter.As(ctx, respFilter, asOpts)...)
	}

	if !respFilter.Keys.IsNull() && !respFilter.Keys.IsUnknown() {
		resp.Diagnostics.Append(respFilter.Keys.ElementsAs(ctx, &clientOpts.ResponseFilter.Keys, false)...)
	}

	if !respFilter.Include.IsNull() && !respFilter.Include.IsUnknown() {
		clientOpts.ResponseFilter.Include = respFilter.Include.ValueBool()
	}

	clientOpts.DriftDetection = true
	if !data.DriftDetection.IsNull() && !data.DriftDetection.IsUnknown() {
		clientOpts.DriftDetection = data.DriftDetection.ValueBool()
	}

	if !data.WriteReturnsObject.IsNull() && !data.WriteReturnsObject.IsUnknown() {
		clientOpts.WriteReturnsObject = data.WriteReturnsObject.ValueBool()
	}

	if !data.CreateReturnsObject.IsNull() && !data.CreateReturnsObject.IsUnknown() {
		clientOpts.CreateReturnsObject = data.CreateReturnsObject.ValueBool()
	}

	if !data.XSSIPrefix.IsNull() && !data.XSSIPrefix.IsUnknown() {
		clientOpts.XSSIPrefix = data.XSSIPrefix.ValueString()
	}

	if !data.RateLimit.IsNull() && !data.RateLimit.IsUnknown() {
		clientOpts.RateLimit = data.RateLimit.ValueFloat64()
	}

	if !data.TestPath.IsNull() && !data.TestPath.IsUnknown() {
		clientOpts.TestPath = data.TestPath.ValueString()
	}

	if !data.OAuthClientCredentials.IsNull() && !data.OAuthClientCredentials.IsUnknown() {
		clientOpts.OAuthClientCredentials = toOAuthCredentials(ctx, data.OAuthClientCredentials)
	}

	if !data.CertString.IsNull() && !data.CertString.IsUnknown() {
		clientOpts.CertString = data.CertString.ValueString()
	}

	if !data.KeyString.IsNull() && !data.KeyString.IsUnknown() {
		clientOpts.KeyString = data.KeyString.ValueString()
	}

	if !data.CertFile.IsNull() && !data.CertFile.IsUnknown() {
		clientOpts.CertFile = data.CertFile.ValueString()
	}

	if !data.KeyFile.IsNull() && !data.KeyFile.IsUnknown() {
		clientOpts.KeyFile = data.KeyFile.ValueString()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := restclient.New(ctx, clientOpts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create API client",
			fmt.Sprintf("... details ... %s", err),
		)

		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RestapiProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewRestobjectResource,
	}
}

func (p *RestapiProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRestobjectDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RestapiProvider{
			version: version,
		}
	}
}

func toOAuthCredentials(ctx context.Context, credentials types.Object) *restclient.OAuthCredentials {
	credentialsMap := &OAuthClientCredentials{}
	oauthCredentials := &restclient.OAuthCredentials{}

	if !credentials.IsNull() || !credentials.IsUnknown() {
		credentials.As(ctx, credentialsMap, basetypes.ObjectAsOptions{})
	}

	endpointParams := url.Values{}

	if !credentialsMap.EndpointParams.IsNull() {
		endpointParamsMap := make(map[string][]string, 0)

		credentialsMap.EndpointParams.ElementsAs(ctx, &endpointParamsMap, false)

		for k, vals := range endpointParams {
			for _, val := range vals {
				endpointParams.Add(k, val)
			}
		}
	}

	if credentialsMap.ClientID.IsNull() && credentialsMap.ClientSecret.IsNull() && credentialsMap.TokenEndpoint.IsNull() {
		return oauthCredentials
	}

	if !credentials.IsNull() && !credentials.IsUnknown() {
		credentialsMap.Scopes.ElementsAs(ctx, &oauthCredentials.Scopes, false)
	}

	oauthCredentials.ClientID = credentialsMap.ClientID.ValueString()
	oauthCredentials.ClientSecret = credentialsMap.ClientSecret.ValueString()
	oauthCredentials.TokenEndpoint = credentialsMap.TokenEndpoint.ValueString()
	oauthCredentials.EndpointParams = endpointParams

	return oauthCredentials
}
