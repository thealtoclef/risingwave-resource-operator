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
	"crypto/rand"
	"math/big"
)

const (
	// DefaultPasswordLength is the default length for generated passwords.
	DefaultPasswordLength = 16

	// Character sets for password generation.
	lowercaseLetters = "abcdefghijklmnopqrstuvwxyz"
	uppercaseLetters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits           = "0123456789"
	specialChars     = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	allChars         = lowercaseLetters + uppercaseLetters + digits + specialChars
)

// GenerateRandomPassword generates a random alphanumeric password with special characters.
// The password will contain at least one lowercase letter, one uppercase letter,
// one digit, and one special character.
func GenerateRandomPassword(length int) string {
	if length < 8 {
		length = DefaultPasswordLength
	}

	// Ensure password contains at least one of each required character type.
	password := make([]byte, 0, length)

	// Add one lowercase letter
	if idx, err := randInt(len(lowercaseLetters)); err == nil {
		password = append(password, lowercaseLetters[idx])
	}

	// Add one uppercase letter
	if idx, err := randInt(len(uppercaseLetters)); err == nil {
		password = append(password, uppercaseLetters[idx])
	}

	// Add one digit
	if idx, err := randInt(len(digits)); err == nil {
		password = append(password, digits[idx])
	}

	// Add one special character
	if idx, err := randInt(len(specialChars)); err == nil {
		password = append(password, specialChars[idx])
	}

	// Fill the rest with random characters from all sets
	for i := len(password); i < length; i++ {
		if idx, err := randInt(len(allChars)); err == nil {
			password = append(password, allChars[idx])
		}
	}

	// Shuffle the password characters
	shuffleBytes(password)

	return string(password)
}

// randInt returns a random integer in the range [0, max).
func randInt(max int) (int, error) {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(nBig.Int64()), nil
}

// shuffleBytes shuffles a byte slice in place using Fisher-Yates algorithm.
func shuffleBytes(data []byte) {
	for i := len(data) - 1; i > 0; i-- {
		j, err := randInt(i + 1)
		if err != nil {
			continue
		}
		data[i], data[j] = data[j], data[i]
	}
}
