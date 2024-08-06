package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapPgModelToTerraformModel_Success(t *testing.T) {

	type Source struct {
		Field1 string
		Field2 int
		Field3 bool
	}
	type Destination struct {
		Field1 types.String
		Field2 types.Int64
		Field3 types.Bool
	}

	testMatrix := []struct {
		name          string
		src           interface{}
		dest          interface{}
		customAssigns map[string]any
		wantErr       bool
	}{
		{
			name: "SuccessMap",
			src: Source{
				Field1: "test",
				Field2: 123,
				Field3: true,
			},
			dest:          &Destination{},
			customAssigns: map[string]any{},
		},
	}

	for _, tt := range testMatrix {
		t.Run(tt.name, func(t *testing.T) {
			src := tt.src.(Source)
			dest := tt.dest.(*Destination)
			err := mapPgModelToTerraformModel(src, dest, tt.customAssigns)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, types.StringValue(src.Field1), dest.Field1)
			assert.Equal(t, types.Int64Value(int64(src.Field2)), dest.Field2)
			assert.Equal(t, types.BoolValue(src.Field3), dest.Field3)
		})
	}
}

func TestMapSetValueToSlice(t *testing.T) {
	mockStringElements := []string{"value1", "value2"}
	mockInt32Elements := []int32{1, 2}
	mockInt64Elements := []int64{1, 2}
	mockBoolElements := []bool{true, false}
	ctx := context.TODO()

	t.Run("SuccessMapStringValues", func(t *testing.T) {
		set, diags := types.SetValueFrom(ctx, types.StringType, mockStringElements)
		assert.Empty(t, diags)

		result := mapSetValueToSlice[string](set)
		assert.Equal(t, mockStringElements, result)
	})

	t.Run("SuccessMapInt32Values", func(t *testing.T) {
		set, diags := types.SetValueFrom(ctx, types.Int32Type, mockInt32Elements)
		assert.Empty(t, diags)

		result := mapSetValueToSlice[int32](set)
		assert.Equal(t, mockInt32Elements, result)
	})

	t.Run("SuccessMapInt64Values", func(t *testing.T) {
		set, diags := types.SetValueFrom(ctx, types.Int64Type, mockInt64Elements)
		assert.Empty(t, diags)

		result := mapSetValueToSlice[int64](set)
		assert.Equal(t, mockInt64Elements, result)
	})

	t.Run("SuccessMapBoolValues", func(t *testing.T) {
		set, diags := types.SetValueFrom(ctx, types.BoolType, mockBoolElements)
		assert.Empty(t, diags)

		result := mapSetValueToSlice[bool](set)
		assert.Equal(t, mockBoolElements, result)
	})
}
