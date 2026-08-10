package main

import (
	"os"

	"github.com/nelsonwerd/countershape/internal/reference/app"
	"github.com/nelsonwerd/countershape/internal/reference/httpstudy"
)

func main() {
	os.Exit(app.Run(os.Args[1:], app.Runtime{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Columns: app.TerminalColumns(os.Getenv("COLUMNS")),
		Studies: map[string]app.StudyHandler{
			"http": httpstudy.Handler(),
		},
	}))
}
