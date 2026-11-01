package cli

import (
	"fmt"
	"strings"

	"github.com/kevenmiano/nestgo/database/sqlgen"
)

// parseConnectionDialects is shared by compile and read-only check to ensure
// that the CLI never validates output with a different SQL dialect plan.
func parseConnectionDialects(values []string) (map[string]sqlgen.Dialect, error) {
	if len(values) == 0 {
		return nil, nil
	}
	dialects := make(map[string]sqlgen.Dialect, len(values))
	for _, value := range values {
		name, dialectName, ok := strings.Cut(value, ":")
		if !ok || name == "" || dialectName == "" {
			return nil, fmt.Errorf("invalid --connection %q (expected name:postgres or name:mysql)", value)
		}
		if _, exists := dialects[name]; exists {
			return nil, fmt.Errorf("duplicate --connection %q", name)
		}
		switch dialectName {
		case "postgres":
			dialects[name] = sqlgen.PostgreSQL
		case "mysql":
			dialects[name] = sqlgen.Question
		default:
			return nil, fmt.Errorf("unsupported dialect %q for connection %q", dialectName, name)
		}
	}
	return dialects, nil
}
