// Command okf works with Open Knowledge Format (OKF) v0.2 bundles.
//
//	okf check [-strict] <bundle>
//	okf viz [-o file] [-name name] [-layers a,b,c] [-check] <bundle>
//
// check reports conformance errors and guidance warnings, and exits 1 on an error, or on a warning
// with -strict. viz writes the bundle's graph page, by default to viz.html in the bundle; with
// -check it writes nothing and exits 1 when the file differs from a fresh render.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/katalystng/okf-go/bundle"
	"github.com/katalystng/okf-go/check"
	"github.com/katalystng/okf-go/viz"
)

const usage = `usage:
  okf check [-strict] <bundle>
  okf viz [-o file] [-name name] [-layers a,b,c] [-check] <bundle>
`

// errFailed marks a run that reported its own failure and needs no further message.
var errFailed = errors.New("failed")

func main() {
	err := run(os.Args[1:], os.Stdout, os.Stderr)
	switch {
	case err == nil:
	case errors.Is(err, errFailed):
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "okf:", err)
		os.Exit(2)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return errors.New("name a command")
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout)
	case "viz":
		return runViz(args[1:], stdout)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return nil
	}
	fmt.Fprint(stderr, usage)
	return fmt.Errorf("unknown command %q", args[0])
}

func runCheck(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "fail on a warning too")
	root, err := parse(fs, args)
	if err != nil {
		return err
	}
	b, err := bundle.Load(root)
	if err != nil {
		return err
	}
	findings := check.Run(b)
	errs, warns := 0, 0
	for _, f := range findings {
		fmt.Fprintln(stdout, f)
		if f.Severity == check.Error {
			errs++
		} else {
			warns++
		}
	}
	fmt.Fprintf(stdout, "%d concepts, %d errors, %d warnings\n", len(b.Concepts), errs, warns)
	if errs > 0 || (*strict && warns > 0) {
		return errFailed
	}
	return nil
}

func runViz(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("viz", flag.ContinueOnError)
	out := fs.String("o", "", "the page to write (default <bundle>/viz.html)")
	name := fs.String("name", "", "the bundle's display name (default the folder's name)")
	layers := fs.String("layers", "", "the top-level folders in color order, comma-separated")
	verify := fs.Bool("check", false, "write nothing, and fail when the page is out of date")
	root, err := parse(fs, args)
	if err != nil {
		return err
	}
	if *out == "" {
		*out = filepath.Join(root, "viz.html")
	}
	if *name == "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return err
		}
		*name = viz.DefaultName(filepath.ToSlash(abs))
	}
	b, err := bundle.Load(root)
	if err != nil {
		return err
	}
	opts := viz.Options{Name: *name}
	if *layers != "" {
		opts.Layers = strings.Split(*layers, ",")
	}
	page, stats, err := viz.Render(b, opts)
	if err != nil {
		return err
	}
	if *verify {
		current, err := os.ReadFile(*out)
		if err != nil || !bytes.Equal(current, page) {
			fmt.Fprintf(stdout, "%s is out of date; run okf viz to regenerate it\n", *out)
			return errFailed
		}
		return nil
	}
	if err := os.WriteFile(*out, page, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %s: %d concepts, %d edges, %d bytes\n", *out, stats.Concepts, stats.Edges, stats.Bytes)
	return nil
}

// parse parses the flags and returns the one bundle argument.
func parse(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 1 {
		return "", errors.New("name one bundle directory")
	}
	return fs.Arg(0), nil
}
