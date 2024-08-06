package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"reflect"
	"strings"
	"terraform-provider-postgresql/internal/client"
)

// common error messages
const (
	msgErrGetPgConnection = "Error establishing a PostgreSQL connection"
	msgErrMapPgModel      = "Error mapping Postgres model to Terraform model"
)

func parsePgClientFromRequest[R datasource.ConfigureRequest | resource.ConfigureRequest](ctx context.Context, req R) (client.PgClient, diag.Diagnostics) {
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
	pgClient, ok := providerData.(client.PgClient)
	if !ok {
		diags.AddError(
			"Unable to parse postgres client from request",
			fmt.Sprintf("Expected *client.PgClient, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil, diags
	}

	initConfig := pgClient.GetInitConfig()
	ctx = tflog.SetField(ctx, "pg_host", initConfig.Host)
	ctx = tflog.SetField(ctx, "pg_username", initConfig.Username)
	ctx = tflog.SetField(ctx, "pg_database", initConfig.Database)
	tflog.Info(ctx, "Postgres client parsed from request")

	return pgClient, diags

}

func sliceToTerraformSetString[T interface{} | string](arr []T) string {
	var strSet []string
	for _, v := range arr {
		strSet = append(strSet, fmt.Sprintf(`"%v"`, v))
	}
	return fmt.Sprintf(`[%v]`, strings.Join(strSet, ", "))
}

func mapSetValueToSlice[T string | int | int32 | int64 | bool](set basetypes.SetValue) []T {
	var slice []T
	var parsedElem any
	for _, elem := range set.Elements() {
		switch elem.(type) {
		case basetypes.StringValue:
			parsedElem = any(elem).(basetypes.StringValue).ValueString()
		case basetypes.Int32Value:
			parsedElem = any(elem).(basetypes.Int32Value).ValueInt32()
		case basetypes.Int64Value:
			parsedElem = any(elem).(basetypes.Int64Value).ValueInt64()
		case basetypes.BoolValue:
			parsedElem = any(elem).(basetypes.BoolValue).ValueBool()
		}

		slice = append(slice, parsedElem.(T))
	}
	return slice
}

func mapPgModelToTerraformModel(src, dest interface{}, customAssign map[string]any) error {
	srcVal := reflect.ValueOf(src)
	destVal := reflect.ValueOf(dest)

	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if destVal.Kind() == reflect.Ptr {
		destVal = destVal.Elem()
	}

	if srcVal.Kind() != reflect.Struct || destVal.Kind() != reflect.Struct {
		return fmt.Errorf("both parameters should be structs or pointers to structs, src: %s, dest: %s", srcVal.Kind(), destVal.Kind())
	}

	for i := 0; i < srcVal.NumField(); i++ {
		srcField := srcVal.Field(i)
		srcFieldName := srcVal.Type().Field(i).Name

		destField := destVal.FieldByName(srcFieldName)
		if !destField.IsValid() || !destField.CanSet() {
			continue
		}

		if assignVal, ok := customAssign[srcFieldName]; ok {
			destField.Set(reflect.ValueOf(assignVal))
			continue
		}

		var srcFieldValue interface{}
		var diagErr diag.Diagnostics

		switch destField.Type() {
		case reflect.TypeOf(basetypes.StringValue{}):
			srcFieldValue = types.StringValue(srcField.String())
		case reflect.TypeOf(basetypes.Int64Value{}):
			srcFieldValue = types.Int64Value(srcField.Int())
		case reflect.TypeOf(basetypes.BoolValue{}):
			srcFieldValue = types.BoolValue(srcField.Bool())
		case reflect.TypeOf(basetypes.SetValue{}):
			setValue := destField.Interface().(basetypes.SetValue)
			srcFieldValue, diagErr = types.SetValueFrom(context.TODO(), setValue.ElementType(nil), srcField.Interface())
			if diagErr.HasError() {
				return fmt.Errorf(diagErr[0].Summary())
			}
		default:
			return fmt.Errorf("field types do not match")
		}

		destField.Set(reflect.ValueOf(srcFieldValue))
	}

	return nil
}
