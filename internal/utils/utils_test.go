package utils

import (
	"testing"
)

func TestNamespacedName(t *testing.T) {
	result := NamespacedName("foo", "bar")
	expected := "foo@bar"

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}
