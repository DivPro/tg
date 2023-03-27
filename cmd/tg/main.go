// Copyright (c) 2020 Khramtsov Aleksei (seniorGolang@gmail.com).
// This file (main.go at 24.06.2020, 15:03) is subject to the terms and
// conditions defined in file 'LICENSE', which is part of this project source code.
package main

import (
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/seniorGolang/tg/v2/pkg/generator"
	"github.com/seniorGolang/tg/v2/pkg/logger"
	"github.com/seniorGolang/tg/v2/pkg/skeleton"
)

var (
	Version    = "v2.3.15"
	BuildStamp = time.Now().String()
)

var log = logger.Log.WithField("module", "tg")

func main() {

	app := cli.NewApp()
	app.Version = Version
	app.EnableBashCompletion = true
	app.Usage = "make Go-Kit API easy"
	app.Name = "golang service 't'ransport 'g'enerator (tg)"
	app.Compiled, _ = time.Parse(time.RFC3339, BuildStamp)

	app.Commands = []*cli.Command{
		{
			Name:   "init",
			Usage:  "init project",
			Action: cmdInit,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "project",
					Usage: "project name",
				},
				&cli.StringFlag{
					Name:  "repo",
					Usage: "base repository",
				},
				&cli.BoolFlag{
					Name:  "trace",
					Usage: "use Jaeger tracer",
				},
				&cli.BoolFlag{
					Name:  "mongo",
					Usage: "enable mongo support",
				},
			},
			ArgsUsage:   "[project name]",
			UsageText:   "tg init someProject",
			Description: "init directory structures, basic configuration package",
		},
		{
			Name:   "azure",
			Usage:  "generate Azure manifests by interfaces in 'service' package",
			Action: cmdAzure,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "services",
					Value: "./pkg/someService/service",
					Usage: "path to services package",
				},
				&cli.StringFlag{
					Name:  "appName",
					Value: "service",
					Usage: "application name",
				},
				&cli.StringFlag{
					Name:  "routePrefix",
					Value: "",
					Usage: "router path prefix name",
				},
				&cli.StringFlag{
					Name:  "logLevel",
					Value: "Debug",
					Usage: "log level name",
				},
				&cli.BoolFlag{
					Name:  "enableHealth",
					Value: false,
					Usage: "enable health check",
				},
				&cli.StringFlag{
					Name:  "outPath",
					Usage: "path to output folder",
				},
			},
			UsageText:   "tg azure",
			Description: "generate Azure manifests layer by interfaces",
		},
		{
			Name:   "transport",
			Usage:  "generate services transport layer by interfaces in 'service' package",
			Action: cmdTransport,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "services",
					Value: "./pkg/someService/service",
					Usage: "path to services package",
				},
				&cli.StringFlag{
					Name:  "out",
					Usage: "path to output folder",
				},
				&cli.StringFlag{
					Name:  "outSwagger",
					Usage: "path to output swagger file",
				},
				&cli.StringFlag{
					Name:  "redoc",
					Usage: "path to output redoc bundle",
				},
				&cli.BoolFlag{
					Name:  "jaeger",
					Usage: "use Jaeger tracer",
				},
				&cli.BoolFlag{
					Name:  "zipkin",
					Usage: "use Zipkin tracer",
				},
				&cli.BoolFlag{
					Name:  "mongo",
					Usage: "enable mongo support",
				},
				&cli.StringFlag{
					Name:  "implements",
					Usage: "path to generate implements",
				},
				&cli.StringFlag{
					Name:  "tests",
					Usage: "path to generate tests",
				},
			},

			UsageText:   "tg transport",
			Description: "generate services transport layer by interfaces",
		},
		{
			Name:   "client",
			Usage:  "generate services clients by interfaces in 'service' package",
			Action: cmdClient,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "services",
					Value: "./pkg/someService/service",
					Usage: "path to services package",
				},
				&cli.StringFlag{
					Name:  "outPath",
					Value: "./pkg/clients",
					Usage: "path to output clients",
				},
				&cli.BoolFlag{
					Name:  "go",
					Value: false,
					Usage: "enable go client with package manifest",
				},
				&cli.BoolFlag{
					Name:  "js",
					Value: false,
					Usage: "enable js client with package manifest",
				},
				&cli.BoolFlag{
					Name:  "ts",
					Value: false,
					Usage: "enable ts client with package manifest",
				},
			},

			UsageText:   "tg client --services ./pkg/someService/service",
			Description: "generate services transport layer by interfaces",
		},
		{
			Name:   "swagger",
			Usage:  "generate swagger documentation by interfaces in 'service' package",
			Action: cmdSwagger,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "services",
					Value: "./pkg/someService/service",
					Usage: "path to services package",
				},
				&cli.StringFlag{
					Name:  "outFile",
					Usage: "path to output folder",
				},
				&cli.StringSliceFlag{
					Name:  "iface",
					Usage: "interfaces included to swagger",
				},
				&cli.StringFlag{
					Name:  "redoc",
					Usage: "path to output redoc bundle",
				},
			},

			UsageText:   "tg swagger --iface firstIface --iface secondIface",
			Description: "generate swagger documentation by interfaces",
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdInit(c *cli.Context) (err error) {

	defer func() {
		if err == nil {
			log.Info("done")
		}
	}()
	repo := c.String("repo")
	project := c.String("project")

	if repo == "" {
		repo = project
	}
	return skeleton.GenerateSkeleton(log, Version, project, repo, "./"+c.Args().First(), c.Bool("trace"), c.Bool("mongo"))
}

