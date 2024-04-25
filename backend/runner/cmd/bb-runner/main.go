package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/common/util/proc_lock"
	"github.com/buildbeaver/buildbeaver/common/version"
	"github.com/buildbeaver/buildbeaver/runner/app"
)

func main() {
	fmt.Printf("BB Runner v%s\n", version.VersionToString())
	fmt.Printf("Starting with args: %v\n", util.FilterOSArgs(os.Args, app.LogSafeFlags))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	config, err := app.ConfigFromFlags()
	if err != nil {
		log.Fatalf("Error parsing flags: %s", err)
	}
	app, err := app.New(config)
	if err != nil {
		log.Fatalf("Error creating runner: %s", err)
	}

	// TODO: Should we allow multiple runner instances on the same machine?
	lockFile, err := proc_lock.CreateLockFile(proc_lock.RunnerLockFile)
	if err != nil {
		log.Fatalf("Error: Another instance of the BuildBeaver runner is currently running")
	}
	defer lockFile.Close()

	err = app.CleanUpOldResources()
	if err != nil {
		// Log and ignore errors during cleanup
		log.Printf("Warning: errors during resource cleanup: %s", err.Error())
	}

	err = app.Start(ctx)
	if err != nil {
		log.Fatalf("Error starting runner: %s", err)
	}
	defer app.Stop()
	<-ctx.Done()
}
