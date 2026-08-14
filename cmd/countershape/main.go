package main

import (
	"os"

	"github.com/nelsonwerd/countershape/internal/reference/app"
	_ "github.com/nelsonwerd/countershape/internal/reference/clistudy"
	_ "github.com/nelsonwerd/countershape/internal/reference/httpstudy"
	"github.com/nelsonwerd/countershape/internal/reference/reproduce"
)

func main() {
	os.Exit(app.Run(os.Args[1:], app.Runtime{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Columns: app.TerminalColumns(os.Getenv("COLUMNS")),
		Studies: reproduce.Studies(),
	}))
}
