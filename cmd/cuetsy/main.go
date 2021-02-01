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
	// Haven't figured everything out yet about how they're related to the
	// working directory, the argument here, the name of a package given,
	// imports, etc. This is more "quick n' dumb" for now - just enough to
	// test out the contained test file.
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
