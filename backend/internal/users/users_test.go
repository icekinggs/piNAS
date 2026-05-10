package users

import (
	"errors"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	valid := []string{"admin", "maria_01", "john-doe", "joao.silva"}
	for _, username := range valid {
		if err := ValidateUsername(username); err != nil {
			t.Fatalf("expected %q to be valid: %v", username, err)
		}
	}

	invalid := []string{"ab", "1admin", "root/user", "bad space", "bad$char"}
	for _, username := range invalid {
		if err := ValidateUsername(username); !errors.Is(err, ErrInvalidUsername) {
			t.Fatalf("expected %q to be invalid, got %v", username, err)
		}
	}
}
