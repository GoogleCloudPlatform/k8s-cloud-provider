/*
Copyright 2023 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// `checkcov` takes as input the output of "go test -cover ..." and a checkcov
// YAML file containing expected coverage limits.
//
// Usage
//
//	$ go test -cover ./pkg/...  > cov.out
//	$ go run tools/checkcov.go -configFile checkcov.yaml -covFile cov.out

var (
	coverageOutputFile  = flag.String("covFile", "", "Output of go test -cover")
	configFile          = flag.String("configFile", "", "Configuration file")
	defaultCovThreshold = flag.Int("defaultCovThreshold", 80, "Default coverage percent.")
	packagePrefix       = flag.String("packagePrefix", "", "Prefix of the go package")
)

const (
	// maxRecommendedCov is the maximum recommended threshold.
	maxRecommendedCov = 80
	// suggestMargin is the margin above which we recommend adjusting coverage
	// thresholds.
	suggestMargin = 10
)

// configFileData is the root of the YAML file format.
type configFileData struct {
	// package path => item
	Entries map[string]*covEntry
}

type covEntry struct {
	// Expected should be an integer from 0 - 100%. If omittted, assumed to be 80%.
	Expected int
}

var covLine = regexp.MustCompile(`.*[ \t]+coverage:[ \t]+(?P<cov>[.0-9]+)%.*`)

type coverageResult map[string]float64

func loadCovOrDie() coverageResult {
	f, err := os.Open(*coverageOutputFile)
	if err != nil {
		fmt.Printf("Error opening coverageOutputFile %q: %v", *coverageOutputFile, err)
		os.Exit(2)
	}
	in, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("Error reading coverageOutputFile %q: %v", *coverageOutputFile, err)
		os.Exit(2)
	}

	ret := coverageResult{}
	lines := strings.Split(string(in), "\n")

	for lineIndex, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		packageName := fields[1]

		switch fields[0] {
		case "?":
			ret[packageName] = 0.0
			continue
		case "ok":
			// fallthrough to process the "ok" lines.
		default:
			continue
		}

		index := covLine.SubexpIndex("cov")
		matches := covLine.FindStringSubmatch(line)
		if matches == nil {
			fmt.Printf("Error: could not parse line %d: (text was %q)\n", lineIndex+1, line)
			os.Exit(2)
		}
		cov, err := strconv.ParseFloat(matches[index], 32)
		if err != nil {
			fmt.Printf("Error: could not parse line %d: %v (text was %q)\n", lineIndex+1, err, line)
			os.Exit(2)
		}
		ret[packageName] = cov
	}

	return ret
}

func loadConfigOrDie() *configFileData {
	f, err := os.Open(*configFile)
	if err != nil {
		fmt.Printf("Error opening configFile %q: %v\n", *configFile, err)
		os.Exit(2)
	}
	all, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("Error reading configFile %q: %v\n", *configFile, err)
		os.Exit(2)
	}

	var ret configFileData
	err = yaml.Unmarshal(all, &ret)
	if err != nil {
		fmt.Printf("Error parsing configFile %q: %v\n", *configFile, err)
		os.Exit(2)
	}

	return &ret
}

func checkOrDie(cov coverageResult, cfg *configFileData) {
	var hasErr bool

	for pkg, value := range cov {
		shortName := strings.TrimPrefix(pkg, *packagePrefix)
		entry, ok := cfg.Entries[shortName]
		if !ok {
			// If an entry does not exist, assume default threshold.
			entry = &covEntry{Expected: *defaultCovThreshold}
		}
		if entry == nil {
			// Package with no configuration is ignored.
			continue
		}
		if value < float64(entry.Expected) {
			fmt.Printf("Error: %s does not have enough coverage (%0.1f < %d)\n", pkg, value, entry.Expected)
			hasErr = true
		}
		// Print a recommendation if coverage is significantly above expected.
		if entry.Expected < maxRecommendedCov && value > float64(entry.Expected)+suggestMargin {
			fmt.Printf("Info : Consider increasing the coverage for %s (%0.1f > %d + 10%%)\n", pkg, value, entry.Expected)
		}
	}

	if hasErr {
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	m := loadCovOrDie()
	cfg := loadConfigOrDie()

	checkOrDie(m, cfg)
}
