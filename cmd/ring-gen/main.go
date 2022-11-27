package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		&cli.StringFlag{
			Name:     "node",
			Required: true,
			Usage:    "specify the input node.json",
		},
		&cli.StringFlag{
			Name:     "view",
			Required: true,
			Usage:    "specify the intput view.json",
		},
		&cli.StringFlag{
			Name:     "alloc",
			Required: true,
			Usage:    "specify the output alloc.json",
		},
		&cli.Int64Flag{
			Name:     "bw",
			Required: true,
			Usage:    "specify the total bandwidth [Gbps]",
		},
		&cli.Float64Flag{
			Name:     "ras",
			Required: false,
			Value:    50.0,
			Usage:    "specify the remote access score [20.0-80.0]",
		},
		&cli.Float64Flag{
			Name:     "ral",
			Required: false,
			Value:    0.1,
			Usage:    "specify the remote access limit [0.0-1.0]",
		},
		&cli.Float64Flag{
			Name:     "rjs",
			Required: false,
			Value:    80.0,
			Usage:    "specify the reject score [80.0-100.0]",
		},
		&cli.StringSliceFlag{
			Name:     "modl",
			Required: false,
			Usage:    "specify a mode limit [mix=0.5]",
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
			nodeFile  = ctx.String("node")
			viewFile  = ctx.String("view")
			allocFile = ctx.String("alloc")
			bw        = ctx.Int64("bw")
			ras       = ctx.Float64("ras")
			ral       = ctx.Float64("ral")
			rjs       = ctx.Float64("rjs")
			verbose   = ctx.Bool("vv")
		)
		if bw <= 0 {
			return errors.New("invalid bw")
		}
		if !(ras >= 20.0 && ras <= 80.0) {
			return errors.New("invalid ras")
		}
		if !(ral >= 0.0 && ral <= 1.0) {
			return errors.New("invalid ral")
		}
		if !(rjs >= 80.0 && rjs <= 100.0) {
			return errors.New("invalid rjs")
		}
		for _, s := range ctx.StringSlice("modl") {
			pair := strings.Split(s, "=")
			if len(pair) != 2 {
				return errors.New("invalid modl")
			}
			modl, err := strconv.ParseFloat(pair[1], 64)
			if err != nil {
				return errors.New("invalid modl")
			}
			if !(modl >= 0.0 && modl <= 1.0) {
				return errors.New("invalid modl")
			}
			limitOfMode[pair[0]] = modl
		}
		return doCreate(ctx.Context, nodeFile, viewFile, allocFile, bw, ras, ral, rjs, verbose)
	},
}
