package simple

import "context"

type Generator struct {
	counter int
}

func New() *Generator {
	//nolint:exhaustruct
	return &Generator{}
}

func (g *Generator) GetID(_ context.Context) (int, error) {
	g.counter++

	return g.counter, nil
}
