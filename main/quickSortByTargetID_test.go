package main

import "bytes"

func QuickSortByTargetID(nodes []Node, targetID []byte)[]Node{
	if len(nodes) < 2 {
		return nodes
	}
	left, right := 0, len(nodes) - 1

	// Pick a pivot
	pivotIndex := rand.Int() % len(nodes)

	// Move the pivot to the right
	nodes[pivotIndex], nodes[right] = nodes[right], nodes[pivotIndex]

	// Pile elements smaller than the pivot on the left
	for i := range nodes {
		if a[i] < a[right] {
			a[i], a[left] = a[left], a[i]
			left++
		}
	}

	// Place the pivot after the last smaller element
	nodes[left], nodes[right] = nodes[right], nodes[left]

	// Go down the rabbit hole
	QuickSortByTargetID(nodes[:left])
	QuickSortByTargetID(nodes[left + 1:])
	return nodes
}
