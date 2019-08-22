package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/shu-go/gli"
)

type globalCmd struct {
	Async    bool `help:"launch command2 asynchronously (default false)"`
	Pipe     bool `help:"pipe to command2 (default false)"`
	Throttle int  `help:"set interval (ms) between launchings of command2. Input is buffered (default 0 (no throttling))"`
}

func main() {
	app := gli.NewWith(&globalCmd{})
	app.Name = "lbyl"
	app.Desc = "Launch command with stdin (pipe) line by line."
	app.Version = "0.0.0"
	app.Usage = `
command1 | lbyl [OPTIONS] command2 -opt1 -opt2 ?
command1 | lbyl -pipe [OPTIONS] command2 -opt1 -opt2 
`
	app.Copyright = "(C) 2018 Shuhei Kubota"

	app.Run(os.Args)
}

func (c globalCmd) Run(args []string) error {
	//fmt.Println("--------------------------------------------------")
	//fmt.Printf("Args: %#v\n", os.Args)
	//fmt.Println("--------------------------------------------------")

	var buf [][]byte = nil
	var prevts time.Time = time.Now()

	var command string
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var err error
		line := scanner.Bytes()

		buf = append(buf, line)

		if c.Throttle != 0 {
			var currts time.Time = time.Now()
			if len(buf) != 0 && currts.Sub(prevts) > time.Duration(c.Throttle)*time.Millisecond {
				err = launchCommand(c.Pipe, c.Async, buf, command, args...)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
				}

				prevts = currts
				buf = buf[:0]
			}
		} else {
			err = launchCommand(c.Pipe, c.Async, buf, command, args...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
			}
			buf = buf[:0]
		}
	}

	if c.Throttle != 0 && len(buf) != 0 {
		err := launchCommand(c.Pipe, c.Async, buf, command, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
		}
	}

	return nil
}

func launchCommand(pipe, async bool, buf [][]byte, command string, args ...string) error {
	if len(command) == 0 {
		return nil
	}

	if !pipe {
		sbuf := make([]string, 0, len(buf))
		for _, b := range buf {
			sbuf = append(sbuf, string(b))
		}
		var s string = strings.Join(sbuf, "\n")
		args = replaceEach("?", s, args...)
	}

	c := exec.Command(command, args...)
	//fmt.Fprintln(os.Stderr, "--------------------------------------------------")
	//fmt.Fprintf(os.Stderr, "Command: %v %#v\n", command, args)

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if pipe {
		sp, err := c.StdinPipe()
		if err != nil {
			return fmt.Errorf("can't get StdinPipe(): %v", err)
		}
		for _, b := range buf {
			sp.Write(b)
			sp.Write([]byte("\n"))
			//fmt.Fprintf(os.Stderr, "Pipe: %#v\n", string(b))
		}
		sp.Close()
	}
	//fmt.Fprintln(os.Stderr, "--------------------------------------------------")

	var err error
	if async {
		err = c.Start()
	} else {
		err = c.Run()
	}

	if err != nil {
		return fmt.Errorf("failed launching (command:%v, args:%#v): %v", command, args, err)
	}

	return nil
}

func replaceEach(old, new string, strs ...string) []string {
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		result = append(result, strings.Replace(s, old, new, -1))
	}
	return result
}
