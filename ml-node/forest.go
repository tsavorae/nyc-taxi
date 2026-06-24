package main

import "math/rand"

// trainTree builds one tree on a bootstrap sample of the dataset.
func trainTree(samples []Sample, maxDepth, maxFeatures int) Tree {
	n := len(samples)
	boot := make([]Sample, n)
	for i := range boot {
		boot[i] = samples[rand.Intn(n)]
	}
	var nodes []Node
	buildTree(boot, 0, maxDepth, maxFeatures, &nodes)
	return Tree{Nodes: nodes}
}

// predictTree walks the flat node slice and returns the predicted value.
func predictTree(tree Tree, features [4]float64) float64 {
	if len(tree.Nodes) == 0 {
		return 0
	}
	idx := 0
	for {
		n := tree.Nodes[idx]
		if n.IsLeaf {
			return n.Value
		}
		if features[n.Feature] <= n.Threshold {
			idx = n.Left
		} else {
			idx = n.Right
		}
	}
}
