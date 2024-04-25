package main

import (
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/commands"
	_ "github.com/buildbeaver/buildbeaver/bb/cmd/bb/commands/cleanup"
	_ "github.com/buildbeaver/buildbeaver/bb/cmd/bb/commands/run"
)

func main() {
	commands.Execute()
}
