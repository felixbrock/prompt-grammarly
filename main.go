package main

import "github.com/felixbrock/lemonai/internal/app"

/*
- Remove credentials from all files
- Check for SQL injection
- Check for XSS
- Block IP addresses: Too many requests
*/

func main() {
	app.App()
}
