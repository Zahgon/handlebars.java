// Copyright (c) 2012 Edgar Espina
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.

package jvm

// NewLinkedHashMap creates an insertion-ordered map.
func NewLinkedHashMap() *Map { return &Map{Kind: LinkedHashMap} }

// NewHashMap creates a map with Java HashMap iteration semantics.
func NewHashMap() *Map { return &Map{Kind: HashMap} }

// IsHash reports whether this map is a HashMap.
func (m *Map) IsHash() bool { return m != nil && m.Kind == HashMap }

func (m *Map) find(k any) int {
	if m == nil {
		return -1
	}
	for i := range m.entries {
		if m.entries[i].alive && Equals(m.entries[i].key, k) {
			return i
		}
	}
	return -1
}

// Put adds or replaces a mapping and returns its former value, or nil.
func (m *Map) Put(k, v any) any {
	if i := m.find(k); i >= 0 {
		old := m.entries[i].value
		m.entries[i].value = v
		return old
	}
	e := mapEntry{key: k, value: v, hash: spread(HashCode(k)), alive: true}
	m.entries = append(m.entries, e)
	m.size++
	if m.Kind == HashMap {
		m.putHashIndex(len(m.entries) - 1)
	}
	return nil
}

// Get finds a mapping.
func (m *Map) Get(k any) (any, bool) {
	i := m.find(k)
	if i < 0 {
		return nil, false
	}
	return m.entries[i].value, true
}

// ContainsKey reports whether a key is present.
func (m *Map) ContainsKey(k any) bool { return m.find(k) >= 0 }

// Remove removes a mapping and returns its former value, or nil.
func (m *Map) Remove(k any) any {
	i := m.find(k)
	if i < 0 {
		return nil
	}
	m.entries[i].alive = false
	m.size--
	if m.Kind == HashMap {
		bucket := int(uint32(m.entries[i].hash) & uint32(len(m.table)-1))
		var root, node *rbNode
		if m.tree[bucket] {
			root = m.roots[bucket]
			if root == nil {
				indices := append([]int(nil), m.table[bucket]...)
				sortEntryIndices(indices)
				root = m.buildTree(indices)
				m.roots[bucket] = root
			}
			node = findRBNode(root, i)
		}
		for p, entryIndex := range m.table[bucket] {
			if entryIndex == i {
				m.table[bucket] = append(m.table[bucket][:p], m.table[bucket][p+1:]...)
				break
			}
		}
		if node != nil && len(m.table[bucket]) > 0 {
			if root == nil || root.right == nil || root.left == nil || root.left.left == nil {
				m.tree[bucket], m.roots[bucket] = false, nil
			} else {
				root = removeRBNode(root, node)
				m.roots[bucket] = root
				m.moveRootEntryFirst(bucket, root.entry)
			}
		}
	}
	return m.entries[i].value
}

// Len returns the map size.
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	return m.size
}

// Keys returns keys in Java iteration order.
func (m *Map) Keys() []any {
	es := m.Entries()
	out := make([]any, len(es))
	for i := range es {
		out[i] = es[i].Key
	}
	return out
}

// Entries returns entries in Java iteration order.
func (m *Map) Entries() []Entry {
	if m == nil {
		return nil
	}
	out := make([]Entry, 0, m.size)
	if m.Kind == LinkedHashMap {
		for _, e := range m.entries {
			if e.alive {
				out = append(out, Entry{e.key, e.value})
			}
		}
		return out
	}
	for _, bucket := range m.table {
		for _, i := range bucket {
			e := m.entries[i]
			if e.alive {
				out = append(out, Entry{e.key, e.value})
			}
		}
	}
	return out
}

func spread(h int32) int32 { return int32(uint32(h) ^ uint32(h)>>16) }

