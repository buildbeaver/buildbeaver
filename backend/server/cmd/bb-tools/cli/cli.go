package cli

import (
	"fmt"
	"log"
	"os"
)

var Stderr = log.New(os.Stderr, "", 0)
var Stdout = log.New(os.Stdout, "", 0)

func Exit(err error) {
	if err != nil {
		Stderr.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

// AskForConfirmation prompts the user in the command line for an 'are you sure' response, using
// the supplied prompt. The user must respond either "Y" (with a capital Y) or there are a variety of
// acceptable no responses including "n", "N", "no", "No" and "NO".
// If skipConfirmation is true then the confirmation then 'true' will always be returned without seeking
// interactive confirmation from the user.
// Returns true if the user confirmed, or false if not confirmed.
func AskForConfirmation(prompt string, skipConfirmation bool) bool {
	if skipConfirmation {
		return true
	}

	Stdout.Printf("%s (please type Y or N): ", prompt)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		Stdout.Printf("Error reading confirmation response: %s", err)
		return false
	}

	switch response {
	case "Y":
		return true
	case "n", "N", "no", "No", "NO":
		return false
	default:
		return AskForConfirmation("Please type (capital) Y for Yes or N for No and press enter", false)
	}
}
