package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kardianos/osext"
)

var verbose bool

func PrintlnVerbose(a ...interface{}) {
	if verbose {
		fmt.Println(a...)
	}
}

func main() {
	fmt.Println("Starting download script...")

	args := os.Args[1:]

	bin_path, err := osext.ExecutableFolder()
	adb := bin_path + "/adb"
	adb = filepath.ToSlash(adb)

	serialnumber := ""
	verbosity := "verbose"

	bin_file_name := args[0]
	if len(args) > 1 {
		serialnumber = args[1]
	}
	if len(args) > 2 {
		verbosity = args[2]
	}

	if verbosity == "quiet" {
		verbose = false
	} else {
		verbose = true
	}

	PrintlnVerbose("Args to shell:", args)
	PrintlnVerbose("Serial Number: " + serialnumber)
	PrintlnVerbose("BIN FILE " + bin_file_name)

	if runtime.GOOS == "darwin" {
		library_path := os.Getenv("DYLD_LIBRARY_PATH")
		if !strings.Contains(library_path, bin_path) {
			os.Setenv("DYLD_LIBRARY_PATH", bin_path+":"+library_path)
		}
	}

	adb_search_command := []string{adb, "devices"}

	err, found, _ := launchCommandAndWaitForOutput(adb_search_command, serialnumber, false)

	if err == nil && found == false {
		err, found, _ = launchCommandAndWaitForOutput(adb_search_command, strings.ToUpper(serialnumber), false)
		if found == true {
			serialnumber = strings.ToUpper(serialnumber)
		}
	}

	if err == nil && found == false {
		err, found, _ = launchCommandAndWaitForOutput(adb_search_command, strings.ToLower(serialnumber), false)
		if found == true {
			serialnumber = strings.ToLower(serialnumber)
		}
	}

	if err != nil {
		fmt.Println("ERROR: Target board not found")
		os.Exit(1)
	}

	var serialnumberslice []string

	if found == true {
		serialnumberslice = []string{"-s", serialnumber}
	}

	adb_test := []string{adb}
	adb_test = append(adb_test, serialnumberslice...)
	adb_test = append(adb_test, "shell", "ps", "x")
	err, running, _ := launchCommandAndWaitForOutput(adb_test, "arduino-connector", false)

	targetFolder := ""

	if running {
		/*
			paths := strings.Split(match, "-config")
			baseFolder := filepath.Dir(paths[0])
			if _, err := os.Stat(baseFolder); err == nil {
				targetFolder = baseFolder + "/sketches/"
			}
		*/
		// this path must exist (the connector creates it)
		targetFolder = "/tmp/sketches/"
	} else {
		fmt.Println("Arduino Connector not running on the target board")
		os.Exit(1)
	}

	adb_push := []string{adb}
	adb_push = append(adb_push, serialnumberslice...)
	adb_push = append(adb_push, "push", bin_file_name, targetFolder+filepath.Base(bin_file_name))
	err, _, _ = launchCommandAndWaitForOutput(adb_push, "", true)

	adb_chmod := []string{adb}
	adb_chmod = append(adb_chmod, serialnumberslice...)
	adb_chmod = append(adb_chmod, "shell", "chmod", "+x", targetFolder+filepath.Base(bin_file_name))
	err, _, _ = launchCommandAndWaitForOutput(adb_chmod, "", true)

	if err == nil {
		fmt.Println("SUCCESS!")
		os.Exit(0)
	} else {
		fmt.Println("ERROR: Upload failed")
		os.Exit(1)
	}
}

func launchCommandBackground(command []string, stringToSearch string, print_output bool) (error, bool) {
	oscmd := exec.Command(command[0], command[1:]...)
	tellCommandNotToSpawnShell(oscmd)
	err := oscmd.Start()
	return err, false
}

func launchCommandAndWaitForOutput(command []string, stringToSearch string, print_output bool) (error, bool, string) {
	oscmd := exec.Command(command[0], command[1:]...)
	tellCommandNotToSpawnShell(oscmd)
	stdout, _ := oscmd.StdoutPipe()
	stderr, _ := oscmd.StderrPipe()
	multi := io.MultiReader(stderr, stdout)
	err := oscmd.Start()
	in := bufio.NewScanner(multi)
	matchingLine := ""
	in.Split(bufio.ScanLines)
	found := false
	for in.Scan() {
		if print_output {
			PrintlnVerbose(in.Text())
		}
		if stringToSearch != "" {
			if strings.Contains(in.Text(), stringToSearch) {
				matchingLine = in.Text()
				found = true
			}
		}
	}
	err = oscmd.Wait()
	return err, found, matchingLine
}
