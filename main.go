package main

import (
	"context"
	"os"

	"github.com/kidixdev/PromptSensei/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
