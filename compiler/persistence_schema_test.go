package compiler

import (
	"strings"
	"testing"
)

func TestValidateSchemaRelationsAndIndexes(t *testing.T) {
	schema := Schema{Entities: []SchemaEntity{
		{
			Name: "User", Table: "users",
			Columns: []SchemaColumn{{Name: "id", PrimaryKey: true}, {Name: "email"}},
			Indexes: []SchemaIndex{{Name: "idx_users_email", Columns: []string{"email"}, Unique: true}},
			Relations: []SchemaRelation{{Name: "Posts", Kind: "one-to-many", Target: "Post", ForeignKey: "user_id"}},
		},
		{
			Name: "Post", Table: "posts",
			Columns: []SchemaColumn{{Name: "id", PrimaryKey: true}, {Name: "user_id"}},
			Relations: []SchemaRelation{{Name: "User", Kind: "many-to-one", Target: "User", ForeignKey: "user_id"}},
		},
	}}
	if err := ValidateSchema(schema); err != nil {
		t.Fatal(err)
	}
	schema.Entities[0].Indexes[0].Columns = []string{"nonexistent"}
	if err := ValidateSchema(schema); err == nil || !strings.Contains(err.Error(), "unknown column") {
		t.Fatalf("expected unknown index column diagnostic: %v", err)
	}
}

func TestValidateSchemaRejectsUnsafeMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		schema Schema
	}{
		{"injection", Schema{Entities: []SchemaEntity{{Name: "User", Table: "users; DROP TABLE users", Columns: []SchemaColumn{{Name: "id"}}}}}},
		{"missing-target", Schema{Entities: []SchemaEntity{{Name: "User", Table: "users", Columns: []SchemaColumn{{Name: "id"}}, Relations: []SchemaRelation{{Name: "Posts", Kind: "one-to-many", Target: "Post", ForeignKey: "user_id"}}}}}},
		{"view-index", Schema{Entities: []SchemaEntity{{Name: "Report", Table: "report_view", View: true, Columns: []SchemaColumn{{Name: "id"}}, Indexes: []SchemaIndex{{Name: "idx_report", Columns: []string{"id"}}}}}}},
		{"missing-through", Schema{Entities: []SchemaEntity{{Name: "User", Table: "users", Columns: []SchemaColumn{{Name: "id"}}, Relations: []SchemaRelation{{Name: "Roles", Kind: "many-to-many", Target: "User"}}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSchema(tc.schema); err == nil {
				t.Fatal("expected schema diagnostic")
			}
		})
	}
}
