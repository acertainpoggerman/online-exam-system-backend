package common

func Map[B, C any](bs []B, fn func(B) C) []C {
	result := make([]C, len(bs))
	for i, b := range bs {
		result[i] = fn(b)
	}
	return result
}

func ZeroNilString(val *string) string {
	if val != nil {
		return *val
	}
	return ""
}
