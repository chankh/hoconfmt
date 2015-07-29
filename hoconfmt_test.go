package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update .golden files")

func runTest(t *testing.T, in, out string) {
	var buf bytes.Buffer
	err := processFile(in, nil, &buf)
	if err != nil {
		t.Error(err)
		return
	}

	expected, err := ioutil.ReadFile(out)
	if err != nil {
		t.Error(err)
		return
	}

	if got := buf.Bytes(); !bytes.Equal(got, expected) {
		if *update {
			if in != out {
				if err := ioutil.WriteFile(out, got, 0666); err != nil {
					t.Error(err)
				}
				return
			}
			// in == out: don't accidentally destroy input
			t.Errorf("WARNING: -update did not rewrite input file %s", in)
		}

		t.Errorf("(hoconfmt %s) != %s (see %s.hoconfmt)", in, out, in)
		d, err := diff(expected, got)
		if err == nil {
			t.Errorf("%s", d)
		}
		if err := ioutil.WriteFile(in+".hoconfmt", got, 0666); err != nil {
			t.Error(err)
		}
	}
}

// TestRewrite processes testdata/*.input files and compares them to the
// corresponding testdata/*.golden files. The hoconfmt flags used to process
// a file must be provided via a comment of the form
//
//     //hoconfmt flags
// in the processed file within the first 20 lines, if any.
func TestRewrite(t *testing.T) {
	// determine input files
	match, err := filepath.Glob("testdata/*.input")
	if err != nil {
		t.Fatal(err)
	}

	for _, in := range match {
		out := in // for files where input and output are identical
		if strings.HasSuffix(in, ".input") {
			out = in[:len(in)-len(".input")] + ".golden"
		}
		runTest(t, in, out)
		if in != out {
			// Check idempotence
			runTest(t, out, out)
		}
	}
}

func TestCRLF(t *testing.T) {
	const input = "testdata/crlf.input"   // must contain CR/LF's
	const golden = "testdata/crlf.golden" // must not contain any CR's

	data, err := ioutil.ReadFile(input)
	if err != nil {
		t.Error(err)
	}
	if bytes.Index(data, []byte("\r\n")) < 0 {
		t.Errorf("%s contains no CR/LF's", input)
	}

	data, err = ioutil.ReadFile(golden)
	if err != nil {
		t.Error(err)
	}
	if bytes.Index(data, []byte("\r")) > 0 {
		t.Errorf("%s contains CR's", golden)
	}
}
