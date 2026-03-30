package aero

import "strings"

type nodeKind uint8

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindWildcard
)

type children map[string]*trieNode

type trieNode struct {
	endpoint       *endpoint
	staticChildren children
	kind           nodeKind
	label          string
	paramChild     *trieNode
	wildcardChild  *trieNode
}

func newTrieNode(kind nodeKind, label string) *trieNode {
	n := &trieNode{
		kind:  kind,
		label: label,
	}
	if kind == nodeKindStatic {
		n.staticChildren = make(map[string]*trieNode, 2)
	}

	return n
}

type segmentTrie struct {
	root *trieNode
}

func newSegmentTrie() *segmentTrie {
	return &segmentTrie{
		root: newTrieNode(nodeKindStatic, ""),
	}
}

func (t *segmentTrie) Insert(path string, mi int, route *route) {
	node := t.root

	if path == "/" {
		if node.endpoint == nil {
			node.endpoint = newEndpoint()
		}
		node.endpoint.setRoute(mi, route)
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
				node.paramChild = newTrieNode(nodeKindParam, seg[1:])
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

	if node.endpoint == nil {
		node.endpoint = newEndpoint()
	}
	node.endpoint.setRoute(mi, route)
}

func (t *segmentTrie) Search(path string, params *ParamValues, paramsCount *int) *endpoint {
	node := t.root

	if path == "/" {
		return node.endpoint
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
			params[*paramsCount] = Param{
				Key:   node.paramChild.label,
				Value: seg,
			}
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

	return node.endpoint
}