func (m *Map) putHashIndex(entryIndex int) {
	if len(m.table) == 0 {
		m.table = make([][]int, 16)
		m.tree = make([]bool, 16)
		m.roots = make([]*rbNode, 16)
		m.limit = 12
	}
	e := m.entries[entryIndex]
	b := int(uint32(e.hash) & uint32(len(m.table)-1))
	if m.tree[b] {
		parent := -1
		if root := m.roots[b]; root != nil {
			p := root
			for {
				dir := m.treeCompare(entryIndex, p.entry)
				if dir <= 0 {
					if p.left == nil {
						parent = p.entry
						x := &rbNode{entry: entryIndex, parent: p}
						p.left = x
						root = balanceInsertion(root, x)
						break
					}
					p = p.left
				} else {
					if p.right == nil {
						parent = p.entry
						x := &rbNode{entry: entryIndex, parent: p}
						p.right = x
						root = balanceInsertion(root, x)
						break
					}
					p = p.right
				}
			}
			m.roots[b] = root
		} else {
			parent = m.treeInsertionParent(b, entryIndex)
		}
		at := len(m.table[b])
		for i, old := range m.table[b] {
			if old == parent {
				at = i + 1
				break
			}
		}
		m.table[b] = append(m.table[b], 0)
		copy(m.table[b][at+1:], m.table[b][at:])
		m.table[b][at] = entryIndex
		if m.roots[b] != nil {
			m.moveRootEntryFirst(b, m.roots[b].entry)
		} else {
			m.moveTreeRootFirst(b)
		}
	} else {
		m.table[b] = append(m.table[b], entryIndex)
	}
	// putVal's binCount starts at zero for the head; the check against
	// TREEIFY_THRESHOLD-1 therefore fires when the ninth node is appended.
	if !m.tree[b] && len(m.table[b]) >= 9 {
		if len(m.table) < 64 {
			m.resize()
		} else {
			m.tree[b] = true
			m.moveTreeRootFirst(b)
		}
	}
	if m.size > m.limit {
		m.resize()
	}
}

func (m *Map) treeInsertionParent(bucket, entry int) int {
	indices := append([]int(nil), m.table[bucket]...)
	for i := 1; i < len(indices); i++ {
		for j := i; j > 0 && indices[j] < indices[j-1]; j-- {
			indices[j], indices[j-1] = indices[j-1], indices[j]
		}
	}
	root := m.buildTree(indices)
	if root == nil {
		return -1
	}
	p := root
	for {
		dir := m.treeCompare(entry, p.entry)
		if dir <= 0 {
			if p.left == nil {
				return p.entry
			}
			p = p.left
		} else {
			if p.right == nil {
				return p.entry
			}
			p = p.right
		}
	}
}

func (m *Map) resize() {
	old := m.table
	oldTree := m.tree
	if len(old) >= 1<<30 {
		m.limit = int(^uint(0) >> 1)
		return
	}
	m.table = make([][]int, len(old)*2)
	m.tree = make([]bool, len(old)*2)
	m.roots = make([]*rbNode, len(old)*2)
	m.limit = len(m.table) * 3 / 4
	for b, bucket := range old {
		for _, i := range bucket {
			nb := b
			if uint32(m.entries[i].hash)&uint32(len(old)) != 0 {
				nb += len(old)
			}
			m.table[nb] = append(m.table[nb], i)
		}
		if oldTree[b] {
			if len(m.table[b]) > 6 {
				m.tree[b] = true
			}
			if len(m.table[b+len(old)]) > 6 {
				m.tree[b+len(old)] = true
			}
		}
	}
}

type rbNode struct {
	entry               int
	parent, left, right *rbNode
	red                 bool
}

func (m *Map) moveTreeRootFirst(bucket int) {
	indices := append([]int(nil), m.table[bucket]...)
	// Tree shape depends on insertion sequence, not on linked-bin root moves.
	sortEntryIndices(indices)
	root := m.treeRoot(indices)
	if root < 0 {
		return
	}
	linked := m.table[bucket]
	for i, v := range linked {
		if v == root && i != 0 {
			copy(linked[1:i+1], linked[0:i])
			linked[0] = root
			return
		}
	}
}

func sortEntryIndices(indices []int) {
	for i := 1; i < len(indices); i++ {
		for j := i; j > 0 && indices[j] < indices[j-1]; j-- {
			indices[j], indices[j-1] = indices[j-1], indices[j]
		}
	}
}

