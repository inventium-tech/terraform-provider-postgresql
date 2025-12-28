package provider

import (
	"fmt"
	"strings"
	"terraform-provider-postgresql/internal/helpers"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

type postgresqlUserFunctionModel struct {
	Id           types.String                    `tfsdk:"id"`
	Name         types.String                    `tfsdk:"name"`
	Args         []postgresqlUserFunctionArgType `tfsdk:"args"`
	Returns      types.String                    `tfsdk:"returns"`
	Body         types.String                    `tfsdk:"body"`
	Language     types.String                    `tfsdk:"language"`
	AllowReplace types.Bool                      `tfsdk:"allow_replace"`
	Database     types.String                    `tfsdk:"database"`
	Schema       types.String                    `tfsdk:"schema"`
	Owner        types.String                    `tfsdk:"owner"`
	Comment      types.String                    `tfsdk:"comment"`
	LastUpdated  types.String                    `tfsdk:"last_updated"`
}

type postgresqlUserFunctionArgType struct {
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Mode    types.String `tfsdk:"mode"`    // Optional, e.g., IN, OUT, INOUT & VARIADIC
	Default types.String `tfsdk:"default"` // Optional, default value for the argument
}

func (f *postgresqlUserFunctionModel) SetId() {
	argsString := helpers.SliceReduce(f.Args, func(acc string, arg postgresqlUserFunctionArgType) string {
		if acc == "" {
			return arg.String()
		}
		return fmt.Sprintf("%s, %s", acc, arg.String())
	})
	f.Id = types.StringValue(fmt.Sprintf("%s.%s.%s(%s)", f.Database.ValueString(), f.Schema.ValueString(), f.Name.ValueString(), argsString))
}

func (a *postgresqlUserFunctionArgType) String() string {
	if a == nil {
		return ""
	}
	var parts []string
	modeValue := a.Mode.ValueString()
	if !a.Mode.IsNull() && modeValue != "" && modeValue != "IN" {
		parts = append(parts, modeValue)
	}
	if !a.Name.IsNull() && a.Name.ValueString() != "" {
		parts = append(parts, strings.ToLower(a.Name.ValueString()))
	}
	if !a.Type.IsNull() && a.Type.ValueString() != "" {
		parts = append(parts, strings.ToLower(a.Type.ValueString()))
	}
	if !a.Default.IsNull() && a.Default.ValueString() != "" {
		parts = append(parts, "DEFAULT")
		parts = append(parts, strings.ToLower(a.Default.ValueString()))
	}
	return strings.Join(parts, " ")

}
