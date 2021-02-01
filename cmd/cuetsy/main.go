package main

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/sdboyer/cuetsy/encoder"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	// loadedInstances := load.Instances([]string{os.Args[1]}, nil)
	loadedInstances := load.Instances([]string{"."}, &load.Config{Package: "cuetsy"})
	instances := cue.Build(loadedInstances)
	for _, inst := range instances {
		b, err := encoder.Generate(inst)
		if err != nil {
			errors.Print(os.Stderr, err, &errors.Config{
				Cwd: wd,
			})
			os.Exit(1)
		}
		fmt.Println(string(b))
	}
}
