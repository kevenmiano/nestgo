package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileDirRegistersRepositoryFactoryInModule(t *testing.T) {
	dir := t.TempDir()
	source := `package orders
import "context"
@Entity({table: "orders"})
type Order struct { ID int64 }
@Repository(Order, "orders")
type OrdersRepository struct{}
@Query("SELECT id FROM orders WHERE id=:id")
func (r *OrdersRepository) Find(ctx context.Context, id int64) (*Order, error) { return nil, nil }
@Module({providers:[]})
type OrdersModule struct{}
`
	if err := os.WriteFile(filepath.Join(dir, "orders.mgo"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := CompileDir(dir); err != nil {
		t.Fatal(err)
	}
	metadata, err := os.ReadFile(filepath.Join(dir, "zz_modforge_metadata.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"OrdersRepositoryGenerated", "Factory:", "NewOrdersRepositoryGeneratedFromConnections", "database.Connections"} {
		if !strings.Contains(string(metadata), expected) {
			t.Fatalf("missing %q from generated metadata: %s", expected, metadata)
		}
	}
}
