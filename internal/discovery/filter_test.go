package discovery

import (
	"testing"
)

func TestFilter_FilterByName(t *testing.T) {
	filter := NewFilter()

	tests := []struct {
		name     string
		tests    []string
		pattern  string
		expected int // Expected number of matches
	}{
		{
			name:     "empty pattern returns all",
			tests:    []string{"UserTest.php", "PaymentTest.php", "OrderTest.php"},
			pattern:  "",
			expected: 3,
		},
		{
			name:     "wildcard pattern matches suffix",
			tests:    []string{"UserTest.php", "PaymentTest.php", "OrderTest.php"},
			pattern:  "*UserTest.php",
			expected: 1,
		},
		{
			name:     "wildcard pattern matches substring",
			tests:    []string{"UserTest.php", "PaymentTest.php", "OrderTest.php", "PaymentServiceTest.php"},
			pattern:  "*Payment*",
			expected: 2,
		},
		{
			name:     "simple contains match",
			tests:    []string{"UserTest.php", "PaymentTest.php", "OrderTest.php"},
			pattern:  "Payment",
			expected: 1,
		},
		{
			name:     "no matches",
			tests:    []string{"UserTest.php", "PaymentTest.php"},
			pattern:  "*NonExistent*",
			expected: 0,
		},
		{
			name:     "full path with wildcard",
			tests:    []string{"/path/to/UserTest.php", "/path/to/PaymentTest.php"},
			pattern:  "*UserTest.php",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.FilterByName(tt.tests, tt.pattern)
			if len(result) != tt.expected {
				t.Errorf("expected %d matches, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestFilter_FilterByName_EdgeCases(t *testing.T) {
	filter := NewFilter()

	t.Run("empty test list", func(t *testing.T) {
		result := filter.FilterByName([]string{}, "*Test.php")
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("pattern with multiple wildcards", func(t *testing.T) {
		tests := []string{"UserServiceTest.php", "UserControllerTest.php", "PaymentTest.php"}
		result := filter.FilterByName(tests, "*User*Test.php")
		if len(result) < 2 {
			t.Errorf("expected at least 2 matches, got %d", len(result))
		}
	})
}
