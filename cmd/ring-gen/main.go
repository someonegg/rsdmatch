package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "ring-gen",
		Usage: "Utility for working with scheduling ring",
		Commands: []*cli.Command{
			createCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}

var createCmd = &cli.Command{
	Name:    "create",
	Usage:   "Create a scheduling ring",
	Aliases: []string{"c"},
	Flags: []cli.Flag{
		&cli.Float64Flag{
			Name:     "bw",
			Required: true,
			Usage:    "specify the total bandwidth [Gbps]",
		},
		&cli.Float64Flag{
			Name:     "scale",
			Aliases:  []string{"s"},
			Required: false,
			Value:    1.0,
			Usage:    "specify the scale of bandwidth",
		},
		&cli.StringFlag{
			Name:     "node",
			Required: false,
			Value:    "node.json",
			Usage:    "specify the input node.json",
		},
		&cli.StringFlag{
			Name:     "view",
			Required: false,
			Value:    "view.json",
			Usage:    "specify the intput view.json",
		},
		&cli.StringFlag{
			Name:     "ring",
			Required: false,
			Value:    "ring.json",
			Usage:    "specify the output ring.json",
		},
		&cli.IntFlag{
			Name:     "ecn",
			Required: false,
			Value:    5,
			Usage:    "specify the enough node count for a view",
		},
		&cli.Float64Flag{
			Name:     "ras",
			Required: false,
			Value:    50.0,
			Usage:    "specify the remote access score [20.0-80.0]",
		},
		&cli.Float64Flag{
			Name:     "rjs",
			Required: false,
			Value:    80.0,
			Usage:    "specify the reject score [80.0-100.0]",
		},
		&cli.Float64Flag{
			Name:     "ral",
			Required: false,
			Value:    0.1,
			Usage:    "specify the remote access ratio limit [0.0-1.0]",
		},
		&cli.BoolFlag{
			Name:     "dist",
			Required: false,
			Value:    false,
			Usage:    "aggregate by standard region",
		},
		&cli.BoolFlag{
			Name:     "storage",
			Required: false,
			Value:    false,
			Usage:    "allocate storage not bandwidth",
		},
		&cli.BoolFlag{
			Name:     "exclusive",
			Required: false,
			Value:    false,
			Usage:    "allocate exclusively",
		},
		&cli.BoolFlag{
			Name:     "vv",
			Required: false,
			Value:    false,
			Usage:    "verbose mode",
		},
	},
	Action: func(ctx *cli.Context) error {
		var (
			bw            = ctx.Float64("bw")
			scale         = ctx.Float64("scale")
			nodeFile      = ctx.String("node")
			viewFile      = ctx.String("view")
			ringFile      = ctx.String("ring")
			ecn           = ctx.Int("ecn")
			ras           = float32(ctx.Float64("ras"))
			rjs           = float32(ctx.Float64("rjs"))
			ral           = float32(ctx.Float64("ral"))
			distMode      = ctx.Bool("dist")
			storageMode   = ctx.Bool("storage")
			exclusiveMode = ctx.Bool("exclusive")
			verbose       = ctx.Bool("vv")
		)
		if bw <= 0 {
			return errors.New("invalid bw")
		}
		if !(ras >= 20.0 && ras <= 80.0) {
			return errors.New("invalid ras")
		}
		if !(rjs >= ras && rjs <= 100.0) {
			return errors.New("invalid rjs")
		}
		if !(ral >= 0.0 && ral <= 1.0) {
			return errors.New("invalid ral")
		}
		return doCreate(
			ctx.Context, bw, scale,
			nodeFile, viewFile, ringFile,
			ecn, ras, rjs, ral,
			distMode, storageMode, exclusiveMode, verbose)
	},
}
