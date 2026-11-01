package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultipleModulesRequireRepositoryOwner(t *testing.T) {
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
@Module({providers:[]})
type BillingModule struct{}
`
	if err := os.WriteFile(filepath.Join(dir, "orders.mgo"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	err := CompileDir(dir)
	if err == nil || !strings.Contains(err.Error(), "expected exactly one owning @Module") {
		t.Fatalf("expected explicit repository ownership diagnostic, got %v", err)
	}
}
