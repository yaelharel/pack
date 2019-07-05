//+build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	packageBase  = "github.com/buildpack/pack"
	goimportscmd = "goimports"
)

var (
	gocmd = mg.GoCmd()
)

var Aliases = map[string]interface{}{
	"format": Tools.Format,
	"test":   Test.All,
}

type Tools mg.Namespace
type Test mg.Namespace
type Verify mg.Namespace

// InstallGoimports installs `goimports`
func (Tools) InstallGoimports() {
	err := sh.RunV(gocmd, split("install -mod=vendor golang.org/x/tools/cmd/goimports")...)
	checkErr(err)
}

// Format checks that format is valid
func (Verify) Format() {
	mg.Deps(Tools.InstallGoimports)

	files, err := listProjectGoFiles()
	checkErr(err)

	output, err := sh.Output(goimportscmd, append(split("-l -local "+packageBase), files...)...)
	checkErr(err)

	if output != "" {
		output, err = sh.Output(goimportscmd, append(split("-d -local "+packageBase), files...)...)
		checkErr(err)

		fmt.Println("ERROR: The following formatting issues were found:")
		fmt.Println()
		fmt.Println(output)
	}
}

// Format formats code files
func (Tools) Format() {
	mg.Deps(Tools.InstallGoimports)

	files, err := listProjectGoFiles()
	checkErr(err)

	err = sh.RunV(goimportscmd, append(split("-l -w -local "+packageBase), files...)...)
	checkErr(err)
}

// All runs all tests
func (Test) All() {
	mg.SerialDeps(Test.Unit, Test.Acceptance)
}

// Unit runs unit (and integration) tests
func (Test) Unit() {
	err := sh.RunV(gocmd, split("test -mod=vendor -v -count=1 -parallel=1 -timeout=0 ./...")...)
	checkErr(err)
}

// Acceptance runs acceptance tests
func (Test) Acceptance() {
	err := sh.RunV(gocmd, split("test -mod=vendor -v -count=1 -parallel=1 -timeout=0 -tags=acceptance ./acceptance")...)
	checkErr(err)
}

func listProjectGoFiles() ([]string, error) {
	var files []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if !strings.HasPrefix(path, "vendor/") && filepath.Ext(info.Name()) == ".go" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func split(args string) []string {
	return strings.Split(args, " ")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
