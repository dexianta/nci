package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// JobConfFile matches .refci/conf.yml as a top-level job map.
//
//	my-job:
//	  branch_pattern: main
//	  path_patterns:
//	    - services/**
//	  script: .refci/main.sh
type JobConfFile map[string]JobConfSpec

// JobConfSpec matches one job entry in .refci/conf.yml.
type JobConfSpec struct {
	BranchPattern string   `yaml:"branch_pattern"`
	PathPatterns  []string `yaml:"path_patterns"`
	Script        string   `yaml:"script"`
}

// LoadJobConfs loads job definitions from .refci/conf.yml format.
func LoadJobConfs(path string) ([]JobConf, error) {
	confPath := strings.TrimSpace(path)
	if confPath == "" {
		confPath = filepath.Join(".refci", "conf.yml")
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil, fmt.Errorf("read job conf: %w", err)
	}

	return ParseJobConfs(string(data)), nil
}

func ParseJobConfs(raw string) []JobConf {
	var file JobConfFile
	if err := yaml.Unmarshal([]byte(raw), &file); err != nil {
		return nil
	}
	if len(file) == 0 {
		return nil
	}

	normalized := make(map[string]JobConfSpec, len(file))
	for name, spec := range file {
		key := strings.TrimSpace(name)
		if key == "" {
			continue
		}
		if _, exists := normalized[key]; exists {
			continue
		}
		normalized[key] = spec
	}
	if len(normalized) == 0 {
		return nil
	}

	keys := make([]string, 0, len(normalized))
	for key := range normalized {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]JobConf, 0, len(keys))
	for _, name := range keys {
		spec := normalized[name]
		out = append(out, JobConf{
			Name:          name,
			BranchPattern: spec.BranchPattern,
			PathPatterns:  spec.PathPatterns,
			ScriptPath:    spec.Script,
		})
	}

	return out
}
