// Copyright ©2015 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package topo

import (
	"sort"

	"github.com/openshift/github.com/gonum/graph"
	"github.com/openshift/github.com/gonum/graph/internal"
)

// johnson implements Johnson's "Finding all the elementary
// circuits of a directed graph" algorithm. SIAM J. Comput. 4(1):1975.
//
// Comments in the johnson methods are kept in sync with the comments
// and labels from the paper.
type johnson struct {
	adjacent johnsonGraph      // SCC adjacency list.
	b        []internal.IntSet // Johnson's "B-list".
	blocked  []bool
	s        int

	stack []graph.Node

	result [][]graph.Node
}

// CyclesIn returns the set of elementary cycles in the graph g.
func CyclesIn(g graph.Directed) [][]graph.Node {
	jg := johnsonGraphFrom(g)
	j := johnson{
		adjacent: jg,
		b:        make([]internal.IntSet, len(jg.orig)),
		blocked:  make([]bool, len(jg.orig)),
	}

	// len(j.nodes) is the order of g.
	for j.s < len(j.adjacent.orig)-1 {
		// We use the previous SCC adjacency to reduce the work needed.
		sccs := TarjanSCC(j.adjacent.subgraph(j.s))
		// A_k = adjacency structure of strong component K with least
		//       vertex in subgraph of G induced by {s, s+1, ... ,n}.
		j.adjacent = j.adjacent.sccSubGraph(sccs, 2) // Only allow SCCs with >= 2 vertices.
		if j.adjacent.order() == 0 {
			break
		}

		// s = least vertex in V_k
		if s := j.adjacent.leastVertexIndex(); s < j.s {
			j.s = s
		}
		for i, v := range j.adjacent.orig {
			if !j.adjacent.nodes.Has(v.ID()) {
				continue
			}
			if len(j.adjacent.succ[v.ID()]) > 0 {
				j.blocked[i] = false
				j.b[i] = make(internal.IntSet)
			}
		}
		//L3:
		_ = j.circuit(j.s)
		j.s++
	}

	return j.result
}

// circuit is the CIRCUIT sub-procedure in the paper.
func (j *johnson) circuit(v int) bool {
	f := false
	n := j.adjacent.orig[v]
	j.stack = append(j.stack, n)
	j.blocked[v] = true

	//L1:
	for w := range j.adjacent.succ[n.ID()] {
		w = j.adjacent.indexOf(w)
		if w == j.s {
			// Output circuit composed of stack followed by s.
			r := make([]graph.Node, len(j.stack)+1)
			copy(r, j.stack)
			r[len(r)-1] = j.adjacent.orig[j.s]
			j.result = append(j.result, r)
			f = true
		} else if !j.blocked[w] {
			if j.circuit(w) {
				f = true
			}
		}
	}

	//L2:
	if f {
		j.unblock(v)
	} else {
		for w := range j.adjacent.succ[n.ID()] {
			j.b[j.adjacent.indexOf(w)].Add(v)
		}
	}
	j.stack = j.stack[:len(j.stack)-1]

	return f
}

// unblock is the UNBLOCK sub-procedure in the paper.
func (j *johnson) unblock(u int) {
	j.blocked[u] = false
	for w := range j.b[u] {
		j.b[u].Remove(w)
		if j.blocked[w] {
			j.unblock(w)
		}
	}
}

// johnsonGraph is an edge list representation of a graph with helpers
// necessary for Johnson's algorithm
type johnsonGraph struct {
	// Keep the original graph nodes and a
	// look-up to into the non-sparse
	// collection of potentially sparse IDs.
	orig  []graph.Node
	index map[int]int

	nodes internal.IntSet
	succ  map[int]internal.IntSet
}

// johnsonGraphFrom returns a deep copy of the graph g.
func johnsonGraphFrom(g graph.Directed) johnsonGraph {
	nodes := g.Nodes()
	sort.Sort(byID(nodes))
	c := johnsonGraph{
		orig:  nodes,
		index: make(map[int]int, len(nodes)),

		nodes: make(internal.IntSet, len(nodes)),
		succ:  make(map[int]internal.IntSet),
	}
	for i, u := range nodes {
		c.index[u.ID()] = i
		for _, v := range g.From(u) {
			if c.succ[u.ID()] == nil {
				c.succ[u.ID()] = make(internal.IntSet)
				c.nodes.Add(u.ID())
			}
			c.nodes.Add(v.ID())
			c.succ[u.ID()].Add(v.ID())
		}
	}
	return c
}

