//go:build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
)

func main() {
	// Use current working directory - the script should be run from the __testdata directory
	baseDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Generate tar.gz file
	tarGzPath := filepath.Join(baseDir, "s3", "state.tar.gz")
	if err := generateTarGz(tarGzPath); err != nil {
		panic(err)
	}
	// Copy to other directories
	copyFile(tarGzPath, filepath.Join(baseDir, "gcs", "state.tar.gz"))
	copyFile(tarGzPath, filepath.Join(baseDir, "azure", "state.tar.gz"))

	// Generate instances.json
	instancesPath := filepath.Join(baseDir, "s3", "instances.json")
	if err := generateInstancesJSON(instancesPath); err != nil {
		panic(err)
	}
	// Copy to other directories
	copyFile(instancesPath, filepath.Join(baseDir, "gcs", "instances.json"))
	copyFile(instancesPath, filepath.Join(baseDir, "azure", "instances.json"))

	println("Test data generated successfully")
}

func generateTarGz(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add test files to the archive
	files := map[string]string{
		"test-instance.json": `{"instanceId":"test-inst-001","instanceName":"Test Instance","status":"deployed"}`,
		"subdir/nested.json": `{"nested":true}`,
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func generateInstancesJSON(path string) error {
	instances := []state.InstanceState{
		{
			InstanceID:            "integration-test-001",
			InstanceName:          "Integration Test Instance 1",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067200,
		},
		{
			InstanceID:            "integration-test-002",
			InstanceName:          "Integration Test Instance 2",
			Status:                core.InstanceStatusDeployed,
			LastDeployedTimestamp: 1704067300,
		},
	}

	data, err := json.MarshalIndent(instances, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
