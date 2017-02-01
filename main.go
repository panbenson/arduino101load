package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/facchinm/go-serial"
	"github.com/kardianos/osext"
	"github.com/mattn/go-shellwords"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	_ "path/filepath"
	_ "runtime"
	"strings"
	"time"
)

var verbose bool

func PrintlnVerbose(a ...interface{}) {
	if verbose {
		fmt.Println(a...)
	}
}

func touch_port_1200bps(portname string, WaitForUploadPort bool) (string, error) {
	initialPortName := portname
	log.Println("Restarting in bootloader mode")

	before_reset_ports, _ := serial.GetPortsList()
	log.Println(before_reset_ports)

	var ports []string

	mode := &serial.Mode{
		BaudRate: 1200,
		Vmin:     0,
		Vtimeout: 1,
	}
	port, err := serial.OpenPort(portname, mode)
	if err != nil {
		log.Println(err)
		return "", err
	}
	err = port.SetDTR(false)
	if err != nil {
		log.Println(err)
	}
	port.Close()

	timeout := false
	go func() {
		time.Sleep(10 * time.Second)
		timeout = true
	}()

	// wait for port to disappear
	if WaitForUploadPort {
		for {
			ports, _ = serial.GetPortsList()
			log.Println(ports)
			portname = findNewPortName(ports, before_reset_ports)
			if portname != "" {
				break
			}
			if timeout {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	// wait for port to reappear
	if WaitForUploadPort {
		after_reset_ports, _ := serial.GetPortsList()
		log.Println(after_reset_ports)
		for {
			ports, _ = serial.GetPortsList()
			log.Println(ports)
			portname = findNewPortName(ports, after_reset_ports)
			if portname != "" {
				time.Sleep(time.Millisecond * 500)
				break
			}
			if timeout {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	if portname == "" {
		portname = initialPortName
		err = errors.New("no new port found")
	} else {
		err = nil
	}
	return portname, err
}

func main_load(args []string) {

	// ARG 1: Path to binaries
	// ARG 2: BIN File to download
	// ARG 3: TTY port to use.
	// ARG 4: quiet/verbose
	// path may contain \ need to change all to /

	uploadPort := ""
	sketchName := "MKR3000Test.bin"
	var ok error

	before_reset_ports, _ := serial.GetPortsList()

	for _, port := range before_reset_ports {
		uploadPort, ok = touch_port_1200bps(port, true)
		if ok == nil {
			break
		}
	}

	folderPath, _ := osext.ExecutableFolder()

	uploadPort = strings.Replace(uploadPort, "/dev/", "", -1)

	dfu_download := []string{folderPath + "/bossac", "-i", "-d", "--port=" + uploadPort, "-U", "true", "-i", "-e", "-w", "-v", folderPath + "/" + sketchName, "-R"}
	err, _, out := launchCommandAndWaitForOutput(dfu_download, "", true)

	fmt.Println(out)

	if err == nil {
		fmt.Println("SUCCESS: Sketch will execute in about 5 seconds.")
		os.Exit(0)
	} else {
		fmt.Println("ERROR: Upload failed on " + uploadPort)
		os.Exit(1)
	}
}

func findNewPortName(slice1 []string, slice2 []string) string {
	m := map[string]int{}

	for _, s1Val := range slice1 {
		m[s1Val] = 1
	}
	for _, s2Val := range slice2 {
		m[s2Val] = m[s2Val] + 1
	}

	for mKey, mVal := range m {
		if mVal == 1 {
			return mKey
		}
	}

	return ""
}

func main_debug(args []string) {

	if len(args) < 1 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}

	verbose = true

	type Command struct {
		command    string
		background bool
	}

	var commands []Command

	fullcmdline := strings.Join(args[:], "")
	temp_commands := strings.Split(fullcmdline, ";")
	for _, command := range temp_commands {
		background_commands := strings.Split(command, "&")
		for i, command := range background_commands {
			var cmd Command
			cmd.background = (i < len(background_commands)-1)
			cmd.command = command
			commands = append(commands, cmd)
		}
	}

	var err error

	for _, command := range commands {
		fmt.Println("command: " + command.command)
		cmd, _ := shellwords.Parse(command.command)
		fmt.Println(cmd)
		if command.background == false {
			err, _, _ = launchCommandAndWaitForOutput(cmd, "", true)
		} else {
			err, _ = launchCommandBackground(cmd, "", true)
		}
		if err != nil {
			fmt.Println("ERROR: Command \" " + command.command + " \" failed")
			os.Exit(1)
		}
	}
	os.Exit(0)
}

func main() {
	name := os.Args[0]
	args := os.Args[1:]

	if strings.Contains(name, "load") {
		fmt.Println("Starting download script...")
		main_load(args)
	}

	if strings.Contains(name, "debug") {
		fmt.Println("Starting debug script...")
		main_debug(args)
	}

	fmt.Println("Wrong executable name")
	os.Exit(1)
}

func searchBLEversionInDFU(file string, string_to_search string) bool {
	read, _ := ioutil.ReadFile(file)
	return strings.Contains(string(read), string_to_search)
}

func launchCommandAndWaitForOutput(command []string, stringToSearch string, print_output bool) (error, bool, string) {
	oscmd := exec.Command(command[0], command[1:]...)
	tellCommandNotToSpawnShell(oscmd)
	stdout, _ := oscmd.StdoutPipe()
	stderr, _ := oscmd.StderrPipe()
	multi := io.MultiReader(stderr, stdout)
	err := oscmd.Start()
	in := bufio.NewScanner(multi)
	in.Split(bufio.ScanLines)
	found := false
	out := ""
	for in.Scan() {
		if print_output {
			PrintlnVerbose(in.Text())
		}
		out += in.Text() + "\n"
		if stringToSearch != "" {
			if strings.Contains(in.Text(), stringToSearch) {
				found = true
			}
		}
	}
	err = oscmd.Wait()
	return err, found, out
}

func launchCommandBackground(command []string, stringToSearch string, print_output bool) (error, bool) {
	oscmd := exec.Command(command[0], command[1:]...)
	tellCommandNotToSpawnShell(oscmd)
	err := oscmd.Start()
	return err, false
}
