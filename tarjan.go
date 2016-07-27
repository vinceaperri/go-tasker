package tasker

import (
	"errors"
)

type string_stack struct {
	stack []string
	count int
}

func (ss *string_stack) push(e string) {
	ss.stack = append(ss.stack, e)
	ss.count++
}

func (ss *string_stack) pop() (string, error) {
	if ss.count == 0 {
		return "", errors.New("Stack is empty")
	}
	ss.count--
	e := ss.stack[ss.count]
	ss.stack = ss.stack[:ss.count]
	return e, nil
}

func new_string_stack() *string_stack {
	return &string_stack{make([]string, 0), 0}
}

type tarjan_info struct {
	graph map[string][]string

	index      int
	stack      *string_stack
	index_of   map[string]int
	lowlink_of map[string]int
	on_stack   map[string]bool

	sccs [][]string
}

func tarjan_strongconnect(v string, ti *tarjan_info) {
	ti.index_of[v] = ti.index
	ti.lowlink_of[v] = ti.index
	ti.index++
	ti.stack.push(v)
	ti.on_stack[v] = true

	for _, w := range ti.graph[v] {
		if _, ok := ti.index_of[w]; !ok {
			tarjan_strongconnect(w, ti)
			if ti.lowlink_of[w] < ti.lowlink_of[v] {
				ti.lowlink_of[v] = ti.lowlink_of[w]
			}
		} else if on_stack, ok := ti.on_stack[w]; ok && on_stack {
			if ti.index_of[w] < ti.lowlink_of[v] {
				ti.lowlink_of[v] = ti.index_of[w]
			}
		}
	}

	if ti.lowlink_of[v] == ti.index_of[v] {
		scc := make([]string, 0)
		for {
			w, err := ti.stack.pop()
			if err != nil {
				panic(err)
			}
			ti.on_stack[w] = false
			scc = append(scc, w)
			if v == w {
				break
			}
		}
		ti.sccs = append(ti.sccs, scc)
	}
}

func tarjan_algo(graph map[string][]string) [][]string {
	info := &tarjan_info{
		graph,
		0,
		&string_stack{make([]string, 0), 0},
		make(map[string]int),
		make(map[string]int),
		make(map[string]bool),
		make([][]string, 0),
	}
	for v := range graph {
		if _, ok := info.index_of[v]; !ok {
			tarjan_strongconnect(v, info)
		}
	}
	return info.sccs
}
