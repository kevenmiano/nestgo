package fasthttp

import (
	"sort"
	"strings"

	"github.com/kevenmiano/nestgo/httpadapter"
)

type routeEntry struct {
	index int
	route httpadapter.Route
	parts []string
}

type routeBucket struct {
	literal map[string][]routeEntry
	wildcard []routeEntry
}

type routeIndex map[int]*routeBucket

func splitPath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func compileRoutes(routes []httpadapter.Route) routeIndex {
	index := make(routeIndex)
	for i, route := range routes {
		parts := splitPath(route.Path)
		bucket := index[len(parts)]
		if bucket == nil {
			bucket = &routeBucket{literal: make(map[string][]routeEntry)}
			index[len(parts)] = bucket
		}
		entry := routeEntry{index: i, route: route, parts: parts}
		if isParam(parts[0]) {
			bucket.wildcard = append(bucket.wildcard, entry)
		} else {
			bucket.literal[parts[0]] = append(bucket.literal[parts[0]], entry)
		}
	}
	return index
}

func isParam(s string) bool {
	return len(s) >= 3 && s[0] == '{' && s[len(s)-1] == '}'
}

// lookup preserves registration order for overlapping patterns and preserves
// 405 semantics when a path exists for a different HTTP method.
func (idx routeIndex) lookup(method, path string) (httpadapter.Handler, map[string]string, bool) {
	parts := splitPath(path)
	bucket := idx[len(parts)]
	if bucket == nil {
		return nil, nil, false
	}
	candidates := append([]routeEntry(nil), bucket.literal[parts[0]]...)
	candidates = append(candidates, bucket.wildcard...)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].index < candidates[j].index })
	foundPath := false
	for _, entry := range candidates {
		matched := true
		for i, part := range entry.parts {
			if !isParam(part) && part != parts[i] {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		foundPath = true
		if entry.route.Method != method {
			continue
		}
		var params map[string]string
		for i, part := range entry.parts {
			if isParam(part) {
				if params == nil {
					params = make(map[string]string)
				}
				params[part[1:len(part)-1]] = parts[i]
			}
		}
		return entry.route.Handler, params, true
	}
	return nil, nil, foundPath
}
