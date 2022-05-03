package utils

type Set[K comparable] struct {
	m map[K]struct{}
}

func NewSet[K comparable]() *Set[K] {
	return &Set[K]{m: map[K]struct{}{}}
}

func (s *Set[K]) Has(k K) bool {
	_, ok := s.m[k]
	return ok
}

func (s *Set[K]) Add(k K) {
	s.m[k] = struct{}{}
}

func (s *Set[K]) Remove(k K) {
	delete(s.m, k)
}

func (s *Set[K]) Slice() []K {
	ret := make([]K, 0, len(s.m))
	for k := range s.m {
		ret = append(ret, k)
	}
	return ret
}