func (m *Map) moveRootEntryFirst(bucket, root int) {
	linked := m.table[bucket]
	for i, entry := range linked {
		if entry == root && i != 0 {
			copy(linked[1:i+1], linked[:i])
			linked[0] = root
			return
		}
	}
}

func findRBNode(n *rbNode, entry int) *rbNode {
	if n == nil {
		return nil
	}
	if n.entry == entry {
		return n
	}
	if found := findRBNode(n.left, entry); found != nil {
		return found
	}
	return findRBNode(n.right, entry)
}

func removeRBNode(root, p *rbNode) *rbNode {
	pl, pr := p.left, p.right
	var replacement *rbNode
	if pl != nil && pr != nil {
		s := pr
		for s.left != nil {
			s = s.left
		}
		s.red, p.red = p.red, s.red
		sr, pp := s.right, p.parent
		if s == pr {
			p.parent = s
			s.right = p
		} else {
			sp := s.parent
			p.parent = sp
			if s == sp.left {
				sp.left = p
			} else {
				sp.right = p
			}
			s.right = pr
			pr.parent = s
		}
		p.left = nil
		p.right = sr
		if sr != nil {
			sr.parent = p
		}
		s.left = pl
		pl.parent = s
		s.parent = pp
		if pp == nil {
			root = s
		} else if p == pp.left {
			pp.left = s
		} else {
			pp.right = s
		}
		if sr != nil {
			replacement = sr
		} else {
			replacement = p
		}
	} else if pl != nil {
		replacement = pl
	} else if pr != nil {
		replacement = pr
	} else {
		replacement = p
	}
	if replacement != p {
		pp := p.parent
		replacement.parent = pp
		if pp == nil {
			root = replacement
			replacement.red = false
		} else if p == pp.left {
			pp.left = replacement
		} else {
			pp.right = replacement
		}
		p.left, p.right, p.parent = nil, nil, nil
	}
	r := root
	if !p.red {
		r = balanceDeletion(root, replacement)
	}
	if replacement == p {
		pp := p.parent
		p.parent = nil
		if pp != nil {
			if p == pp.left {
				pp.left = nil
			} else if p == pp.right {
				pp.right = nil
			}
		}
	}
	return r
}

func (m *Map) treeRoot(indices []int) int {
	root := m.buildTree(indices)
	if root == nil {
		return -1
	}
	return root.entry
}

func (m *Map) buildTree(indices []int) *rbNode {
	var root *rbNode
	for _, entry := range indices {
		x := &rbNode{entry: entry, red: true}
		if root == nil {
			x.red = false
			root = x
			continue
		}
		p := root
		for {
			dir := m.treeCompare(entry, p.entry)
			if dir <= 0 {
				if p.left == nil {
					p.left = x
					x.parent = p
					break
				}
				p = p.left
			} else {
				if p.right == nil {
					p.right = x
					x.parent = p
					break
				}
				p = p.right
			}
		}
		root = balanceInsertion(root, x)
	}
	return root
}

func (m *Map) treeCompare(a, b int) int {
	x, y := m.entries[a], m.entries[b]
	if x.hash < y.hash {
		return -1
	}
	if x.hash > y.hash {
		return 1
	}
	if ClassName(x.key) == ClassName(y.key) {
		if c, err := CompareTo(x.key, y.key); err == nil && c != 0 {
			return c
		}
	}
	if ClassName(x.key) < ClassName(y.key) {
		return -1
	}
	if ClassName(x.key) > ClassName(y.key) {
		return 1
	}
	// Java falls back to identityHashCode. Entry allocation order is the only
	// deterministic identity available in the Go model.
	if a < b {
		return -1
	}
	return 1
}

