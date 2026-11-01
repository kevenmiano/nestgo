package compiler

import (
	"fmt"
	"go/token"
	"strings"
)

// pipelineInterfaceAssertions emits static Go interface satisfaction checks
// for decorated providers. It deduplicates assertions across controllers and
// methods, so registration is unaffected and runtime reflection is unchanged.
func pipelineInterfaceAssertions(files []*File) []string {
	seen := make(map[string]bool)
	var assertions []string
	for _, file := range files {
		for _, declaration := range file.Declarations {
			for _, annotation := range declaration.Annotations {
				var contract string
				switch annotation.Name {
				case "UseGuards":
					contract = "Guard"
				case "UsePipes":
					contract = "Pipe"
				case "UseFilters":
					contract = "ExceptionFilter"
				default:
					continue
				}
				arg := strings.TrimSpace(annotation.Argument)
				if strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") {
					arg = strings.TrimSpace(arg[1 : len(arg)-1])
				}
				for _, raw := range strings.Split(arg, ",") {
					name := strings.TrimSpace(raw)
					// pipelineMetadata reports invalid provider references.
					if !token.IsIdentifier(name) {
						continue
					}
					check := fmt.Sprintf("var _ httpadapter.%s = (*%s)(nil)", contract, name)
					if !seen[check] {
						seen[check] = true
						assertions = append(assertions, check)
					}
				}
			}
		}
	}
	sortStrings(assertions)
	return assertions
}
