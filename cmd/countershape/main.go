package main

import (
	"os"

	"github.com/nelsonwerd/countershape/internal/reference/app"
	_ "github.com/nelsonwerd/countershape/internal/reference/clistudy"
	_ "github.com/nelsonwerd/countershape/internal/reference/httpstudy"
	"github.com/nelsonwerd/countershape/internal/reference/reproduce"
	"github.com/nelsonwerd/countershape/internal/server"
	webassets "github.com/nelsonwerd/countershape/web"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "studio" {
		os.Exit(server.RunCLI(os.Args[2:], webassets.Assets(), os.Stdout, os.Stderr))
	}
	os.Exit(app.Run(os.Args[1:], app.Runtime{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Columns: app.TerminalColumns(os.Getenv("COLUMNS")),
		Studies: reproduce.Studies(),
	}))
}
