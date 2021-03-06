package requirements

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gerrit.akraino.org/kni/installer/pkg/utils"
	getter "github.com/hashicorp/go-getter"
)

// Requirement : Structure that contains the settings needed for managing a requirement
type Requirement struct {
	binaryName string
	sourceRepo string
	buildPath  string
}

// New constructor for the generator
func New(binaryName string, sourceRepo string, buildPath string) Requirement {
	r := Requirement{binaryName, sourceRepo, buildPath}
	return r
}

// download requirement from a tarball or folder
func (r Requirement) FetchRequirementFolder() {
	// extract the tarball if exists
	log.Println(fmt.Sprintf("Pulling %s tarball from %s", r.binaryName, r.sourceRepo))

	extractDir := fmt.Sprintf("%s/%s_content", r.buildPath, r.binaryName)
	client := &getter.Client{Src: r.sourceRepo, Dst: extractDir, Mode: getter.ClientModeAny}
	err := client.Get()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error cloning tarball repository: %s", err))
		os.Exit(1)
	}

	// find the binary inside the extracted content
	alternativeBinaryName := path.Base(r.sourceRepo)
	err = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if (info.Name() == r.binaryName || info.Name() == alternativeBinaryName) && !info.IsDir() {
			// we found the binary, move it. Give exec perms as well
			finalBinary := fmt.Sprintf("%s/%s", r.buildPath, r.binaryName)
			os.Rename(path, finalBinary)
			os.Chmod(finalBinary, 0755)
			os.RemoveAll(extractDir)
			return nil
		}
		return nil
	})
}

// generates the openshift binary
func (r Requirement) BuildOpenshiftBinary() {
	extractDir := fmt.Sprintf("%s/src/github.com/openshift/installer", r.buildPath)
	client := &getter.Client{Src: r.sourceRepo, Dst: extractDir, Mode: getter.ClientModeAny}
	err := client.Get()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error cloning tarball repository: %s", err))
		os.Exit(1)
	}

	// build the openshift binary
	envVars := []string{"TAGS=libvirt", fmt.Sprintf("GOPATH=%s", r.buildPath)}
	utils.ExecuteCommand(extractDir, envVars, true, true, "hack/build.sh")

	// copy the generated binary to the build directory
	var cpEnvVars []string
	utils.ExecuteCommand("", cpEnvVars, true, true, "cp", fmt.Sprintf("%s/bin/openshift-install", extractDir), r.buildPath)
	log.Println(fmt.Sprintf("Installer is available on %s/openshift-install", r.buildPath))
}

// download a requirement from a git repo and build it
func (r Requirement) FetchRequirementGit() {
	if r.binaryName == "openshift-install" {
		r.BuildOpenshiftBinary()
	} else {
		log.Fatal(fmt.Sprintf("Build of binary %s is not supported", r.binaryName))
		os.Exit(1)
	}
}

// downloads an individual requirement
func (r Requirement) FetchRequirement() {
	log.Println(fmt.Sprintf("Downloading %s requirement from %s", r.binaryName, r.sourceRepo))

	// first check if the binary already exists
	binaryPath := fmt.Sprintf("%s/%s", r.buildPath, r.binaryName)
	if _, err := os.Stat(binaryPath); err == nil {
		log.Println(fmt.Sprintf("Using existing %s", binaryPath))
	} else if os.IsNotExist(err) {
		if strings.Contains(r.sourceRepo, ".git") {
			r.FetchRequirementGit()
		} else {
			r.FetchRequirementFolder()
		}
	}
}
