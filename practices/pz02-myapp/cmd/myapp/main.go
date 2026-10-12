package main

import (
	"os"

	"github.com/Pondaaaaaa/GO/practices/pz02-myapp/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		os.Exit(1)
	}
}
