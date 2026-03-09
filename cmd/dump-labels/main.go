package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/G-Research/dagger/engine"
	"github.com/G-Research/dagger/engine/client/pathutil"
	enginetel "github.com/G-Research/dagger/engine/telemetry"
)

func main() {
	workdir, err := normalizeWorkdir(".")
	if err != nil {
		panic(err)
	}

	labels := enginetel.LoadDefaultLabels(workdir, engine.Version)
	for _, k := range slices.Sorted(maps.Keys(labels.AsMap())) {
		fmt.Println(k + "=" + labels.AsMap()[k])
	}
}

func normalizeWorkdir(workdir string) (string, error) {
	if workdir == "" {
		workdir = os.Getenv("DAGGER_WORKDIR")
	}

	if workdir == "" {
		var err error
		workdir, err = pathutil.Getwd()
		if err != nil {
			return "", err
		}
	}
	workdir, err := pathutil.Abs(workdir)
	if err != nil {
		return "", err
	}

	return workdir, nil
}
