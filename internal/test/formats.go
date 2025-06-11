package test

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"strings"
	"testing"
)

type TfCollectionValues interface {
	types.List | types.Set
}

type TfSingleValue interface {
	types.String | types.Bool | types.Int32 | types.Int64 | types.Float64
}

// FormatTerraformAttribute converts a Terraform Framework attribute value to an HCL string representation
func FormatTerraformAttribute(t *testing.T, value attr.Value, fieldName string) string {
	t.Helper()

	if value.IsNull() || value.IsUnknown() {
		return ""
	}

	if listValue, isList := value.(types.List); isList {
		return fmt.Sprintf("%s = [%s]", fieldName, testFormatTerraformCollectionValues(t, listValue))
	}
	if setValue, isSet := value.(types.Set); isSet {
		return fmt.Sprintf("%s = [%s]", fieldName, testFormatTerraformCollectionValues(t, setValue))
	}

	return fmt.Sprintf("%s = %s", fieldName, testFormatTerraformSingleValue(t, value))
}

func testFormatTerraformCollectionValues[T TfCollectionValues](t *testing.T, collection T) string {
	t.Helper()

	elements := make([]attr.Value, 0)
	switch c := any(collection).(type) {
	case types.List:
		elements = c.Elements()
	case types.Set:
		elements = c.Elements()
	default:
		t.Fatalf("Unsupported collection type: %T", c)
	}

	values := make([]string, 0, len(elements))
	for _, element := range elements {
		if element.IsNull() || element.IsUnknown() {
			continue
		}

		values = append(values, testFormatTerraformSingleValue(t, element))
	}
	return strings.Join(values, ", ")
}

func testFormatTerraformSingleValue(t *testing.T, value attr.Value) string {
	t.Helper()

	var result string
	switch v := any(value).(type) {
	case types.String:
		result = fmt.Sprintf(`"%s"`, v.ValueString())
	case types.Bool:
		result = fmt.Sprintf("%t", v.ValueBool())
	case types.Int32:
		result = fmt.Sprintf("%d", v.ValueInt32())
	case types.Int64:
		result = fmt.Sprintf("%d", v.ValueInt64())
	case types.Float64:
		result = fmt.Sprintf("%f", v.ValueFloat64())
	default:
		t.Fatalf("Unsupported single value type: %T", v)
	}

	return result
}
