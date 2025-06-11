package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanUpSlice(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		// Test with string slice containing duplicates and empty strings
		// CleanUpSlice removes adjacent duplicates and empty strings
		input := []string{"a", "b", "a", "", "c", ""}
		expected := []string{"a", "b", "a", "c"} // Non-adjacent duplicates are preserved
		result := CleanUpSlice(input)
		assert.Equal(t, expected, result)
	})

	t.Run("int slice", func(t *testing.T) {
		// Test with int slice containing duplicates and zeros
		// CleanUpSlice removes adjacent duplicates and zeros
		input := []int{1, 2, 1, 0, 3, 0}
		expected := []int{1, 2, 1, 3} // Non-adjacent duplicates are preserved
		result := CleanUpSlice(input)
		assert.Equal(t, expected, result)
	})

	t.Run("bool slice", func(t *testing.T) {
		// Test with bool slice containing duplicates and false values
		// CleanUpSlice removes adjacent duplicates and false values
		input := []bool{true, false, true, false}
		expected := []bool{true, true} // Non-adjacent duplicates are preserved
		result := CleanUpSlice(input)
		assert.Equal(t, expected, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		// Test with empty slice
		var input []string
		result := CleanUpSlice(input)
		assert.Equal(t, input, result)
	})

	t.Run("int64 slice", func(t *testing.T) {
		// Test with int64 slice
		// CleanUpSlice removes adjacent duplicates and zeros
		input := []int64{1, 0, 2, 0, 1, 3}
		expected := []int64{1, 2, 1, 3} // Non-adjacent duplicates are preserved
		result := CleanUpSlice(input)
		assert.Equal(t, expected, result)
	})
}

func TestSliceReduce(t *testing.T) {
	t.Run("sum integers", func(t *testing.T) {
		// Test reducing a slice of integers to their sum
		input := []int{1, 2, 3, 4, 5}
		result := SliceReduce(input, func(acc, val int) int {
			return acc + val
		})
		assert.Equal(t, 15, result)
	})

	t.Run("sum integers with initial value", func(t *testing.T) {
		// Test reducing with an initial value
		input := []int{1, 2, 3, 4, 5}
		result := SliceReduce(input, func(acc, val int) int {
			return acc + val
		}, 10)
		assert.Equal(t, 25, result)
	})

	t.Run("concatenate strings", func(t *testing.T) {
		// Test reducing a slice of strings to a concatenated string
		input := []string{"a", "b", "c"}
		result := SliceReduce(input, func(acc, val string) string {
			return acc + val
		})
		assert.Equal(t, "abc", result)
	})

	t.Run("empty slice", func(t *testing.T) {
		// Test with empty slice
		var input []int
		result := SliceReduce(input, func(acc, val int) int {
			return acc + val
		})
		assert.Equal(t, 0, result)
	})
}

func TestSliceMap(t *testing.T) {
	t.Run("double integers", func(t *testing.T) {
		// Test mapping a slice of integers to their doubled values
		input := []int{1, 2, 3, 4, 5}
		expected := []int{2, 4, 6, 8, 10}
		result := SliceMap(input, func(val int) int {
			return val * 2
		})
		assert.Equal(t, expected, result)
	})

	t.Run("convert int to string", func(t *testing.T) {
		// Test mapping a slice of integers to their string representations
		input := []int{1, 2, 3}
		expected := []string{"1", "2", "3"}
		result := SliceMap(input, func(val int) string {
			return string(rune('0' + val))
		})
		assert.Equal(t, expected, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		// Test with empty slice
		var input []int
		result := SliceMap(input, func(val int) int {
			return val * 2
		})
		assert.Nil(t, result)
	})
}
