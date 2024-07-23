package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"strings"
	"terraform-provider-postgresql/internal/client"
)

func standardDataSourceConfigure[R datasource.ConfigureRequest | resource.ConfigureRequest](ctx context.Context, req R) (client.PGClient, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	var providerData any

	switch any(req).(type) {
	case datasource.ConfigureRequest:
		if reqCast, ok := any(req).(datasource.ConfigureRequest); ok {
			providerData = reqCast.ProviderData
		}
	case resource.ConfigureRequest:
		if reqCast, ok := any(req).(resource.ConfigureRequest); ok {
			providerData = reqCast.ProviderData
		}
	default:
		diags.AddError(
			"Unexpected Type for the req parameter",
			"Expected type 'datasource.ConfigureRequest' or 'resource.ConfigureRequest'",
		)
		return nil, diags
	}

	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if providerData == nil {
		return nil, diags
	}

	pgClient, ok := providerData.(client.PGClient)
	if !ok {
		diags.AddError(
			"Unexpected Configure type for Client",
			fmt.Sprintf("Expected *postgresClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, diags
	}
	clientConfig := pgClient.GetConfig()
	ctx = tflog.SetField(ctx, "pg_host", clientConfig.Host)
	ctx = tflog.SetField(ctx, "pg_username", clientConfig.Username)
	ctx = tflog.SetField(ctx, "pg_password", clientConfig.Password)
	ctx = tflog.SetField(ctx, "pg_database", clientConfig.Database)
	tflog.MaskFieldValuesWithFieldKeys(ctx, "pg_password")

	tflog.Info(ctx, "Loaded Client")

	return pgClient, diags
}

func sliceToStringSet[T interface{} | string](arr []T) string {
	var strSet []string
	for _, v := range arr {
		strSet = append(strSet, fmt.Sprintf(`"%v"`, v))
	}
	return fmt.Sprintf(`[%v]`, strings.Join(strSet, ", "))
}
