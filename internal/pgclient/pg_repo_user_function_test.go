package pgclient

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"terraform-provider-postgresql/internal/helpers"
	"terraform-provider-postgresql/internal/test"
	"testing"
)

// Unit tests for helper functions

func TestExtractFunctionBody(t *testing.T) {
	tests := []struct {
		name        string
		functionDef string
		expected    string
		wantErr     bool
	}{
		{
			name:        "Extract body from valid function definition",
			functionDef: "CREATE OR REPLACE FUNCTION public.test_function(arg1 TEXT)\n RETURNS TEXT\n LANGUAGE plpgsql\nAS $function$\nBEGIN RETURN 'Hello, arg1!'; END;\n$function$",
			expected:    "BEGIN RETURN 'Hello, arg1!'; END;",
			wantErr:     false,
		},
		{
			name:        "Extract body with multiple lines",
			functionDef: "CREATE OR REPLACE FUNCTION public.test_function(arg1 TEXT)\n RETURNS TEXT\n LANGUAGE plpgsql\nAS $function$\nBEGIN\n  RETURN 'Hello, ' || arg1;\nEND;\n$function$",
			expected:    "BEGIN\n  RETURN 'Hello, ' || arg1;\nEND;",
			wantErr:     false,
		},
		{
			name:        "Function definition without body",
			functionDef: "CREATE OR REPLACE FUNCTION public.test_function(arg1 TEXT)\n RETURNS TEXT\n LANGUAGE plpgsql\nAS",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "Empty function definition",
			functionDef: "",
			expected:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFunctionBody(tt.functionDef)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		rawName  string
		expected string
		wantErr  bool
	}{
		{
			name:     "Extract name from function with arguments",
			rawName:  "test_function(arg1 TEXT)",
			expected: "test_function",
			wantErr:  false,
		},
		{
			name:     "Extract name from function without arguments",
			rawName:  "test_function()",
			expected: "test_function",
			wantErr:  false,
		},
		{
			name:     "Extract name from function without parentheses",
			rawName:  "test_function",
			expected: "test_function",
			wantErr:  false,
		},
		{
			name:     "Empty function name",
			rawName:  "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFunctionName(tt.rawName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractFunctionArgsParsed(t *testing.T) {
	tests := []struct {
		name     string
		rawName  string
		expected []PgFunctionArgType
		wantErr  bool
	}{
		{
			name:    "Extract args from function with single argument",
			rawName: "test_function(arg1 TEXT)",
			expected: []PgFunctionArgType{
				{Name: "arg1", Type: "TEXT"},
			},
			wantErr: false,
		},
		{
			name:    "Extract args from function with multiple arguments",
			rawName: "test_function(arg1 TEXT, arg2 INTEGER, arg3 BOOLEAN DEFAULT false)",
			expected: []PgFunctionArgType{
				{Name: "arg1", Type: "TEXT"},
				{Name: "arg2", Type: "INTEGER"},
				{Name: "arg3", Type: "BOOLEAN", Default: "false"},
			},
			wantErr: false,
		},
		{
			name:    "Extract args from function with argument modes",
			rawName: "test_function(IN arg1 TEXT, OUT arg2 INTEGER, INOUT arg3 TEXT)",
			expected: []PgFunctionArgType{
				{Name: "arg1", Type: "TEXT", Mode: "IN"},
				{Name: "arg2", Type: "INTEGER", Mode: "OUT"},
				{Name: "arg3", Type: "TEXT", Mode: "INOUT"},
			},
			wantErr: false,
		},
		{
			name:     "Function without arguments",
			rawName:  "test_function()",
			expected: []PgFunctionArgType{},
			wantErr:  false,
		},
		{
			name:     "Function without parentheses",
			rawName:  "test_function",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Empty function name",
			rawName:  "",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFunctionArgsParsed(tt.rawName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(result))

				for i, expected := range tt.expected {
					if i < len(result) {
						assert.Equal(t, expected.Name, result[i].Name)
						assert.Equal(t, expected.Type, result[i].Type)
						assert.Equal(t, expected.Mode, result[i].Mode)
						assert.Equal(t, expected.Default, result[i].Default)
					}
				}
			}
		})
	}
}

func TestExtractFunctionArgsFull(t *testing.T) {
	tests := []struct {
		name     string
		rawName  string
		expected string
		wantErr  bool
	}{
		{
			name:     "Extract full args from function with single argument",
			rawName:  "test_function(arg1 TEXT)",
			expected: "arg1 text",
			wantErr:  false,
		},
		{
			name:     "Extract full args from function with multiple arguments",
			rawName:  "test_function(arg1 TEXT, arg2 INTEGER, arg3 BOOLEAN DEFAULT false)",
			expected: "arg1 text, arg2 integer, arg3 boolean DEFAULT false",
			wantErr:  false,
		},
		{
			name:     "Extract full args from function with argument modes",
			rawName:  "test_function(IN arg1 TEXT, OUT arg2 INTEGER, INOUT arg3 TEXT)",
			expected: "arg1 text, OUT arg2 integer, INOUT arg3 text",
			wantErr:  false,
		},
		{
			name:     "Function without arguments",
			rawName:  "test_function()",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "Function without parentheses",
			rawName:  "test_function",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty function name",
			rawName:  "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFunctionArgsFull(tt.rawName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestExtractFunctionArgsTypes(t *testing.T) {
	tests := []struct {
		name     string
		rawName  string
		expected string
		wantErr  bool
	}{
		{
			name:     "Extract arg types from function with single argument",
			rawName:  "test_function(arg1 TEXT)",
			expected: "TEXT",
			wantErr:  false,
		},
		{
			name:     "Extract arg types from function with multiple arguments",
			rawName:  "test_function(arg1 TEXT, arg2 INTEGER, arg3 BOOLEAN DEFAULT false)",
			expected: "TEXT, INTEGER, BOOLEAN",
			wantErr:  false,
		},
		{
			name:     "Extract arg types from function with argument modes",
			rawName:  "test_function(IN arg1 TEXT, OUT arg2 INTEGER, INOUT arg3 TEXT)",
			expected: "TEXT, INTEGER, TEXT",
			wantErr:  false,
		},
		{
			name:     "Function without arguments",
			rawName:  "test_function()",
			expected: "",
			wantErr:  false,
		},
		{
			name:     "Function without parentheses",
			rawName:  "test_function",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Empty function name",
			rawName:  "",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractFunctionArgsTypes(tt.rawName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Integration tests for UserFunctionRepo

func TestUserFunctionRepo_Integration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	var (
		err    error
		exists bool
		model  *UserFunctionModel
	)

	runOpts := test.PostgresContainerRunOptions{
		Database: "test_user_function_db",
		Username: "test_user_function_user",
		Password: "test_password",
	}

	pgClient := loadTestPostgresqlClient(t, runOpts)
	pgConn, err := pgClient.GetConnection(t.Context())
	assert.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
	})

	ctx := t.Context()
	repo := NewUserFunctionRepo()

	// Test function parameters
	testFuncParams := UserFunctionCreateParams{
		Name:     "test_function",
		Args:     []PgFunctionArgType{{Name: "arg1", Type: "TEXT"}},
		Returns:  "TEXT",
		Language: "plpgsql",
		Body:     "BEGIN RETURN 'Hello, ' || arg1; END;",
		Schema:   "public",
		Comment:  "Test function",
	}

	// Test Create
	t.Run("Create function", func(t *testing.T) {
		err = repo.Create(ctx, pgConn, testFuncParams)
		assert.NoError(t, err)
	})

	// Test Exists
	t.Run("Function exists", func(t *testing.T) {
		exists, err = repo.Exists(ctx, pgConn, "public.test_function(arg1 TEXT)")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test GetOne
	t.Run("Get function", func(t *testing.T) {
		model, err = repo.GetOne(ctx, pgConn, "public.test_function(arg1 TEXT)")
		assert.NoError(t, err)
		assert.NotNil(t, model)
		assert.Equal(t, "test_function", model.Name.String)
		assert.Equal(t, "text", model.Returns.String)
		assert.Equal(t, "plpgsql", model.Language.String)
		assert.Equal(t, "public", model.Schema.String)
		assert.Equal(t, runOpts.Username, model.Owner.String)
		assert.Equal(t, "Test function", model.Comment.String)
	})

	// Test Update
	t.Run("Update function", func(t *testing.T) {
		// Update the function name
		newName := "updated_function"
		updateParams := UserFunctionUpdateParams{
			Name: &newName,
		}

		err = repo.Update(ctx, pgConn, "public.test_function(arg1 TEXT)", updateParams)
		assert.NoError(t, err)

		// Verify original function no longer exists
		exists, err = repo.Exists(ctx, pgConn, "public.test_function(arg1 TEXT)")
		assert.NoError(t, err)
		assert.False(t, exists)

		// Verify function exists with new name
		updatedFuncId := "public.updated_function(arg1 TEXT)"
		exists, err = repo.Exists(ctx, pgConn, updatedFuncId)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Update the function comment
		newComment := "Updated comment"
		updateCommentParams := UserFunctionUpdateParams{
			Comment: &newComment,
		}

		err = repo.Update(ctx, pgConn, updatedFuncId, updateCommentParams)
		assert.NoError(t, err)

		// Verify comment was updated
		model, err = repo.GetOne(ctx, pgConn, updatedFuncId)
		assert.NoError(t, err)
		assert.Equal(t, "updated_function", model.Name.String)
		assert.Equal(t, "Updated comment", model.Comment.String)

		// Clean up
		err = repo.Drop(ctx, pgConn, updatedFuncId)
		assert.NoError(t, err)
	})

	// Test function with multiple arguments
	t.Run("Function with multiple arguments", func(t *testing.T) {
		multiArgParams := UserFunctionCreateParams{
			Name: "multi_arg_function",
			Args: []PgFunctionArgType{
				{Name: "arg1", Type: "TEXT"},
				{Name: "arg2", Type: "INTEGER"},
				{Name: "arg3", Type: "BOOLEAN", Default: "false"},
			},
			Returns:  "TEXT",
			Language: "plpgsql",
			Body:     "BEGIN RETURN 'Args: ' || arg1 || ', ' || arg2::text || ', ' || arg3::text; END;",
			Schema:   "public",
		}

		mockArgsStr := helpers.SliceReduce(multiArgParams.Args, func(result string, arg PgFunctionArgType) string {
			if result == "" {
				return arg.String()
			}
			return fmt.Sprintf("%s,%s", result, arg.String())
		})
		mockFuncSignature := fmt.Sprintf("%s.%s(%s)", multiArgParams.Schema, multiArgParams.Name, mockArgsStr)

		// Create function
		err = repo.Create(ctx, pgConn, multiArgParams)
		assert.NoError(t, err)

		// Verify function exists
		exists, err = repo.Exists(ctx, pgConn, mockFuncSignature)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Get function
		model, err = repo.GetOne(ctx, pgConn, mockFuncSignature)
		assert.NoError(t, err)
		assert.NotNil(t, model)

		// Drop function
		err = repo.Drop(ctx, pgConn, mockFuncSignature)
		assert.NoError(t, err)
	})

	// Test function with different argument modes (IN, OUT, INOUT)
	t.Run("Function with argument modes", func(t *testing.T) {
		modeArgParams := UserFunctionCreateParams{
			Name: "mode_arg_function",
			Args: []PgFunctionArgType{
				{Name: "in_arg", Type: "TEXT", Mode: "IN"},
				{Name: "out_arg", Type: "INTEGER", Mode: "OUT"},
				{Name: "inout_arg", Type: "TEXT", Mode: "INOUT"},
			},
			Returns:  "RECORD",
			Language: "plpgsql",
			Body:     "BEGIN out_arg := length(in_arg); inout_arg := upper(inout_arg); END;",
			Schema:   "public",
		}

		mockArgsStr := helpers.SliceReduce(modeArgParams.Args, func(result string, arg PgFunctionArgType) string {
			if result == "" {
				return arg.String()
			}
			return fmt.Sprintf("%s,%s", result, arg.String())
		})
		mockFuncSignature := fmt.Sprintf("%s.%s(%s)", modeArgParams.Schema, modeArgParams.Name, mockArgsStr)

		// Create function
		err = repo.Create(ctx, pgConn, modeArgParams)
		assert.NoError(t, err)

		// Verify function exists
		exists, err = repo.Exists(ctx, pgConn, mockFuncSignature)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Drop function
		err = repo.Drop(ctx, pgConn, mockFuncSignature)
		assert.NoError(t, err)
	})
}
