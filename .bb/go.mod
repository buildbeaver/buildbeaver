module github.com/buildbeaver/build

go 1.19

replace github.com/buildbeaver/sdk/dynamic/bb => ../sdk/dynamic/go

require github.com/buildbeaver/sdk/dynamic/bb v0.0.0-00010101000000-000000000000

require (
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.2 // indirect
	golang.org/x/net v0.17.0 // indirect
)
