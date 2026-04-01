package helpers

import (
	"slices"
)

func CleanUpSlice[T ~[]E, E comparable](slice T) T {
	if len(slice) == 0 {
		return slice
	}

	slice = slices.Compact(slice)
	var fn func(E) bool

	switch any(slice[0]).(type) {
	case string:
		fn = func(v E) bool { return any(v).(string) == "" }
	case int:
		fn = func(v E) bool { return any(v).(int) == 0 }
	case bool:
		fn = func(v E) bool { return !any(v).(bool) }
	case int8:
		fn = func(v E) bool { return any(v).(int8) == 0 }
	case int16:
		fn = func(v E) bool { return any(v).(int16) == 0 }
	case int32:
		fn = func(v E) bool { return any(v).(int32) == 0 }
	case int64:
		fn = func(v E) bool { return any(v).(int64) == 0 }
	}

	if fn != nil {
		return slices.DeleteFunc(slice, fn)
	}
	return slice
}

func SliceReduce[T, M any](s []T, f func(M, T) M, initValue ...M) M {
	var acc M

	if len(initValue) > 0 {
		acc = initValue[0]
	}

	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

func SliceMap[T, M any](s []T, f func(T) M) []M {
	if len(s) == 0 {
		return nil
	}

	result := make([]M, 0, len(s))
	for _, v := range s {
		result = append(result, f(v))
	}
	return result
}

func SliceDifference[T comparable](a, b []T) []T {
	setB := make(map[T]struct{}, len(b))
	for _, v := range b {
		setB[v] = struct{}{}
	}

	var result []T
	for _, v := range a {
		if _, found := setB[v]; !found {
			result = append(result, v)
		}
	}
	return result
}

// SliceIntersection returns elements that appear in both slices
func SliceIntersection[T comparable](a, b []T) []T {
	setB := make(map[T]struct{}, len(b))
	for _, v := range b {
		setB[v] = struct{}{}
	}

	var result []T
	for _, v := range a {
		if _, found := setB[v]; found {
			result = append(result, v)
		}
	}
	return result
}
