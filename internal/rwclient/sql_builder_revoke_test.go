package rwclient

import (
	"testing"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

// TestBuildRevokeStatements tests REVOKE statement generation for all object types
func TestBuildRevokeStatements(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		spec     *v1alpha1.RisingWaveUserSpec
		wantLen  int
		contains []string
	}{
		{
			name:     "empty grants",
			userName: "testuser",
			spec:     &v1alpha1.RisingWaveUserSpec{},
			wantLen:  0,
		},
		{
			name:     "database privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Privileges: []v1alpha1.DatabasePrivilegeType{
								v1alpha1.DatabasePrivilegeConnect,
								v1alpha1.DatabasePrivilegeCreate,
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE CONNECT, CREATE ON DATABASE "mydb" FROM "testuser"`,
			},
		},
		{
			name:     "table privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Tables: []v1alpha1.NestedTablePrivilege{
										{
											Name: "mytable",
											Privileges: []v1alpha1.TablePrivilegeType{
												v1alpha1.TablePrivilegeSelect,
												v1alpha1.TablePrivilegeInsert,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT, INSERT ON TABLE "mydb"."public"."mytable" FROM "testuser"`,
			},
		},
		{
			name:     "all tables in schema wildcard",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Tables: []v1alpha1.NestedTablePrivilege{
										{
											Name: "*",
											Privileges: []v1alpha1.TablePrivilegeType{
												v1alpha1.TablePrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT ON ALL TABLES IN SCHEMA "public" FROM "testuser"`,
			},
		},
		{
			name:     "view privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Views: []v1alpha1.NestedViewPrivilege{
										{
											Name: "myview",
											Privileges: []v1alpha1.ViewPrivilegeType{
												v1alpha1.ViewPrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT ON VIEW "mydb"."public"."myview" FROM "testuser"`,
			},
		},
		{
			name:     "materialized view privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									MaterializedViews: []v1alpha1.NestedMaterializedViewPrivilege{
										{
											Name: "mymv",
											Privileges: []v1alpha1.MaterializedViewPrivilegeType{
												v1alpha1.MaterializedViewPrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT ON MATERIALIZED VIEW "mydb"."public"."mymv" FROM "testuser"`,
			},
		},
		{
			name:     "source privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Sources: []v1alpha1.NestedSourcePrivilege{
										{
											Name: "mysource",
											Privileges: []v1alpha1.SourcePrivilegeType{
												v1alpha1.SourcePrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT ON SOURCE "mydb"."public"."mysource" FROM "testuser"`,
			},
		},
		{
			name:     "sink privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Sinks: []v1alpha1.NestedSinkPrivilege{
										{
											Name: "mysink",
											Privileges: []v1alpha1.SinkPrivilegeType{
												v1alpha1.SinkPrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE SELECT ON SINK "mydb"."public"."mysink" FROM "testuser"`,
			},
		},
		{
			name:     "secret privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Secrets: []v1alpha1.NestedSecretPrivilege{
										{
											Name: "mysecret",
											Privileges: []v1alpha1.SecretPrivilegeType{
												v1alpha1.SecretPrivilegeUsage,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE USAGE ON SECRET "mydb"."public"."mysecret" FROM "testuser"`,
			},
		},
		{
			name:     "function privileges",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Functions: []v1alpha1.NestedFunctionPrivilege{
										{
											Name: "myfunc",
											Privileges: []v1alpha1.FunctionPrivilegeType{
												v1alpha1.FunctionPrivilegeExecute,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 1,
			contains: []string{
				`REVOKE EXECUTE ON FUNCTION "mydb"."public"("myfunc") FROM "testuser"`,
			},
		},
		{
			name:     "multiple object types",
			userName: "testuser",
			spec: &v1alpha1.RisingWaveUserSpec{
				Grants: &v1alpha1.GrantSpec{
					Databases: []v1alpha1.DatabasePrivilege{
						{
							Name: "mydb",
							Privileges: []v1alpha1.DatabasePrivilegeType{
								v1alpha1.DatabasePrivilegeConnect,
							},
							Schemas: []v1alpha1.NestedSchemaPrivilege{
								{
									Name: "public",
									Privileges: []v1alpha1.SchemaPrivilegeType{
										v1alpha1.SchemaPrivilegeUsage,
									},
									Tables: []v1alpha1.NestedTablePrivilege{
										{
											Name: "t1",
											Privileges: []v1alpha1.TablePrivilegeType{
												v1alpha1.TablePrivilegeSelect,
											},
										},
									},
									Views: []v1alpha1.NestedViewPrivilege{
										{
											Name: "v1",
											Privileges: []v1alpha1.ViewPrivilegeType{
												v1alpha1.ViewPrivilegeSelect,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			wantLen: 4,
			contains: []string{
				`REVOKE CONNECT ON DATABASE "mydb" FROM "testuser"`,
				`REVOKE USAGE ON SCHEMA "public" FROM "testuser"`,
				`REVOKE SELECT ON TABLE "mydb"."public"."t1" FROM "testuser"`,
				`REVOKE SELECT ON VIEW "mydb"."public"."v1" FROM "testuser"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRevokeStatements(tt.userName, tt.spec)
			if len(got) != tt.wantLen {
				t.Errorf("BuildRevokeStatements() returned %d statements, want %d", len(got), tt.wantLen)
			}
			for _, want := range tt.contains {
				found := false
				for _, stmt := range got {
					if stmt == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("BuildRevokeStatements() did not contain expected statement %q\nGot: %v", want, got)
				}
			}
		})
	}
}

// TestNormalizePrivileges tests the NormalizePrivileges function
func TestNormalizePrivileges(t *testing.T) {
	tests := []struct {
		name  string
		privs []string
		want  string
	}{
		{
			name:  "empty privileges",
			privs: []string{},
			want:  "USAGE",
		},
		{
			name:  "single privilege",
			privs: []string{"SELECT"},
			want:  "SELECT",
		},
		{
			name:  "multiple privileges",
			privs: []string{"SELECT", "INSERT", "UPDATE"},
			want:  "SELECT, INSERT, UPDATE",
		},
		{
			name:  "ALL PRIVILEGES normalized to ALL",
			privs: []string{"ALL PRIVILEGES"},
			want:  "ALL",
		},
		{
			name:  "mixed with ALL PRIVILEGES",
			privs: []string{"SELECT", "ALL PRIVILEGES", "INSERT"},
			want:  "SELECT, ALL, INSERT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePrivileges(tt.privs)
			if got != tt.want {
				t.Errorf("NormalizePrivileges() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBuildRevokeNestedTablePrivilege tests table REVOKE generation
func TestBuildRevokeNestedTablePrivilege(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		database string
		schema   string
		priv     *v1alpha1.NestedTablePrivilege
		wantStmt string
	}{
		{
			name:     "specific table",
			userName: "testuser",
			database: "mydb",
			schema:   "public",
			priv: &v1alpha1.NestedTablePrivilege{
				Name: "mytable",
				Privileges: []v1alpha1.TablePrivilegeType{
					v1alpha1.TablePrivilegeSelect,
					v1alpha1.TablePrivilegeInsert,
				},
			},
			wantStmt: `REVOKE SELECT, INSERT ON TABLE "mydb"."public"."mytable" FROM "testuser"`,
		},
		{
			name:     "all tables wildcard",
			userName: "testuser",
			database: "mydb",
			schema:   "public",
			priv: &v1alpha1.NestedTablePrivilege{
				Name: "*",
				Privileges: []v1alpha1.TablePrivilegeType{
					v1alpha1.TablePrivilegeSelect,
				},
			},
			wantStmt: `REVOKE SELECT ON ALL TABLES IN SCHEMA "public" FROM "testuser"`,
		},
		{
			name:     "ALL table privileges",
			userName: "testuser",
			database: "mydb",
			schema:   "public",
			priv: &v1alpha1.NestedTablePrivilege{
				Name: "mytable",
				Privileges: []v1alpha1.TablePrivilegeType{
					v1alpha1.TablePrivilegeAll,
				},
			},
			wantStmt: `REVOKE ALL ON TABLE "mydb"."public"."mytable" FROM "testuser"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRevokeNestedTablePrivilege(tt.userName, tt.database, tt.schema, tt.priv)
			if got != tt.wantStmt {
				t.Errorf("buildRevokeNestedTablePrivilege() = %q, want %q", got, tt.wantStmt)
			}
		})
	}
}

// TestBuildRevokeNestedViewPrivilege tests view REVOKE generation
func TestBuildRevokeNestedViewPrivilege(t *testing.T) {
	tests := []struct {
		name     string
		userName string
		database string
		schema   string
		priv     *v1alpha1.NestedViewPrivilege
		wantStmt string
	}{
		{
			name:     "specific view",
			userName: "testuser",
			database: "mydb",
			schema:   "public",
			priv: &v1alpha1.NestedViewPrivilege{
				Name: "myview",
				Privileges: []v1alpha1.ViewPrivilegeType{
					v1alpha1.ViewPrivilegeSelect,
				},
			},
			wantStmt: `REVOKE SELECT ON VIEW "mydb"."public"."myview" FROM "testuser"`,
		},
		{
			name:     "all views wildcard",
			userName: "testuser",
			database: "mydb",
			schema:   "public",
			priv: &v1alpha1.NestedViewPrivilege{
				Name: "*",
				Privileges: []v1alpha1.ViewPrivilegeType{
					v1alpha1.ViewPrivilegeSelect,
				},
			},
			wantStmt: `REVOKE SELECT ON ALL VIEWS IN SCHEMA "public" FROM "testuser"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRevokeNestedViewPrivilege(tt.userName, tt.database, tt.schema, tt.priv)
			if got != tt.wantStmt {
				t.Errorf("buildRevokeNestedViewPrivilege() = %q, want %q", got, tt.wantStmt)
			}
		})
	}
}
