package slicex

func GroupBy[T any, K comparable](slice []T, getKey func(T) K) map[K][]T {
	m := make(map[K][]T)
	for _, v := range slice {
		key := getKey(v)
		m[key] = append(m[key], v)
	}
	return m
}
