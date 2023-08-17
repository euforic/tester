package main

import (
	"fmt"
	"os"
)

func main() {
	userName := os.Getenv("USERNAME_INPUT")
	fmt.Printf("Hello %s!\n", userName)
}
