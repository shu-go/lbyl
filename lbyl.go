package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	async := flag.Bool("async", false, "launch command2 asynchronously (default false)")
	pipe := flag.Bool("pipe", false, "pipe to command2 (default false)")
	throttle := flag.Int("throttle", 0, "set `interval (ms)` between launchings of command2. Input is buffered (default 0 (no throttling))")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage of %v:
  command1 | %v [OPTIONS] command2 -opt1 -opt2 ?
  command1 | %v -pipe [OPTIONS] command2 -opt1 -opt2

Launch command with stdin (pipe) line by line.

Options:
`,
			os.Args[0], os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	//fmt.Println("--------------------------------------------------")
	//fmt.Printf("Args: %#v\n", os.Args)
	//fmt.Println("--------------------------------------------------")

	var buf [][]byte = nil
	var prevts time.Time = time.Now()

	var command string
	var args []string
	if flag.NArg() > 0 {
		command = flag.Arg(0)
		args = flag.Args()[1:]
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var err error
		line := scanner.Bytes()

		buf = append(buf, line)

		if *throttle != 0 {
			var currts time.Time = time.Now()
			if len(buf) != 0 && currts.Sub(prevts) > time.Duration(*throttle)*time.Millisecond {
				err = launchCommand(*pipe, *async, buf, command, args...)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
				}

				prevts = currts
				buf = buf[:0]
			}
		} else {
			err = launchCommand(*pipe, *async, buf, command, args...)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
			}
			buf = buf[:0]
		}
	}

	if *throttle != 0 && len(buf) != 0 {
		err := launchCommand(*pipe, *async, buf, command, args...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error while launching: %v\n", err)
		}
	}

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
