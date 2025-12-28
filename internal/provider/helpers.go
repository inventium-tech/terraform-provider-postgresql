package provider

import (
	"fmt"
	"terraform-provider-postgresql/internal/pgclient"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type TerraformRequestObject interface {
	*resource.ConfigureRequest | *datasource.ConfigureRequest
}

func parsePgClientFromRequest[R datasource.ConfigureRequest | resource.ConfigureRequest](req R, destClient *pgclient.PostgresqlClient) diag.Diagnostic {
	var isValidType bool
	var diagnostic diag.Diagnostic

	// Extract provider data based on the request type
	var providerData any
	switch v := any(req).(type) {
	case datasource.ConfigureRequest:
		providerData = v.ProviderData
	case resource.ConfigureRequest:
		providerData = v.ProviderData
	default:
		return diag.NewErrorDiagnostic(
			msgErrorParsingProviderData,
			fmt.Sprintf(msgInvalidType, "datasource.ConfigureRequest or resource.ConfigureRequest", req),
		)
	}

	if *destClient, isValidType = providerData.(pgclient.PostgresqlClient); !isValidType {
		diagnostic = diag.NewErrorDiagnostic(
			msgErrorParsingProviderData,
			fmt.Sprintf(msgInvalidType, "pgclient.PostgresqlClient", providerData),
		)
	}

	return diagnostic
}

func setLastUpdatedFieldValue(destVal *types.String) {
	*destVal = types.StringValue(time.Now().Format(time.RFC3339))
}
