// Package splay provides an implementation of a Splay tree. A splay tree
// is a self-balancing binary search tree with amortized O(log N) complexity
// on all operations. Especially, it supports range-based aggregation and
// updation in a simply way. It also has a good performance when data accessing
// has a pattern that the same key is accessed frequently in a short time.
// Generally, its performance againse random operation is not as good as
// a well-tuned red-black tree or AVL tree.
package splay

import (
	g "github.com/zyedidia/generic"
)

type direction int

const (
	dirNone  direction = -1
	dirLeft  direction = 0
	dirRight direction = 1
)

// Tree implements a Splay tree.
type Tree[K, V, A any] struct {
	// null is the sential node representing an empty node.
	// When not splaying, its children are pointing to itself when the tree.
	// When splaying, its children are pointing to a temporary tree that
	// will be attached to the updated root in the reverse order.
	null node[K, V, A]

	// root is the root of the tree.
	root *node[K, V, A]

	// less is the '<' function used to compare keys.
	less g.LessFn[K]

	// aggregator gives a set of functions used to operate on aggregators.
	aggregator Aggregator[V, A]
}

type node[K, V, A any] struct {
	key K
	agg *A

	size int
	chd  [2]*node[K, V, A]
}

func (n *node[K, V, A]) popUp(aggregator Aggregator[V, A]) *node[K, V, A] {
	n.size = 1 + n.chd[dirLeft].size + n.chd[dirRight].size
	n.agg = aggregator.PopUp(n.agg, n.chd[dirLeft].agg, n.chd[dirRight].agg)
	return n
}

func (n *node[K, V, A]) pushDown(aggregator Aggregator[V, A]) *node[K, V, A] {
	aggregator.PushDown(n.agg, n.chd[dirLeft].agg, n.chd[dirRight].agg)
	return n
}

func (t *Tree[K, V, A]) newNode(key K, value V, lchd *node[K, V, A], rchd *node[K, V, A]) *node[K, V, A] {
	return (&node[K, V, A]{
		key: key,
		agg: t.aggregator.FromValue(value),
		chd: [2]*node[K, V, A]{
			lchd,
			rchd,
		},
	}).popUp(t.aggregator)
}

// zig moves one step in a direction during a splay.
// Note: cleanUp() is required after splay.
func (t *Tree[K, V, A]) zig(d direction) {
	newRoot := t.root.pushDown(t.aggregator).chd[d]
	t.root.chd[d] = t.null.chd[d]
	t.null.chd[d] = t.root
	t.root = newRoot
}

// zigzig moves two steps in a direction during a splay.
// Note: cleanUp() is required after splay.
func (t *Tree[K, V, A]) zigzig(d direction) {
	middle := t.root.pushDown(t.aggregator).chd[d]
	newRoot := middle.pushDown(t.aggregator).chd[d]
	middle.chd[d] = t.null.chd[d]
	t.null.chd[d] = middle
	t.root.chd[d] = middle.chd[1^d]
	middle.chd[1^d] = t.root.popUp(t.aggregator)
	t.root = newRoot
}

// cleanUp puts the nodes in the temporary tree (i.e. null.chd)
// back into the tree, attached to the new root in the reverse order.
func (t *Tree[K, V, A]) cleanUp(d direction) {
	// The first node of the reversed temporary tree
	head := t.null.chd[d]
	//  The new child of head after putting head back
	child := t.root.pushDown(t.aggregator).chd[1^d]
	for head != &t.null {
		nextHead := head.pushDown(t.aggregator).chd[d]
		head.chd[d] = child
		child = head.popUp(t.aggregator)
		t.null.chd[d] = nextHead
		head = nextHead
	}
	t.root.chd[1^d] = child
}

// splay performs a splay operation with respect to a given predicate function
// that tells the direction to move.
func (t *Tree[K, V, A]) splay(pred func(*node[K, V, A]) direction) {
	for {
		d1 := pred(t.root)
		if d1 == dirNone || t.root.chd[d1] == &t.null {
			break
		}
		d2 := pred(t.root.chd[d1])
		if d2 == dirNone || t.root.chd[d1].chd[d2] == &t.null {
			t.zig(d1)
			break
		}
		if d1 == d2 {
			t.zigzig(d1)
		} else {
			t.zig(d1)
			t.zig(d2)
		}
	}
	t.cleanUp(dirLeft)
	t.cleanUp(dirRight)
	if t.root != &t.null {
		t.root = t.root.popUp(t.aggregator)
	}
}

// splayNth splays the n-th node in the tree to be the root if 0 <= n < Size().
// Otherwise, splay the closest node.
func (t *Tree[K, V, A]) splayNth(n int) {
	pred := func(cur *node[K, V, A]) direction {
		pos := cur.chd[dirLeft].size
		if n == pos {
			return dirNone
		} else if n < pos {
			return dirLeft
		} else {
			n -= pos + 1
			return dirRight
		}
	}
	t.splay(pred)
}

