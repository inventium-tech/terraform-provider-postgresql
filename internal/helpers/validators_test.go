package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSafeValidator(t *testing.T) {
	t.Run("returns non-nil validator", func(t *testing.T) {
		validator := GetSafeValidator()
		assert.NotNil(t, validator, "GetSafeValidator() should return a non-nil validator")
	})

	t.Run("returns singleton instance", func(t *testing.T) {
		// Get validator instance twice
		validator1 := GetSafeValidator()
		validator2 := GetSafeValidator()

		// Both calls should return the same instance (pointer equality)
		assert.Same(t, validator1, validator2, "GetSafeValidator() should return the same instance on subsequent calls")
	})

	t.Run("validator has required struct validation enabled", func(t *testing.T) {
		// Define a test struct with a required field
		type TestStruct struct {
			RequiredField string `validate:"required"`
		}

		validator := GetSafeValidator()

		// Test with missing required field
		err := validator.Struct(TestStruct{RequiredField: ""})
		assert.Error(t, err, "Validation should fail for empty required field")

		// Test with required field present
		err = validator.Struct(TestStruct{RequiredField: "value"})
		assert.NoError(t, err, "Validation should pass when required field has a value")
	})
}

func TestGetUnsafeValidator(t *testing.T) {
	t.Run("returns non-nil validator", func(t *testing.T) {
		validator := GetUnsafeValidator()
		assert.NotNil(t, validator, "GetUnsafeValidator() should return a non-nil validator")
	})

	t.Run("returns singleton instance", func(t *testing.T) {
		// Get validator instance twice
		validator1 := GetUnsafeValidator()
		validator2 := GetUnsafeValidator()

		// Both calls should return the same instance (pointer equality)
		assert.Same(t, validator1, validator2, "GetUnsafeValidator() should return the same instance on subsequent calls")
	})

	t.Run("validator has private field validation enabled", func(t *testing.T) {
		// Define a test struct with a private field
		type TestStruct struct {
			publicField  string `validate:"required"`
			privateField string `validate:"required"`
		}

		validator := GetUnsafeValidator()

		// Create a struct with empty private field
		testStruct := TestStruct{
			publicField:  "value",
			privateField: "",
		}

		// Validation should fail because the private field is empty and required
		err := validator.Struct(testStruct)
		assert.Error(t, err, "Validation should fail for empty private required field")

		// Set the private field
		testStruct.privateField = "value"

		// Validation should now pass
		err = validator.Struct(testStruct)
		assert.NoError(t, err, "Validation should pass when private field has a value")
	})
}
