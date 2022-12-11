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
			Name:     "alloc",
			Required: false,
			Value:    "alloc.json",
			Usage:    "specify the output alloc.json",
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
			Usage:    "specify the remote access limit [0.0-1.0]",
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
			bw        = ctx.Float64("bw")
			scale     = ctx.Float64("scale")
			nodeFile  = ctx.String("node")
			viewFile  = ctx.String("view")
			allocFile = ctx.String("alloc")
			ecn       = ctx.Int("ecn")
			ras       = float32(ctx.Float64("ras"))
			rjs       = float32(ctx.Float64("rjs"))
			ral       = float32(ctx.Float64("ral"))
			verbose   = ctx.Bool("vv")
		)
		if bw <= 0 {
			return errors.New("invalid bw")
		}
		if !(ras >= 20.0 && ras <= 80.0) {
			return errors.New("invalid ras")
		}
		if !(rjs >= 80.0 && rjs <= 100.0) {
			return errors.New("invalid rjs")
		}
		if !(ral >= 0.0 && ral <= 1.0) {
			return errors.New("invalid ral")
		}
		return doCreate(
			ctx.Context, bw, scale,
			nodeFile, viewFile, allocFile,
			ecn, ras, rjs, ral, verbose)
	},
}