type byID []graph.Node

func (n byID) Len() int           { return len(n) }
func (n byID) Less(i, j int) bool { return n[i].ID() < n[j].ID() }
func (n byID) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }

// order returns the order of the graph.
func (g johnsonGraph) order() int { return g.nodes.Count() }

// indexOf returns the index of the retained node for the given node ID.
func (g johnsonGraph) indexOf(id int) int {
	return g.index[id]
}

// leastVertexIndex returns the index into orig of the least vertex.
func (g johnsonGraph) leastVertexIndex() int {
	for _, v := range g.orig {
		if g.nodes.Has(v.ID()) {
			return g.indexOf(v.ID())
		}
	}
	panic("johnsonCycles: empty set")
}

// subgraph returns a subgraph of g induced by {s, s+1, ... , n}. The
// subgraph is destructively generated in g.
func (g johnsonGraph) subgraph(s int) johnsonGraph {
	sn := g.orig[s].ID()
	for u, e := range g.succ {
		if u < sn {
			g.nodes.Remove(u)
			delete(g.succ, u)
			continue
		}
		for v := range e {
			if v < sn {
				g.succ[u].Remove(v)
			}
		}
	}
	return g
}

// sccSubGraph returns the graph of the tarjan's strongly connected
// components with each SCC containing at least min vertices.
// sccSubGraph returns nil if there is no SCC with at least min
// members.
func (g johnsonGraph) sccSubGraph(sccs [][]graph.Node, min int) johnsonGraph {
	if len(g.nodes) == 0 {
		g.nodes = nil
		g.succ = nil
		return g
	}
	sub := johnsonGraph{
		orig:  g.orig,
		index: g.index,
		nodes: make(internal.IntSet),
		succ:  make(map[int]internal.IntSet),
	}

	var n int
	for _, scc := range sccs {
		if len(scc) < min {
			continue
		}
		n++
		for _, u := range scc {
			for _, v := range scc {
				if _, ok := g.succ[u.ID()][v.ID()]; ok {
					if sub.succ[u.ID()] == nil {
						sub.succ[u.ID()] = make(internal.IntSet)
						sub.nodes.Add(u.ID())
					}
					sub.nodes.Add(v.ID())
					sub.succ[u.ID()].Add(v.ID())
				}
			}
		}
	}
	if n == 0 {
		g.nodes = nil
		g.succ = nil
		return g
	}

	return sub
}

// Nodes is required to satisfy Tarjan.
func (g johnsonGraph) Nodes() []graph.Node {
	n := make([]graph.Node, 0, len(g.nodes))
	for id := range g.nodes {
		n = append(n, johnsonGraphNode(id))
	}
	return n
}

// Successors is required to satisfy Tarjan.
func (g johnsonGraph) From(n graph.Node) []graph.Node {
	adj := g.succ[n.ID()]
	if len(adj) == 0 {
		return nil
	}
	succ := make([]graph.Node, 0, len(adj))
	for n := range adj {
		succ = append(succ, johnsonGraphNode(n))
	}
	return succ
}

func (johnsonGraph) Has(graph.Node) bool {
	panic("search: unintended use of johnsonGraph")
}
func (johnsonGraph) HasEdge(_, _ graph.Node) bool {
	panic("search: unintended use of johnsonGraph")
}
func (johnsonGraph) Edge(_, _ graph.Node) graph.Edge {
	panic("search: unintended use of johnsonGraph")
}
func (johnsonGraph) HasEdgeFromTo(_, _ graph.Node) bool {
	panic("search: unintended use of johnsonGraph")
}
func (johnsonGraph) To(graph.Node) []graph.Node {
	panic("search: unintended use of johnsonGraph")
}

type johnsonGraphNode int

func (n johnsonGraphNode) ID() int { return int(n) }
