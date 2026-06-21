package cli

// levenshtein returns the edit distance between a and b. Used for "did you
// mean" suggestions on mistyped commands and package names.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// closest returns the candidate nearest to target and its edit distance.
// candidates must be non-empty.
func closest(target string, candidates []string) (string, int) {
	best, bestDist := "", 1<<30
	for _, c := range candidates {
		if d := levenshtein(target, c); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best, bestDist
}

// suggestNames returns up to limit candidates within maxDist of target,
// nearest first. Used to offer alternatives when a package name misses.
func suggestNames(target string, candidates []string, maxDist, limit int) []string {
	type scored struct {
		name string
		dist int
	}
	var hits []scored
	for _, c := range candidates {
		if d := levenshtein(target, c); d <= maxDist {
			hits = append(hits, scored{c, d})
		}
	}
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].dist < hits[j-1].dist; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	out := make([]string, 0, limit)
	for _, h := range hits {
		if len(out) >= limit {
			break
		}
		out = append(out, h.name)
	}
	return out
}
