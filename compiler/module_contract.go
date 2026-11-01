package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const moduleContractFile = "zz_nestgo_modules.json"

type moduleContract struct {
	Schema  int                     `json:"schema"`
	Modules map[string]modulePublic `json:"modules"`
}

type modulePublic struct {
	Providers []string `json:"providers"`
	Exports   []string `json:"exports"`
}

func exportModuleContract(files []*File) ([]byte, error) {
	result := moduleContract{Schema: 1, Modules: make(map[string]modulePublic)}
	for _, file := range files {
		for _, declaration := range file.Declarations {
			if declaration.Kind != "type" {
				continue
			}
			for _, annotation := range declaration.Annotations {
				if annotation.Name != "Module" {
					continue
				}
				options, err := parseModuleOptions(file.Path, annotation, declaration.Name)
				if err != nil {
					return nil, err
				}
				result.Modules[declaration.Name] = modulePublic{
					Providers: append([]string{}, options.Providers...),
					Exports:   append([]string{}, options.Exports...),
				}
			}
		}
	}
	return json.MarshalIndent(result, "", "  ")
}

func readModuleContract(dir string) (moduleContract, error) {
	var result moduleContract
	raw, err := os.ReadFile(filepath.Join(dir, moduleContractFile))
	if err != nil {
		return result, fmt.Errorf("module export contract unavailable: %w", err)
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("invalid module export contract: %w", err)
	}
	if result.Schema != 1 || result.Modules == nil {
		return result, fmt.Errorf("unsupported module export contract schema %d", result.Schema)
	}
	return result, nil
}
