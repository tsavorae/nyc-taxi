package main

import "math/rand"

func mean(samples []Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range samples {
		sum += s.Target
	}
	return sum / float64(len(samples))
}

func variance(samples []Sample) float64 {
	if len(samples) == 0 {
		return 0
	}
	mu := mean(samples)
	v := 0.0
	for _, s := range samples {
		d := s.Target - mu
		v += d * d
	}
	return v / float64(len(samples))
}

// bestSplit tries maxFeatures randomly chosen features with up to 20 threshold
// candidates each. Returns the split that reduces variance the most.
func bestSplit(samples []Sample, maxFeatures int) (feature int, threshold float64, gain float64) {
	n := len(samples)
	if n < 2 {
		return
	}
	totalVar := variance(samples)
	featureIdx := rand.Perm(4)[:maxFeatures]

	for _, f := range featureIdx {
		perm := rand.Perm(n)
		limit := 20
		if n < limit {
			limit = n
		}
		for i := 0; i < limit; i++ {
			t := samples[perm[i]].Features[f]
			var left, right []Sample
			for _, s := range samples {
				if s.Features[f] <= t {
					left = append(left, s)
				} else {
					right = append(right, s)
				}
			}
			if len(left) == 0 || len(right) == 0 {
				continue
			}
			g := totalVar - (float64(len(left))*variance(left)+float64(len(right))*variance(right))/float64(n)
			if g > gain {
				gain, feature, threshold = g, f, t
			}
		}
	}
	return
}

// buildTree grows a CART regression tree and serializes it into a flat node
// slice. Returns the index of the root node.
func buildTree(samples []Sample, depth, maxDepth, maxFeatures int, nodes *[]Node) int {
	idx := len(*nodes)
	*nodes = append(*nodes, Node{})

	if depth >= maxDepth || len(samples) < 10 {
		(*nodes)[idx] = Node{IsLeaf: true, Value: mean(samples), Left: -1, Right: -1}
		return idx
	}

	feature, threshold, gain := bestSplit(samples, maxFeatures)
	if gain <= 0 {
		(*nodes)[idx] = Node{IsLeaf: true, Value: mean(samples), Left: -1, Right: -1}
		return idx
	}

	var left, right []Sample
	for _, s := range samples {
		if s.Features[feature] <= threshold {
			left = append(left, s)
		} else {
			right = append(right, s)
		}
	}
	if len(left) == 0 || len(right) == 0 {
		(*nodes)[idx] = Node{IsLeaf: true, Value: mean(samples), Left: -1, Right: -1}
		return idx
	}

	(*nodes)[idx] = Node{Feature: feature, Threshold: threshold, Left: -1, Right: -1}
	(*nodes)[idx].Left = buildTree(left, depth+1, maxDepth, maxFeatures, nodes)
	(*nodes)[idx].Right = buildTree(right, depth+1, maxDepth, maxFeatures, nodes)
	return idx
}
