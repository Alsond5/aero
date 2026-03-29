package aero

import "strings"

type nodeKind uint8

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindWildcard
)

type TrieNode struct {
	kind      nodeKind
	label     string
	paramName string

	staticChildren map[string]*TrieNode
	paramChild     *TrieNode
	wildcardChild  *TrieNode

	route *Route
}

func newTrieNode(kind nodeKind, label string) *TrieNode {
	n := &TrieNode{
		kind:  kind,
		label: label,
	}
	if kind == nodeKindStatic {
		n.staticChildren = make(map[string]*TrieNode, 2)
	}

	return n
}

type SegmentTrie struct {
	root *TrieNode
}

func newARTTree() *SegmentTrie {
	return &SegmentTrie{
		root: newTrieNode(nodeKindStatic, ""),
	}
}

func (t *SegmentTrie) Insert(path string, route *Route) {
	node := t.root

	if path == "/" {
		node.route = route
		return
	}

	start := 0
	if len(path) > 0 && path[0] == '/' {
		start = 1
	}

	for start < len(path) {
		end := strings.IndexByte(path[start:], '/')
		var seg string
		if end == -1 {
			seg = path[start:]
			start = len(path)
		} else {
			seg = path[start : start+end]
			start = start + end + 1
		}

		if seg == "" {
			continue
		}

		switch {
		case seg == "*":
			if node.wildcardChild == nil {
				node.wildcardChild = newTrieNode(nodeKindWildcard, "*")
			}
			node = node.wildcardChild

		case seg[0] == ':':
			if node.paramChild == nil {
				node.paramChild = newTrieNode(nodeKindParam, "")
				node.paramChild.paramName = seg[1:]
			}
			node = node.paramChild

		default:
			child, ok := node.staticChildren[seg]
			if !ok {
				child = newTrieNode(nodeKindStatic, seg)
				node.staticChildren[seg] = child
			}
			node = child
		}
	}

	node.route = route
}

func (t *SegmentTrie) Search(path string, params *ParamValues, paramsCount *int) *Route {
	node := t.root

	if path == "/" {
		return node.route
	}

	start := 0
	if len(path) > 0 && path[0] == '/' {
		start = 1
	}

	for start <= len(path) {
		end := strings.IndexByte(path[start:], '/')

		var seg string
		if end == -1 {
			seg = path[start:]
			start = len(path) + 1
		} else {
			seg = path[start : start+end]
			start = start + end + 1
		}

		if seg == "" {
			continue
		}

		if child, ok := node.staticChildren[seg]; ok {
			node = child
			continue
		}

		if node.paramChild != nil {
			*params = append(*params, Param{
				Key:   node.paramChild.paramName,
				Value: seg,
			})

			*paramsCount++

			node = node.paramChild
			continue
		}

		if node.wildcardChild != nil {
			node = node.wildcardChild
			break
		}

		return nil
	}

	return node.route
}
