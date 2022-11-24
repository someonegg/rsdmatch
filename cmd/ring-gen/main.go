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
			Usage:    "specify the total bandwidth (Gbps)",
		},
		&cli.Float64Flag{
			Name:     "ras",
			Required: false,
			Value:    50.0,
			Usage:    "specify the remote access score (40.0-80.0)",
		},
		&cli.Float64Flag{
			Name:     "ral",
			Required: false,
			Value:    0.1,
			Usage:    "specify the remote access limit (0.0-1.0)",
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
		)
		if bw <= 0 {
			return errors.New("invalid bw")
		}
		if !(ras >= 40.0 && ras <= 80.0) {
			return errors.New("invalid ras")
		}
		if !(ral >= 0.0 && ral <= 1.0) {
			return errors.New("invalid ral")
		}
		return doCreate(ctx.Context, nodeFile, viewFile, allocFile, bw, ras, ral)
	},
}
