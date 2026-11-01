package compiler

import (
	"fmt"
	"strings"
)

// Schema describes database declarations after decorator parsing. No database
// connection or reflection is needed to validate this graph.
type Schema struct {
	Entities []SchemaEntity
}

type SchemaEntity struct {
	Name      string
	Table     string
	View      bool
	Columns   []SchemaColumn
	Indexes   []SchemaIndex
	Relations []SchemaRelation
}

type SchemaColumn struct {
	Name       string
	PrimaryKey bool
}

type SchemaIndex struct {
	Name    string
	Columns []string
	Unique  bool
}

type SchemaRelation struct {
	Name       string
	Target     string
	ForeignKey string
	Kind       string // many-to-one, one-to-one, one-to-many, many-to-many
	Through    string // required for many-to-many
}

// ValidateSchema fails closed on invalid local relationships and index metadata.
// The database still owns referential integrity and migration execution.
func ValidateSchema(schema Schema) error {
	entities := make(map[string]SchemaEntity, len(schema.Entities))
	for _, entity := range schema.Entities {
		if !schemaIdentifier(entity.Name) || !schemaIdentifier(entity.Table) {
			return fmt.Errorf("schema: invalid entity/table %q/%q", entity.Name, entity.Table)
		}
		if _, exists := entities[entity.Name]; exists {
			return fmt.Errorf("schema: duplicate entity %q", entity.Name)
		}
		entities[entity.Name] = entity
	}
	tables := map[string]string{}
	for _, entity := range schema.Entities {
		if previous, exists := tables[entity.Table]; exists {
			return fmt.Errorf("schema: table %q used by both %s and %s", entity.Table, previous, entity.Name)
		}
		tables[entity.Table] = entity.Name
		columns := make(map[string]bool, len(entity.Columns))
		for _, column := range entity.Columns {
			if !schemaIdentifier(column.Name) || columns[column.Name] {
				return fmt.Errorf("schema: invalid or duplicate column %q in %s", column.Name, entity.Name)
			}
			columns[column.Name] = true
		}
		indexNames := map[string]bool{}
		for _, index := range entity.Indexes {
			if entity.View {
				return fmt.Errorf("schema: view %s cannot declare managed indexes", entity.Name)
			}
			if !schemaIdentifier(index.Name) || indexNames[index.Name] || len(index.Columns) == 0 {
				return fmt.Errorf("schema: invalid index %q in %s", index.Name, entity.Name)
			}
			indexNames[index.Name] = true
			for _, name := range index.Columns {
				if !columns[name] {
					return fmt.Errorf("schema: index %s references unknown column %s", index.Name, name)
				}
			}
		}
	}
	for _, entity := range schema.Entities {
		columns := map[string]bool{}
		for _, c := range entity.Columns {
			columns[c.Name] = true
		}
		for _, rel := range entity.Relations {
			target, ok := entities[rel.Target]
			if !ok {
				return fmt.Errorf("schema: %s.%s targets unknown entity %s", entity.Name, rel.Name, rel.Target)
			}
			if target.View || entity.View {
				return fmt.Errorf("schema: managed relation %s.%s cannot involve a view", entity.Name, rel.Name)
			}
			switch rel.Kind {
			case "many-to-one", "one-to-one":
				if !columns[rel.ForeignKey] {
					return fmt.Errorf("schema: %s.%s references unknown foreign key %s", entity.Name, rel.Name, rel.ForeignKey)
				}
			case "one-to-many":
				found := false
				for _, col := range target.Columns {
					found = found || col.Name == rel.ForeignKey
				}
				if !found {
					return fmt.Errorf("schema: %s.%s references missing target foreign key %s", entity.Name, rel.Name, rel.ForeignKey)
				}
			case "many-to-many":
				if !schemaIdentifier(rel.Through) {
					return fmt.Errorf("schema: %s.%s requires a valid through table", entity.Name, rel.Name)
				}
			default:
				return fmt.Errorf("schema: unsupported relation %q", rel.Kind)
			}
		}
	}
	return nil
}

func schemaIdentifier(value string) bool {
	if value == "" || strings.Contains(value, ".") || !identStart(value[0]) {
		return false
	}
	for i := 1; i < len(value); i++ {
		if !identPart(value[i]) {
			return false
		}
	}
	return true
}
