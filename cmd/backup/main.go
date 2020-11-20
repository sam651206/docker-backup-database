package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"backup/pkg/config"
	"backup/pkg/dbdump"
	"backup/pkg/storage"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// Version set at compile-time
var (
	Version string
)

func main() {
	// Load env-file if it exists first
	if filename, found := os.LookupEnv("PLUGIN_ENV_FILE"); found {
		_ = godotenv.Load(filename)
	}

	if _, err := os.Stat("/run/drone/env"); err == nil {
		godotenv.Overload("/run/drone/env")
	}

	cfg := &config.Config{}
	app := &cli.App{
		Name:      "docker-backup-datavase",
		Usage:     "Docker image to periodically backup a your database",
		Copyright: "Copyright (c) " + strconv.Itoa(time.Now().Year()) + " Bo-Yi Wu",
		Version:   Version,
		Flags:     settingsFlags(cfg),
		Action:    run(cfg),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(cfg *config.Config) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// initial storage interface
		s3, err := storage.NewEngine(*cfg)
		if err != nil {
			return err
		}

		// initial database dump interface
		backup, err := dbdump.NewEngine(*cfg)
		if err != nil {
			return err
		}

		// check bucket exist
		if exist, err := s3.BucketExists(cfg.Storage.Bucket); !exist {
			if err != nil {
				return errors.New("bucket not exist or you don't have permission: " + err.Error())
			}
			return errors.New("bucket not exist or you don't have permission")
		}

		if err := backup.Exec(); err != nil {
			return err
		}

		// upload file to s3
		content, err := ioutil.ReadFile(cfg.Storage.DumpName)
		if err != nil {
			return errors.New("can't open the gzip file: " + err.Error())
		}

		// backup database
		return s3.UploadFile(cfg.Storage.Bucket, time.Now().Format("20060102150405")+".sql.gz", content, nil)
	}
}
