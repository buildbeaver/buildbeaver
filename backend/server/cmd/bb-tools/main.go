package main

import (
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands"
	_ "github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands/admin"
	_ "github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands/dump"
	_ "github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands/migrate"
)

func main() {
	commands.Execute()
}
