package fasthttp

import (
	"strings"

	"github.com/kevenmiano/nestgo/httpadapter"
)

type routeEntry struct {
	index int
	route httpadapter.Route
	parts []string
}

type routeBucket struct {
	literal  map[string][]routeEntry
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

// lookup prefers more specific literal routes over wildcards, independent of
// registration order, while preserving 405 for paths bound to other methods.
func (idx routeIndex) lookup(method, path string) (httpadapter.Handler, map[string]string, bool) {
	parts := splitPath(path)
	bucket := idx[len(parts)]
	if bucket == nil {
		return nil, nil, false
	}
	// Merge the two pre-ordered candidate lists without allocating or sorting.
	literals := bucket.literal[parts[0]]
	wildcards := bucket.wildcard
	li, wi := 0, 0
	foundPath := false
	var selected routeEntry
	bestSpecificity := -1
	for li < len(literals) || wi < len(wildcards) {
		var entry routeEntry
		if wi == len(wildcards) || (li < len(literals) && literals[li].index < wildcards[wi].index) {
			entry = literals[li]
			li++
		} else {
			entry = wildcards[wi]
			wi++
		}
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
		specificity := 0
		for _, part := range entry.parts {
			if !isParam(part) {
				specificity++
			}
		}
		if specificity > bestSpecificity {
			selected = entry
			bestSpecificity = specificity
		}
	}
	if bestSpecificity < 0 {
		return nil, nil, foundPath
	}
	var params map[string]string
	for i, part := range selected.parts {
		if isParam(part) {
			if params == nil {
				params = make(map[string]string)
			}
			params[part[1:len(part)-1]] = parts[i]
		}
	}
	return selected.route.Handler, params, true
}

// routeOverlap reports whether two patterns can match the same path and whether
// either is strictly more specific. Overlapping incomparable/equivalent patterns
// are rejected at registration instead of depending on declaration order.
func routeOverlap(a, b string) (overlaps, aMore, bMore bool) {
	left, right := splitPath(a), splitPath(b)
	if len(left) != len(right) {
		return false, false, false
	}
	for i := range left {
		aParam, bParam := isParam(left[i]), isParam(right[i])
		if !aParam && !bParam && left[i] != right[i] {
			return false, false, false
		}
		if !aParam && bParam {
			aMore = true
		}
		if aParam && !bParam {
			bMore = true
		}
	}
	return true, aMore, bMore
}
