package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/buildbeaver/sdk/core/bb/client"
)

func main() {
	ctx := context.Background()

	swaggerConfig := client.NewConfiguration()

	// TODO: Allow configuration of endpoint via command-line
	// Add the supplied endpoint to the start of the list of servers in the config, so it becomes the default
	//endPoint := strings.Trim(dynamicBuildAPI, "/") // remove trailing slash, if present
	//serverURL := fmt.Sprintf("%s/api/v1", endPoint)
	//server := swagger.ServerConfiguration{
	//	URL:         serverURL,
	//	Description: "Default BuildBeaver Core API server URL",
	//}
	//swaggerConfig.Servers = append(swagger.ServerConfigurations{server}, swaggerConfig.Servers...)

	fmt.Printf("Core API Base Path: %s\n", swaggerConfig.Servers[0].URL)
	apiClient := client.NewAPIClient(swaggerConfig)

	createRunnerRequest := client.CreateRunnerRequest{
		Name:                 "test-runner-from-API",
		ClientCertificatePem: "---- NOT A VALID CLIENT CERTIFICATE -----",
	}
	legalEntityID := "legal-entity:dead-beef-cafe"

	runner, response, err := apiClient.RunnersApi.CreateRunner(ctx, legalEntityID).
		CreateRunnerRequest(createRunnerRequest).
		Execute()
	if err != nil {
		fmt.Printf("ERROR returned from CreateRunner: %s\n", err.Error())
		syscall.Exit(1)
	}

	fmt.Printf("CreateRunner returned successfully, response: %v, runner: %v\n", response, runner)
}