func balanceInsertion(root, x *rbNode) *rbNode {
	x.red = true
	for {
		xp := x.parent
		if xp == nil {
			x.red = false
			return x
		}
		if !xp.red || xp.parent == nil {
			return root
		}
		xpp := xp.parent
		if xp == xpp.left {
			y := xpp.right
			if y != nil && y.red {
				y.red = false
				xp.red = false
				xpp.red = true
				x = xpp
			} else {
				if x == xp.right {
					x = xp
					root = rotateLeft(root, x)
					xp = x.parent
					if xp != nil {
						xpp = xp.parent
					} else {
						xpp = nil
					}
				}
				if xp != nil {
					xp.red = false
					if xpp != nil {
						xpp.red = true
						root = rotateRight(root, xpp)
					}
				}
			}
		} else {
			y := xpp.left
			if y != nil && y.red {
				y.red = false
				xp.red = false
				xpp.red = true
				x = xpp
			} else {
				if x == xp.left {
					x = xp
					root = rotateRight(root, x)
					xp = x.parent
					if xp != nil {
						xpp = xp.parent
					} else {
						xpp = nil
					}
				}
				if xp != nil {
					xp.red = false
					if xpp != nil {
						xpp.red = true
						root = rotateLeft(root, xpp)
					}
				}
			}
		}
	}
}

func balanceDeletion(root, x *rbNode) *rbNode {
	for {
		if x == nil || x == root {
			return root
		}
		xp := x.parent
		if xp == nil {
			x.red = false
			return x
		}
		if x.red {
			x.red = false
			return root
		}
		if xp.left == x {
			xpr := xp.right
			if xpr != nil && xpr.red {
				xpr.red = false
				xp.red = true
				root = rotateLeft(root, xp)
				xp = x.parent
				if xp != nil {
					xpr = xp.right
				} else {
					xpr = nil
				}
			}
			if xpr == nil {
				x = xp
				continue
			}
			sl, sr := xpr.left, xpr.right
			if (sr == nil || !sr.red) && (sl == nil || !sl.red) {
				xpr.red = true
				x = xp
				continue
			}
			if sr == nil || !sr.red {
				if sl != nil {
					sl.red = false
				}
				xpr.red = true
				root = rotateRight(root, xpr)
				xp = x.parent
				if xp != nil {
					xpr = xp.right
				} else {
					xpr = nil
				}
			}
			if xpr != nil {
				xpr.red = xp != nil && xp.red
				if xpr.right != nil {
					xpr.right.red = false
				}
			}
			if xp != nil {
				xp.red = false
				root = rotateLeft(root, xp)
			}
			x = root
		} else {
			xpl := xp.left
			if xpl != nil && xpl.red {
				xpl.red = false
				xp.red = true
				root = rotateRight(root, xp)
				xp = x.parent
				if xp != nil {
					xpl = xp.left
				} else {
					xpl = nil
				}
			}
			if xpl == nil {
				x = xp
				continue
			}
			sl, sr := xpl.left, xpl.right
			if (sl == nil || !sl.red) && (sr == nil || !sr.red) {
				xpl.red = true
				x = xp
				continue
			}
			if sl == nil || !sl.red {
				if sr != nil {
					sr.red = false
				}
				xpl.red = true
				root = rotateLeft(root, xpl)
				xp = x.parent
				if xp != nil {
					xpl = xp.left
				} else {
					xpl = nil
				}
			}
			if xpl != nil {
				xpl.red = xp != nil && xp.red
				if xpl.left != nil {
					xpl.left.red = false
				}
			}
			if xp != nil {
				xp.red = false
				root = rotateRight(root, xp)
			}
			x = root
		}
	}
}

func rotateLeft(root, p *rbNode) *rbNode {
	r := p.right
	if r == nil {
		return root
	}
	p.right = r.left
	if r.left != nil {
		r.left.parent = p
	}
	r.parent = p.parent
	if p.parent == nil {
		root = r
	} else if p.parent.left == p {
		p.parent.left = r
	} else {
		p.parent.right = r
	}
	r.left = p
	p.parent = r
	return root
}

func rotateRight(root, p *rbNode) *rbNode {
	l := p.left
	if l == nil {
		return root
	}
	p.left = l.right
	if l.right != nil {
		l.right.parent = p
	}
	l.parent = p.parent
	if p.parent == nil {
		root = l
	} else if p.parent.right == p {
		p.parent.right = l
	} else {
		p.parent.left = l
	}
	l.right = p
	p.parent = l
	return root
}