func cmdClient(c *cli.Context) (err error) {

	defer func() {
		if err == nil {
			log.Info("done")
		}
	}()
	var tr generator.Transport
	if tr, err = generator.NewTransport(log, Version, c.String("services")); err != nil {
		return
	}
	if c.Bool("go") {
		if err = tr.RenderClient(c.String("outPath")); err != nil {
			return
		}
	}
	if c.Bool("js") {
		if err = tr.RenderClientJS(c.String("outPath")); err != nil {
			return
		}
	}
	if c.Bool("ts") {
		if err = tr.RenderClientTS(c.String("outPath")); err != nil {
			return
		}
	}
	return
}

func cmdTransport(c *cli.Context) (err error) {

	defer func() {
		if err == nil {
			log.Info("done")
		}
	}()
	opts := []generator.Option{
		generator.WithTests(c.String("tests")),
		generator.WithImplements(c.String("implements")),
	}
	var tr generator.Transport
	if tr, err = generator.NewTransport(log, Version, c.String("services"), opts...); err != nil {
		return
	}
	outPath, _ := path.Split(c.String("services"))
	outPath = path.Join(outPath, "transport")
	if c.String("out") != "" {
		outPath = c.String("out")
	}
	if err = tr.RenderServer(outPath); err != nil {
		return
	}
	if c.String("outSwagger") != "" {
		err = tr.RenderSwagger(c.String("outSwagger"))
	}
	if c.String("redoc") != "" {
		var output []byte
		log.Infof("write to %s", c.String("redoc"))
		if output, err = exec.Command("redoc-cli", "bundle", c.String("outSwagger"), "-o", c.String("redoc")).Output(); err != nil {
			log.WithError(err).Error(string(output))
		}
	}
	return
}

func cmdSwagger(c *cli.Context) (err error) {

	defer func() {
		if err == nil {
			log.Info("done")
		}
	}()

	var tr generator.Transport
	if tr, err = generator.NewTransport(log, Version, c.String("services")); err != nil {
		return
	}

	outPath := path.Join(c.String("services"), "swagger.yaml")

	if c.String("outFile") != "" {
		outPath = c.String("outFile")
	}
	if err = tr.RenderSwagger(outPath); err == nil {
		if c.String("redoc") != "" {
			var output []byte
			log.Infof("write to %s", c.String("redoc"))
			if output, err = exec.Command("redoc-cli", "bundle", outPath, "-o", c.String("redoc")).Output(); err != nil {
				log.WithError(err).Error(string(output))
			}
		}
	}
	return
}

func cmdAzure(c *cli.Context) (err error) {

	defer func() {
		if err == nil {
			log.Info("done")
		}
	}()
	var tr generator.Transport
	if tr, err = generator.NewTransport(log, Version, c.String("services")); err != nil {
		return
	}
	outPath := path.Join(c.String("services"), "azure-fApp")
	if c.String("outPath") != "" {
		outPath = c.String("outPath")
	}
	return tr.RenderAzure(c.String("appName"), c.String("routePrefix"), outPath, c.String("logLevel"), c.Bool("enableHealth"))
}
