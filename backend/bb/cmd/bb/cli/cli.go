package cli

import (
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
