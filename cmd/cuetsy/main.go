package main

import (
	"fmt"
	"log"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/load"
	"github.com/grafana/cuetsy"
	"github.com/urfave/cli/v2"
)

type Options struct {
	CuePath         string
	Export          bool
	DestinationPath string
}

func convert(options Options) error {
	wd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}

	fi, err := os.Stat(options.CuePath)
	if err != nil {
		return err
	}
	if fi.IsDir() || !fi.Mode().IsRegular() {
		return errors.New("must provide path to exactly one .cue file\n")
	}

	loadedInstances := load.Instances([]string{options.CuePath}, nil)
	instances := cue.Build(loadedInstances)
	// Given the above input constraints, there _should_ only ever be a
	// single element in this slice.
	for _, inst := range instances {
		b, err := cuetsy.Generate(inst.Value(), cuetsy.Config{Export: options.Export})
		if err != nil {
			errors.Print(os.Stderr, err, &errors.Config{
				Cwd: wd,
			})
			os.Exit(1)
		}

		if options.DestinationPath == "-" {
			fmt.Println(string(b))
		} else {
			fd, err := os.Create(options.DestinationPath)
			if err != nil {
				fmt.Println(err.Error())
				return err
			}

			fmt.Fprint(fd, string(b))
		}

	}

	return nil
}

func main() {
	options := Options{}
	app := &cli.App{
		Name:  "cuetsy",
		Usage: "Converting CUE objects to their TypeScript equivalent.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "cuepath",
				Aliases:     []string{"c", "p"},
				Usage:       "The cue file path.",
				Destination: &options.CuePath,
			},
			&cli.BoolFlag{
				Name:        "export",
				Aliases:     []string{"e"},
				Usage:       "Add the export keyword to the ts file.",
				Destination: &options.Export,
			},
			&cli.StringFlag{
				Name:        "destination",
				Aliases:     []string{"d", "dest"},
				Usage:       "The path to the destination ts file. when this option is '-', output to stdout.",
				Destination: &options.DestinationPath,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if len(os.Args) == 1 {
				fmt.Fprint(os.Stderr,
					`
must provide path to exactly one .cue file
Try 'cuetsy --help' for more information.
`)
				os.Exit(2)
			}

			// keep cuetsy [cuefile] pattern
			if len(os.Args) == 2 {
				options.CuePath = os.Args[1]
			}

			if options.DestinationPath == "" && len(os.Args) > 2 {
				options.DestinationPath = options.CuePath[:len(options.CuePath)-3] + "ts"
			}

			convert(options)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)

}
