package sgo

import (
	"fmt"
	"time"
)

func ExampleOrderedPipeline() {
	p := NewOrderedPipeline(2, 2)

	p.AddTask(func() interface{} {
		time.Sleep(2 * time.Second)
		fmt.Println("task one complete.")
		return nil
	}, func(interface{}) {
		fmt.Println("task one collected.")
	})

	p.AddTask(func() interface{} {
		fmt.Println("task two complete.")
		return nil
	}, func(interface{}) {
		fmt.Println("task two collected.")
	})

	p.Close()
	// Output: task two complete.
	// task one complete.
	// task one collected.
	// task two collected.
}

func ExamplePipeline() {
	p := NewPipeline(2, 2)

	p.AddTask(func() interface{} {
		time.Sleep(2 * time.Second)
		fmt.Println("task one complete.")
		return nil
	}, func(interface{}) {
		fmt.Println("task one collected.")
	})

	p.AddTask(func() interface{} {
		fmt.Println("task two complete.")
		return nil
	}, func(interface{}) {
		fmt.Println("task two collected.")
	})

	p.Close()
	// Output: task two complete.
	// task two collected.
	// task one complete.
	// task one collected.
}
