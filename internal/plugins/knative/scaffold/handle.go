//go:build ignore

package main

import (
	"fmt"
	"net/http"
)

// Handle processes an incoming HTTP request.
func Handle(res http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(res, "Hello, World!")
}
