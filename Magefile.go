//go:build mage
// +build mage

package main

import (
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var distDir = "./bin"

func Clean() error {
	return sh.Rm(distDir)
}

func buildBinary() error {
	env := map[string]string{
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "amd64",
		"GOOS":        "linux",
	}

	if err := sh.RunWith(env, "go", "build", "-o", filepath.Join(distDir, "grafana-ntfy"), "./pkg"); err != nil {
		return err
	}
	return nil
}

// Runs go mod download and then installs the binary.
func Build() error {
	mg.Deps(Clean)
	mg.Deps(buildBinary)
	return nil
}

func RunLocal() error {
	mg.Deps(Build)
	env := map[string]string{
		"DEBUG": "1",
	}
	return sh.RunWith(env, filepath.Join(distDir, "grafana-ntfy"), "-ntfy-url", "https://ntfy.sh/mytopic")
}
