package validators

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"regexp"
)

var _ validator.String = postgresqlObjectNameValidator{}

type postgresqlObjectNameValidator struct{}

func (v postgresqlObjectNameValidator) Description(ctx context.Context) string {
	return "postgresql object name must match the regex ^[a-zA-Z_][a-zA-Z0-9_]*$"
}

func (v postgresqlObjectNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v postgresqlObjectNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, res *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	pgObjectNameRE := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	value := req.ConfigValue.ValueString()

	if !pgObjectNameRE.MatchString(value) {
		res.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
			req.Path,
			v.Description(ctx),
			value,
		))
	}
}

func PostgresqlObjectName() validator.String {
	return postgresqlObjectNameValidator{}
}
