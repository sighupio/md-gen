package main

import (
	"os"

	"github.com/sighupio/md-gen/internal/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	os.Exit(exec())
}

func exec() int {
	log := &logrus.Logger{
		Out: os.Stdout,
		Formatter: &logrus.TextFormatter{
			ForceColors:      true,
			DisableTimestamp: true,
		},
		Level: logrus.DebugLevel,
	}

	if _, err := cmd.NewRootCmd().ExecuteC(); err != nil {
		log.Error(err)

		return 1
	}

	return 0
}
