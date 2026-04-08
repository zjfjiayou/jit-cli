module jit-cli

go 1.23.0

require (
	github.com/itchyny/gojq v0.12.18
	github.com/itchyny/timefmt-go v0.1.7 // indirect
	github.com/spf13/cobra v1.10.2
	github.com/zalando/go-keyring v0.2.8
)

replace github.com/itchyny/gojq => ./third_party/gojq

replace github.com/itchyny/timefmt-go => ./third_party/timefmt-go

replace github.com/spf13/cobra => ./third_party/cobra

replace github.com/zalando/go-keyring => ./third_party/go-keyring
