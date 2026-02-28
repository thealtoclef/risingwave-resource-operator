/*
Copyright 2025 RisingWave Labs.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"regexp"
	"testing"
	"unicode"
)

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		expectMinLen   int
		expectMaxLen   int
		checkCharTypes bool
	}{
		{
			name:           "length 16",
			length:         16,
			expectMinLen:   16,
			expectMaxLen:   16,
			checkCharTypes: true,
		},
		{
			name:           "length 32",
			length:         32,
			expectMinLen:   32,
			expectMaxLen:   32,
			checkCharTypes: true,
		},
		{
			name:           "length below minimum defaults to 16",
			length:         4,
			expectMinLen:   16,
			expectMaxLen:   16,
			checkCharTypes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd := GenerateRandomPassword(tt.length)

			if len(pwd) < tt.expectMinLen || len(pwd) > tt.expectMaxLen {
				t.Errorf("GenerateRandomPassword(%d) length = %d, want between %d and %d", tt.length, len(pwd), tt.expectMinLen, tt.expectMaxLen)
			}

			if tt.checkCharTypes {
				hasLower := regexp.MustCompile(`[a-z]`).MatchString(pwd)
				hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(pwd)
				hasDigit := regexp.MustCompile(`[0-9]`).MatchString(pwd)
				hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{}|;:,.<>?]`).MatchString(pwd)

				if !hasLower {
					t.Errorf("GenerateRandomPassword(%d) has no lowercase letters", tt.length)
				}
				if !hasUpper {
					t.Errorf("GenerateRandomPassword(%d) has no uppercase letters", tt.length)
				}
				if !hasDigit {
					t.Errorf("GenerateRandomPassword(%d) has no digits", tt.length)
				}
				if !hasSpecial {
					t.Errorf("GenerateRandomPassword(%d) has no special characters", tt.length)
				}
			}
		})
	}
}

func TestGenerateRandomPasswordUniqueness(t *testing.T) {
	// Generate 10 passwords and verify they are all different
	passwords := make(map[string]bool)
	for i := 0; i < 10; i++ {
		pwd := GenerateRandomPassword(16)
		if passwords[pwd] {
			t.Errorf("GenerateRandomPassword generated duplicate password on iteration %d", i)
		}
		passwords[pwd] = true
	}

	if len(passwords) != 10 {
		t.Errorf("GenerateRandomPassword uniqueness: expected 10 unique passwords, got %d", len(passwords))
	}
}

func TestGenerateRandomPasswordCharacterTypes(t *testing.T) {
	// Run multiple times to ensure consistent character type coverage
	for attempt := 0; attempt < 5; attempt++ {
		pwd := GenerateRandomPassword(20)

		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSpecial := false

		for _, ch := range pwd {
			if unicode.IsLower(ch) {
				hasLower = true
			}
			if unicode.IsUpper(ch) {
				hasUpper = true
			}
			if unicode.IsDigit(ch) {
				hasDigit = true
			}
			if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
				hasSpecial = true
			}
		}

		if !hasLower {
			t.Errorf("Attempt %d: password has no lowercase letters: %s", attempt, pwd)
		}
		if !hasUpper {
			t.Errorf("Attempt %d: password has no uppercase letters: %s", attempt, pwd)
		}
		if !hasDigit {
			t.Errorf("Attempt %d: password has no digits: %s", attempt, pwd)
		}
		if !hasSpecial {
			t.Errorf("Attempt %d: password has no special characters: %s", attempt, pwd)
		}
	}
}
