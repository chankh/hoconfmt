package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/printer"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

var (
	// main operation modes
	list      = flag.Bool("l", false, "list files whose formatting differs from hoconfmt's")
	write     = flag.Bool("w", false, "write result to (source) file instead of stdout")
	doDiff    = flag.Bool("d", false, "display diffs instead of writing files")
	allErrors = flag.Bool("e", false, "report all errors (not just the first 10 on different lines)")

	// debugging
	cpuProfile = flag.String("cpuprofile", "", "write cpu profile to this file")
)

const (
	tabWidth    = 4
	printerMode = printer.UseSpaces
)

var (
	fileSet  = token.NewFileSet() // per process FileSet
	exitCode = 0
)

func report(err error) {
	scanner.PrintError(os.Stderr, err)
	exitCode = 2
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: hoconfmt [flags] [path...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func isConfFile(f os.FileInfo) bool {
	// ignore non .conf files
	name := f.Name()
	return !f.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".conf")
}

func processFile(filename string, in io.Reader, out io.Writer) error {
	if in == nil {
		f, err := os.Open(filename)
		if err != nil {
			return nil
		}
		defer f.Close()
		in = f
	}

	src, err := ioutil.ReadAll(in)
	if err != nil {
		return err
	}

	res, err := format(src, printer.Config{Mode: printerMode, Tabwidth: tabWidth})
	if err != nil {
		return nil
	}

	if !bytes.Equal(src, res) {
		// formatting has changed
		if *list {
			fmt.Fprintln(out, filename)
		}
		if *write {
			err = ioutil.WriteFile(filename, res, 0644)
			if err != nil {
				return err
			}
		}
		if *doDiff {
			data, err := diff(src, res)
			if err != nil {
				return fmt.Errorf("computing diff: %s", err)
			}
			fmt.Printf("diff %s hoconfmt/%s\n", filename, filename)
			out.Write(data)
		}
	}

	if !*list && !*write && !*doDiff {
		_, err = out.Write(res)
	}
	return err
}

func main() {
	// call hoconfmtMain in a separate function
	// so that it can use defer and have them
	// run before the exit.
	hoconfmtMain()
	os.Exit(exitCode)
}

func hoconfmtMain() {
	flag.Usage = usage
	flag.Parse()
}

func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "hoconfmt")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "hoconfmt")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}

func format(src []byte, cfg printer.Config) ([]byte, error) {
	// Determine and prepend leading space.
	i, j := 0, 0
	for j < len(src) && isSpace(src[j]) {
		if src[j] == '\n' {
			i = j + 1 // byte offset of last line in leading space
		}
		j++
	}
	var res []byte
	res = append(res, src[:i]...)

	// Determine and prepend indentation of first code line.
	// Spaces are ignored unless there are no tabs,
	// in which case spaces count as one tab.
	indent := 0
	hasSpace := false
	for _, b := range src[i:j] {
		switch b {
		case ' ':
			hasSpace = true
		case '\t':
			indent++
		}
	}
	if indent == 0 && hasSpace {
		indent = 1
	}
	for i := 0; i < indent; i++ {
		res = append(res, '\t')
	}

	return append(res, src[i:]...), nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
