package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Artifact struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	InternalType int    `json:"internal_type"`
	Type         string `json:"type"`
	Goos         string `json:"goos,omitempty"`
	Goarch       string `json:"goarch,omitempty"`
	Goamd64      string `json:"goamd64,omitempty"`
	Target       string `json:"target,omitempty"`
	Extra        struct {
		Binary    string      `json:"Binary,omitempty"`
		Builder   string      `json:"Builder,omitempty"`
		Ext       string      `json:"Ext,omitempty"`
		ID        string      `json:"ID,omitempty"`
		Binaries  []string    `json:"Binaries,omitempty"`
		Checksum  string      `json:"Checksum,omitempty"`
		Format    string      `json:"Format,omitempty"`
		Replaces  interface{} `json:"Replaces"`
		WrappedIn string      `json:"WrappedIn,omitempty"`
	} `json:"extra,omitempty"`
	Go386   string `json:"go386,omitempty"`
	Goarm   string `json:"goarm,omitempty"`
	Goarm64 string `json:"goarm64,omitempty"`
}

func parseArtifacts(filePath string) ([]Artifact, error) {
	// Open the JSON file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file's content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Unmarshal the JSON data into the XXX struct
	var result []Artifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	// Return the unmarshaled struct
	return result, nil
}

func moveArtifacts(artifacts []Artifact) error {
	for _, artifact := range artifacts {
		if artifact.Type == "Metadata" || artifact.Type == "Checksum" || artifact.Type == "Archive" {
			continue
		}
		outDir := filepath.Join("output", artifact.Extra.ID)
		err := MkdirAll(outDir)
		if err != nil {
			return fmt.Errorf("error creating dir: %v", err)
		}
		outFile := fmt.Sprintf("%s_%s-%s%s", artifact.Extra.Binary, artifact.Goos, artifact.Goarch, artifact.Extra.Ext)
		outPath := filepath.Join(outDir, outFile)
		err = copyFile(artifact.Path, outPath)
		if err != nil {
			return fmt.Errorf("error copying files: %v", err)
		}
	}
	return nil
}
