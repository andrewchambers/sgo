package sgo

import (
	"context"
	"fmt"
	"time"
)

func ExampleGo() {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)

	pg := NewProcessGroup(ctx)

	pg.Go(func(ctx context.Context) {
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("'a' complete")
		case <-ctx.Done():
			fmt.Println("'a' cancelled")
		}
	})

	pg.Go(func(ctx context.Context) {
		select {
		case <-time.After(3 * time.Second):
			fmt.Println("'b' complete")
		case <-ctx.Done():
			fmt.Println("'b' cancelled")
		}
	})

	<-pg.Done()
	fmt.Println("all exited")

	// Output: 'a' complete
	// 'b' cancelled
	// all exited
}
