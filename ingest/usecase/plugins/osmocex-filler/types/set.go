package osmocexfillertypes

type Set[T comparable] struct {
	m map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{m: make(map[T]struct{})}
}

func (s *Set[T]) Add(value T) {
	s.m[value] = struct{}{}
}

func (s *Set[T]) Remove(value T) {
	delete(s.m, value)
}

func (s *Set[T]) Contains(value T) bool {
	_, c := s.m[value]
	return c
}

func (s *Set[T]) Values() []T {
	values := make([]T, 0, len(s.m))
	for value := range s.m {
		values = append(values, value)
	}
	return values
}

func (s *Set[T]) Len() int {
	return len(s.m)
}
