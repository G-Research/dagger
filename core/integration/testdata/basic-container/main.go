package main

import (
	"context"

	"github.com/G-Research/dagger"
)

func main() {
	ctx := context.Background()
	c, err := dagger.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	_, err = c.Container().
		From("alpine:3.16.2").
		WithExec([]string{"echo", "Hello, world!"}).
		Sync(ctx)
	if err != nil {
		panic(err)
	}
}
