package components

import (
	"context"
	"io"
)

type Component interface {
	Render(ctx context.Context, w io.Writer) error
}

func hello() *Component {
	return index()
}
