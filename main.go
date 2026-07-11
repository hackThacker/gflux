package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

type pattern struct {
	Flags    string   `json:"flags,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
	Engine   string   `json:"engine,omitempty"`
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  cat urls.txt | gflux PATTERN        pipe mode\n")
		fmt.Fprintf(os.Stderr, "  gflux PATTERN [path]                file/dir mode\n")
		fmt.Fprintf(os.Stderr, "  cat urls.txt | gflux --all          run ALL patterns on stdin\n")
		fmt.Fprintf(os.Stderr, "  gflux --all [path]                  run ALL patterns on path\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	var saveMode bool
	flag.BoolVar(&saveMode, "save", false, "save a pattern")

	var listMode bool
	flag.BoolVar(&listMode, "list", false, "list available patterns")

	var dumpMode bool
	flag.BoolVar(&dumpMode, "dump", false, "print the grep command rather than executing it")

	var allMode bool
	flag.BoolVar(&allMode, "all", false, "run ALL patterns; results saved to --results dir")

	var resultsDir string
	flag.StringVar(&resultsDir, "results", "./results", "output directory used with --all")

	flag.Parse()

	if listMode {
		pats, err := getPatterns()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return
		}
		fmt.Println(strings.Join(pats, "\n"))
		return
	}

	if saveMode {
		name := flag.Arg(0)
		flags := flag.Arg(1)
		pat := flag.Arg(2)
		err := savePattern(name, flags, pat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
		}
		return
	}

	patDir, err := getPatternDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open user pattern directory")
		return
	}

	if allMode {
		path := flag.Arg(0)
		runAllPatterns(patDir, path, resultsDir)
		return
	}

	patName := flag.Arg(0)
	if patName == "" {
		flag.Usage()
		return
	}
	files := flag.Arg(1)
	if files == "" {
		files = "."
	}

	pat, err := loadPattern(patDir, patName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if dumpMode {
		fmt.Printf("grep %v %q %v\n", pat.Flags, pat.Pattern, files)
		return
	}

	runPattern(pat, files, os.Stdin, os.Stdout)
}

func loadPattern(patDir, patName string) (*pattern, error) {
	filename := filepath.Join(patDir, patName+".json")
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("no such pattern: %s", patName)
	}
	defer f.Close()
	pat := &pattern{}
	if err := json.NewDecoder(f).Decode(pat); err != nil {
		return nil, fmt.Errorf("pattern file malformed: %s", err)
	}
	if pat.Pattern == "" {
		if len(pat.Patterns) == 0 {
			return nil, fmt.Errorf("pattern file has no patterns: %s", filename)
		}
		pat.Pattern = "(" + strings.Join(pat.Patterns, "|") + ")"
	}
	return pat, nil
}

func runPattern(pat *pattern, path string, stdin io.Reader, stdout io.Writer) {
	operator := "grep"
	if pat.Engine != "" {
		operator = pat.Engine
	}
	flagArgs := strings.Fields(pat.Flags)
	var args []string
	if stdinIsPipe() {
		args = append(flagArgs, pat.Pattern)
	} else {
		if path == "" {
			path = "."
		}
		args = append(flagArgs, pat.Pattern, path)
	}
	cmd := exec.Command(operator, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func runAllPatterns(patDir, path, resultsDir string) {
	patternFiles, err := filepath.Glob(patDir + "/*.json")
	if err != nil || len(patternFiles) == 0 {
		fmt.Fprintln(os.Stderr, "no patterns found in:", patDir)
		return
	}
	var stdinBuf []byte
	usingStdin := stdinIsPipe()
	if usingStdin {
		stdinBuf, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error reading stdin:", err)
			return
		}
		if len(stdinBuf) == 0 {
			fmt.Fprintln(os.Stderr, "stdin is empty")
			return
		}
	} else if path == "" {
		fmt.Fprintln(os.Stderr, "--all needs piped stdin OR a path argument")
		fmt.Fprintln(os.Stderr, "  e.g:  cat urls.txt | gflux --all")
		fmt.Fprintln(os.Stderr, "  e.g:  gflux --all ~/dyson/")
		return
	}
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		fmt.Fprintln(os.Stderr, "cannot create results dir:", err)
		return
	}
	fmt.Fprintf(os.Stderr, "[gflux --all] patterns : %s  (%d)\n", patDir, len(patternFiles))
	if usingStdin {
		fmt.Fprintf(os.Stderr, "[gflux --all] input    : stdin  (%d bytes)\n", len(stdinBuf))
	} else {
		fmt.Fprintf(os.Stderr, "[gflux --all] input    : %s\n", path)
	}
	fmt.Fprintf(os.Stderr, "[gflux --all] results  : %s\n\n", resultsDir)
	success, empty, errored := 0, 0, 0
	for _, pf := range patternFiles {
		patName := strings.TrimSuffix(filepath.Base(pf), ".json")
		outputFile := filepath.Join(resultsDir, patName+".txt")
		pat, err := loadPattern(patDir, patName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  [SKIP]  %-28s %s\n", patName, err)
			errored++
			continue
		}
		operator := "grep"
		if pat.Engine != "" {
			operator = pat.Engine
		}
		flagArgs := strings.Fields(pat.Flags)
		var args []string
		var cmdStdin io.Reader
		if usingStdin {
			args = append(flagArgs, pat.Pattern)
			cmdStdin = bytes.NewReader(stdinBuf)
		} else {
			args = append(flagArgs, pat.Pattern, path)
			cmdStdin = nil
		}
		var outBuf bytes.Buffer
		cmd := exec.Command(operator, args...)
		cmd.Stdin = cmdStdin
		cmd.Stdout = &outBuf
		cmd.Stderr = nil
		cmd.Run()
		result := strings.TrimRight(outBuf.String(), "\n")
		if err := os.WriteFile(outputFile, []byte(result+"\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  [ERROR] %-28s cannot write: %s\n", patName, err)
			errored++
			continue
		}
		if result == "" {
			fmt.Fprintf(os.Stderr, "  [EMPTY] %-28s -> %s\n", patName, outputFile)
			empty++
		} else {
			lines := strings.Count(result, "\n") + 1
			fmt.Fprintf(os.Stderr, "  [OK]    %-28s -> %s  (%d matches)\n", patName, outputFile, lines)
			success++
		}
	}
	fmt.Fprintf(os.Stderr, "\n[gflux --all] done  matches=%d  empty=%d  errors=%d\n", success, empty, errored)
}

func getPatternDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	path := filepath.Join(usr.HomeDir, ".config/gflux")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(usr.HomeDir, ".gflux")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(usr.HomeDir, ".config/gf")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	return filepath.Join(usr.HomeDir, ".gf"), nil
}

func savePattern(name, flags, pat string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if pat == "" {
		return errors.New("pattern cannot be empty")
	}
	p := &pattern{Flags: flags, Pattern: pat}
	patDir, err := getPatternDir()
	if err != nil {
		return fmt.Errorf("failed to determine pattern directory: %s", err)
	}
	path := filepath.Join(patDir, name+".json")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("failed to create pattern file: %s", err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	if err := enc.Encode(p); err != nil {
		return fmt.Errorf("failed to write pattern file: %s", err)
	}
	return nil
}

func getPatterns() ([]string, error) {
	out := []string{}
	patDir, err := getPatternDir()
	if err != nil {
		return out, fmt.Errorf("failed to determine pattern directory: %s", err)
	}
	files, err := filepath.Glob(patDir + "/*.json")
	if err != nil {
		return out, err
	}
	for _, f := range files {
		out = append(out, strings.TrimSuffix(filepath.Base(f), ".json"))
	}
	return out, nil
}

func stdinIsPipe() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
