package common

func Map[B, C any](bs []B, fn func(B) C) []C {
	result := make([]C, len(bs))
	for i, b := range bs {
		result[i] = fn(b)
	}
	return result
}

func PtrToString(val *string) string {
	if val != nil {
		return *val
	}
	return ""
}

func EqualUnordered[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	counts := make(map[T]int)
	for _, val := range a {
		counts[val]++
	}

	for _, val := range b {
		if counts[val] == 0 {
			return false
		}
		counts[val]-- // For detecting duplicates
	}
	return true
}
