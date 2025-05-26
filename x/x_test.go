package x_test

import (
	"fmt"
	"os"

	"github.com/tmc/mkprog/x"
)

func Example() {
	// This example demonstrates basic usage of the x package
	result := x.Foo()
	fmt.Println(result)
	// Output: foo
}

func Example_withContext() {
	// This example demonstrates how to use the Foo function
	// in a slightly more complex scenario
	result := x.Foo()
	fmt.Fprintf(os.Stdout, "The function returned: %s\n", result)
	// Output: The function returned: foo
}

// ExampleFoo demonstrates the Foo function specifically
func ExampleFoo() {
	fmt.Println(x.Foo())
	// Output: foo
}

// ExampleFoo_multiple demonstrates how to verify multiple lines of output
func ExampleFoo_multiple() {
	// Get the result
	result := x.Foo()

	// Use the result multiple times
	fmt.Println("First line:", result)
	fmt.Println("Second line:", result)

	// Output:
	// First line: foo
	// Second line: foo
}
