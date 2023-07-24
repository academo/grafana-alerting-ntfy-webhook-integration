//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/Masterminds/semver/v3"
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
		//get from env default to amd64
		"GOARCH": getOrDefault("GOARCH", "amd64"),
		"GOOS":   getOrDefault("GOOS", "linux"),
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

func Deploy() error {
	mg.Deps(Build)
	var err error

	// fail if git is not clean
	if err = sh.Run("git", "diff-index", "--quiet", "HEAD", "--"); err != nil {
		return fmt.Errorf("git is not clean: %w", err)
	}

	version, err := getNextVersion()
	if err != nil {
		return err
	}
	// create the tag
	if err = sh.Run("git", "tag", "-a", version, "-m", "Release "+version); err != nil {
		return err
	}
	// push the tag
	if err = sh.Run("git", "push", "origin", version); err != nil {
		return err
	}
	fmt.Println("Tag pushed to origin")
	// create the release
	err = sh.Run("gh", "release", "create", version, "--latest", "-t", "Release "+version, "--verify-tag", "--generate-notes")
	if err != nil {
		return err
	}
	// build docker image
	err = sh.Run("docker", "build", "-t", "academo/grafana-ntfy:"+version, ".")
	if err != nil {
		return err
	}
	err = sh.Run("docker", "tag", "academo/grafana-ntfy:"+version, "academo/grafana-ntfy:latest")
	if err != nil {
		return err
	}
	// push docker image
	err = sh.Run("docker", "push", "academo/grafana-ntfy:"+version)
	if err != nil {
		return err
	}
	err = sh.Run("docker", "push", "academo/grafana-ntfy:latest")
	if err != nil {
		return err
	}
	return nil
}

func getNextVersion() (string, error) {
	//ask if patch, minor or major default to minor
	releaseTypes := []string{"patch", "minor", "major"}
	typeRelease := &survey.Select{
		Message: "What kind of release is this?",
		Options: releaseTypes,
		Default: releaseTypes[1],
	}
	var releaseType string
	err := survey.AskOne(typeRelease, &releaseType)
	if err != nil {
		return "", err
	}

	// get the current version
	version, err := sh.Output("git", "describe", "--tags", "--abbrev=0")
	if err != nil {
		return "", fmt.Errorf("failed to get current version: %w", err)
	}
	// create the new version
	currentVersion, err := semver.NewVersion(version)
	var nextVersion semver.Version
	if err != nil {
		return "", fmt.Errorf("failed to parse current version: %w", err)
	}
	switch releaseType {
	case "patch":
		nextVersion = currentVersion.IncPatch()
	case "minor":
		nextVersion = currentVersion.IncMinor()
	case "major":
		nextVersion = currentVersion.IncMajor()
	}
	return nextVersion.String(), nil
}

func getOrDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}
	return value
}
