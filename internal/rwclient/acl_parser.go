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

package rwclient

import (
	"strings"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

// PrivilegeGrant represents a single privilege grant for a user.
type PrivilegeGrant struct {
	Privilege       string
	WithGrantOption bool
}

// UserPrivileges represents the set of privileges a user has on an object.
type UserPrivileges struct {
	User       string
	Privileges []PrivilegeGrant
}

// ParseACL parses a RisingWave ACL string (e.g., "{user1=arwd/root,user2=r/root}").
func ParseACL(aclStr string) []UserPrivileges {
	aclStr = strings.Trim(aclStr, "{}")
	if aclStr == "" {
		return nil
	}

	var results []UserPrivileges
	entries := strings.Split(aclStr, ",")
	for _, entry := range entries {
		// entry format: grantee=privs/grantor
		parts := strings.Split(entry, "=")
		if len(parts) != 2 {
			continue
		}

		grantee := parts[0]
		remaining := parts[1]

		// split privs and grantor
		privsGrantor := strings.Split(remaining, "/")
		if len(privsGrantor) != 2 {
			continue
		}

		privsStr := privsGrantor[0]

		up := UserPrivileges{
			User: grantee,
		}

		for i := 0; i < len(privsStr); i++ {
			char := string(privsStr[i])
			withGrantOption := false
			if i+1 < len(privsStr) && privsStr[i+1] == '*' {
				withGrantOption = true
				i++
			}

			up.Privileges = append(up.Privileges, PrivilegeGrant{
				Privilege:       char,
				WithGrantOption: withGrantOption,
			})
		}
		results = append(results, up)
	}

	return results
}

// MapCharToTablePrivilege maps ACL characters to v1alpha1.TablePrivilegeType.
func MapCharToTablePrivilege(char string) v1alpha1.TablePrivilegeType {
	switch char {
	case "r":
		return v1alpha1.TablePrivilegeSelect
	case "a":
		return v1alpha1.TablePrivilegeInsert
	case "w":
		return v1alpha1.TablePrivilegeUpdate
	case "d":
		return v1alpha1.TablePrivilegeDelete
	case "D":
		return v1alpha1.TablePrivilegeTruncate
	case "x":
		return v1alpha1.TablePrivilegeReferences
	case "t":
		return v1alpha1.TablePrivilegeTrigger
	default:
		return ""
	}
}

// MapCharToDatabasePrivilege maps ACL characters to v1alpha1.DatabasePrivilegeType.
func MapCharToDatabasePrivilege(char string) v1alpha1.DatabasePrivilegeType {
	switch char {
	case "c":
		return v1alpha1.DatabasePrivilegeConnect
	case "C":
		return v1alpha1.DatabasePrivilegeCreate
	default:
		return ""
	}
}

// MapCharToSchemaPrivilege maps ACL characters to v1alpha1.SchemaPrivilegeType.
func MapCharToSchemaPrivilege(char string) v1alpha1.SchemaPrivilegeType {
	switch char {
	case "U":
		return v1alpha1.SchemaPrivilegeUsage
	case "C":
		return v1alpha1.SchemaPrivilegeCreate
	default:
		return ""
	}
}

// MapCharToSourcePrivilege maps ACL characters to v1alpha1.SourcePrivilegeType.
func MapCharToSourcePrivilege(char string) v1alpha1.SourcePrivilegeType {
	switch char {
	case "r":
		return v1alpha1.SourcePrivilegeSelect
	default:
		return ""
	}
}

// MapCharToSinkPrivilege maps ACL characters to v1alpha1.SinkPrivilegeType.
func MapCharToSinkPrivilege(char string) v1alpha1.SinkPrivilegeType {
	switch char {
	case "r":
		return v1alpha1.SinkPrivilegeSelect
	default:
		return ""
	}
}

// MapCharToSecretPrivilege maps ACL characters to v1alpha1.SecretPrivilegeType.
func MapCharToSecretPrivilege(char string) v1alpha1.SecretPrivilegeType {
	switch char {
	case "U":
		return v1alpha1.SecretPrivilegeUsage
	default:
		return ""
	}
}

// MapCharToFunctionPrivilege maps ACL characters to v1alpha1.FunctionPrivilegeType.
func MapCharToFunctionPrivilege(char string) v1alpha1.FunctionPrivilegeType {
	switch char {
	case "X":
		return v1alpha1.FunctionPrivilegeExecute
	default:
		return ""
	}
}

// MapCharToTablePrivilegeString maps ACL characters to table privilege strings.
func MapCharToTablePrivilegeString(char string) string {
	return string(MapCharToTablePrivilege(char))
}

// MapCharToSourcePrivilegeString maps ACL characters to source privilege strings.
func MapCharToSourcePrivilegeString(char string) string {
	return string(MapCharToSourcePrivilege(char))
}

// MapCharToSinkPrivilegeString maps ACL characters to sink privilege strings.
func MapCharToSinkPrivilegeString(char string) string {
	return string(MapCharToSinkPrivilege(char))
}

// MapCharToSecretPrivilegeString maps ACL characters to secret privilege strings.
func MapCharToSecretPrivilegeString(char string) string {
	return string(MapCharToSecretPrivilege(char))
}

// MapCharToFunctionPrivilegeString maps ACL characters to function privilege strings.
func MapCharToFunctionPrivilegeString(char string) string {
	return string(MapCharToFunctionPrivilege(char))
}