// splayLowerbound splays the first node >= 'key' to be the root.
// If there is no such node, splay the largest node.
func (t *Tree[K, V, A]) splayLowerbound(key K) {
	t.splay(func(cur *node[K, V, A]) direction {
		if t.less(cur.key, key) {
			return dirRight
		} else {
			return dirLeft
		}
	})
	if t.root != &t.null && t.less(t.root.key, key) {
		t.splayNth(t.root.chd[dirLeft].size + 1)
	}
}

// splayAt splays the node with 'key' to be the root.
// If there is no such node, splay an arbitrary node.
func (t *Tree[K, V, A]) splayAt(key K) {
	t.splay(func(cur *node[K, V, A]) direction {
		if t.less(cur.key, key) {
			return dirRight
		} else if t.less(key, cur.key) {
			return dirLeft
		} else {
			return dirNone
		}
	})
}

// Get returns the value associated with 'key'.
func (t *Tree[K, V, A]) Get(key K) (V, bool) {
	t.splayAt(key)
	if t.root == &t.null || g.Compare(t.root.key, key, t.less) != 0 {
		return t.aggregator.Value(nil), false
	} else {
		return t.aggregator.Value(t.root.agg), true
	}
}

// Put associates 'key' with 'value'.
func (t *Tree[K, V, A]) Put(key K, value V) {
	t.splayLowerbound(key)
	if t.root == &t.null {
		t.root = t.newNode(key, value, &t.null, &t.null)
	} else {
		oldRoot := t.root
		switch g.Compare(t.root.key, key, t.less) {
		case 0:
			t.root.agg = t.aggregator.FromValue(value)
		case 1:
			t.root = t.newNode(key, value, oldRoot.chd[dirLeft], oldRoot)
			oldRoot.chd[dirLeft] = &t.null
			oldRoot.popUp(t.aggregator)
			t.root.popUp(t.aggregator)
		case -1:
			t.root = t.newNode(key, value, oldRoot, &t.null)
		}
	}
}

// Remove removes the value associated with 'key'.
func (t *Tree[K, V, A]) Remove(key K) {
	t.splayAt(key)
	if t.root == &t.null || g.Compare(t.root.key, key, t.less) != 0 {
		return
	}
	oldRoot := t.root
	if t.root.chd[dirRight] != &t.null {
		t.root = t.root.chd[dirRight]
		t.splayNth(0)
		t.root.chd[dirLeft] = oldRoot.chd[dirLeft]
		t.root.popUp(t.aggregator)
	} else {
		t.root = t.root.chd[dirLeft]
	}
}

// Size returns the number of elements in the tree.
func (t *Tree[K, V, A]) Size() int {
	return t.root.size
}

// Each calls 'fn' on every node in the tree in order
func (t *Tree[K, V, A]) Each(fn func(K, V)) {
	for i := 0; i < t.root.size; i++ {
		t.splayNth(i)
		fn(t.root.key, t.aggregator.Value(t.root.agg))
	}
}

// New returns an empty Splay tree.
func New[K, V, A any](less g.LessFn[K], aggregator Aggregator[V, A]) *Tree[K, V, A] {
	ret := &Tree[K, V, A]{
		less:       less,
		aggregator: aggregator,
		null: node[K, V, A]{
			agg:  nil,
			size: 0,
		},
	}
	ret.root = &ret.null
	ret.null.chd[dirLeft] = &ret.null
	ret.null.chd[dirRight] = &ret.null
	return ret
}

// Range returns the aggregator associated with key range [l, r),
// which can be used to obtain statistics or do range-based update.
// Note that the range is only valid before next operation on the tree.
func (t *Tree[K, V, A]) Range(l, r K) *A {
	t.splayLowerbound(l)
	if t.root == &t.null || !t.less(t.root.key, r) {
		// Minumum >= r
		return nil
	}
	rank := t.root.chd[dirLeft].size - 1
	t.splayLowerbound(r)
	if t.root == &t.null || t.less(t.root.key, l) {
		// Maximum < l
		return nil
	}
	t.zig(dirLeft) // Make sure the current root is on the right
	t.splayNth(rank)
	if t.less(t.root.key, l) {
		if t.less(t.root.chd[dirRight].key, r) {
			// Maximum < r
			return t.root.chd[dirRight].agg
		} else {
			return t.root.chd[dirRight].chd[dirLeft].agg
		}
	} else {
		// Minimum >= l
		t.splayLowerbound(r)
		if t.less(t.root.key, r) {
			// Maximum < r
			return t.root.agg
		} else {
			return t.root.chd[dirLeft].agg
		}
	}
}
