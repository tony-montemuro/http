package assert

import (
	"maps"
	"testing"
	"time"
)

func Equal[T comparable](t *testing.T, actual, expected T) {
	t.Helper()

	if actual != expected {
		t.Errorf("got: %v; want: %v", actual, expected)
	}
}

func SliceEqual[T comparable](t *testing.T, actual, expected []T) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("different sizes. got: (%v, len: %d), want: (%v, len: %d)", actual, len(actual), expected, len(expected))
		return
	}

	for i := range len(actual) {
		Equal(t, actual[i], expected[i])
	}
}

func MatrixEqual[T comparable](t *testing.T, actual, expected [][]T) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Errorf("different overall size. got: (%v, len: %d), want: (%v, len: %d)", actual, len(actual), expected, len(expected))
		return
	}

	for i := range len(actual) {
		actualRow := actual[i]
		expectedRow := expected[i]

		if len(actualRow) != len(expectedRow) {
			t.Errorf("row %d has different sizes. got: (%v, len: %d); want: (%v, len: %d)", i, actualRow, len(actualRow), expectedRow, len(expectedRow))
			return
		}

		for j := range actualRow {
			Equal(t, actualRow[j], expectedRow[j])
		}
	}
}

func MapEqual[S, T comparable](t *testing.T, actual, expected map[S]T) {
	t.Helper()

	if !maps.Equal(actual, expected) {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func DateEqual(t *testing.T, actual, expected time.Time) {
	t.Helper()

	if !actual.Equal(expected) {
		t.Errorf("got: %v, want: %v", actual, expected)
	}
}

func ErrorStatus(t *testing.T, err error, expectError bool) bool {
	t.Helper()

	if err != nil {
		if !expectError {
			t.Errorf("got unexpected error: %s", err.Error())
		}
		return false
	}

	if expectError {
		t.Error("did not get expected error")
		return false
	}

	return true
}
