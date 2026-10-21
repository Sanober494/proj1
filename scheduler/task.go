package scheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	effectsPath = "../data/effects.txt"
	inDir       = "../data/in"
	outDir      = "../data/out"
)

// ImageTask holds everything needed to process one image.
type ImageTask struct {
	InPath  string   // Full path of the image to read
	OutPath string   // Full path of the image to write
	Effects []string // Effects to apply, in order
}

// effectsEntry mirrors one JSON string of effects.txt.
type effectsEntry struct {
	InPath  string   `json:"inPath"`
	OutPath string   `json:"outPath"`
	Effects []string `json:"effects"`
}

// loadTasks reads effects.txt once for every data directory listed in
// config.DataDirs (separated by '+') and returns the resulting tasks.
// The directory identifier is prepended to each output file name.
func loadTasks(config Config) []ImageTask {
	var tasks []ImageTask
	for _, dir := range strings.Split(config.DataDirs, "+") {
		tasks = append(tasks, loadTasksForDir(dir)...)
	}
	return tasks
}

func loadTasksForDir(dir string) []ImageTask {
	file, err := os.Open(effectsPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var tasks []ImageTask
	reader := json.NewDecoder(file)
	for {
		var entry effectsEntry
		err := reader.Decode(&entry)
		if err == io.EOF {
			return tasks
		}
		if err != nil {
			panic(err)
		}
		tasks = append(tasks, ImageTask{
			InPath:  fmt.Sprintf("%s/%s/%s", inDir, dir, entry.InPath),
			OutPath: fmt.Sprintf("%s/%s_%s", outDir, dir, entry.OutPath),
			Effects: entry.Effects,
		})
	}
}
