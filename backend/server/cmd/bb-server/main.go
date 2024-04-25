package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/buildbeaver/buildbeaver/common/util"
	"github.com/buildbeaver/buildbeaver/common/version"
	"github.com/buildbeaver/buildbeaver/server/app"
)

func main() {
	fmt.Printf("BB Server v%s\n", version.VersionToString())
	fmt.Printf("Starting with args: %v\n", util.FilterOSArgs(os.Args, app.LogSafeFlags))

	config, err := app.ConfigFromFlags()
	if err != nil {
		log.Fatalf("Error parsing flags: %s", err)
	}

	app, cleanup, err := app.New(context.Background(), config)
	if err != nil {
		log.Fatalf("Error creating app: %s", err)
	}
	defer cleanup()
	app.CoreAPIServer.Start()
	app.RunnerAPIServer.Start()

	if config.InternalRunnerConfig.StartInternalRunners {
		err = app.InternalRunnerManager.Start()
		defer app.InternalRunnerManager.Stop()
		if err != nil {
			log.Fatalf("Error starting internal runners: %s", err)
		}
	}

	// Wait for SIGINT or SIGTERM before shutting down server
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	err = app.CoreAPIServer.Stop(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}
	err = app.RunnerAPIServer.Stop(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Print("Server shutdown complete")
}
