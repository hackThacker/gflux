package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed .gflux/*.json
var embeddedPatterns embed.FS

type pattern struct {
	Flags    string   `json:"flags,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
	Engine   string   `json:"engine,omitempty"`
}

type patternSource struct {
	Name   string
	Path   string // filesystem path or "embedded:"
	Source string // "local", "user", or "embedded"
}

func main() {
	var saveMode bool
	flag.BoolVar(&saveMode, "save", false, "save a pattern")

	var listMode bool
	flag.BoolVar(&listMode, "list", false, "list available patterns")

	var listPatternsMode bool
	flag.BoolVar(&listPatternsMode, "list-patterns", false, "list patterns grouped by source")

	var dumpMode bool
	flag.BoolVar(&dumpMode, "dump", false, "print the grep command rather than executing it")

	var allMode bool
	flag.BoolVar(&allMode, "all", false, "run ALL patterns; results saved to --results dir")

	var resultsDir string
	flag.StringVar(&resultsDir, "results", "./results", "output directory used with --all")

	var silentMode bool
	flag.BoolVar(&silentMode, "silent", false, "silent mode (no banner)")

	flag.Usage = func() {
		showBanner(silentMode)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  cat urls.txt | gflux PATTERN        pipe mode\n")
		fmt.Fprintf(os.Stderr, "  gflux PATTERN [path]                file/dir mode\n")
		fmt.Fprintf(os.Stderr, "  cat urls.txt | gflux --all          run ALL patterns on stdin\n")
		fmt.Fprintf(os.Stderr, "  gflux --all [path]                  run ALL patterns on path\n")
		fmt.Fprintf(os.Stderr, "  gflux --list-patterns               list patterns grouped by source\n")
		fmt.Fprintf(os.Stderr, "  gflux init                          initialize user custom pattern folder\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	showBanner(silentMode)

	if flag.NArg() > 0 && flag.Arg(0) == "init" {
		initCmd()
		return
	}

	if listPatternsMode {
		listPatternsCmd()
		return
	}

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

	if allMode {
		path := flag.Arg(0)
		runAllPatterns(path, resultsDir)
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

	pat, err := loadPattern(patName)
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

func findPattern(patName string) (io.ReadCloser, string, error) {
	// 1. Check local directory: ./.gflux/
	localPath := filepath.Join(".gflux", patName+".json")
	if f, err := os.Open(localPath); err == nil {
		return f, localPath, nil
	}

	// 2. Check user directory
	userDirs, err := getUserPatternDirs()
	if err == nil {
		for _, dir := range userDirs {
			userPath := filepath.Join(dir, patName+".json")
			if f, err := os.Open(userPath); err == nil {
				return f, userPath, nil
			}
		}
	}

	// 3. Check embedded patterns
	embedPath := ".gflux/" + patName + ".json"
	if f, err := embeddedPatterns.Open(embedPath); err == nil {
		return f, "embedded:" + embedPath, nil
	}

	return nil, "", fmt.Errorf("no such pattern: %s", patName)
}

func getUserPatternDirs() ([]string, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(usr.HomeDir, ".gflux"),
		filepath.Join(usr.HomeDir, ".config/gflux"),
		filepath.Join(usr.HomeDir, ".gf"),
		filepath.Join(usr.HomeDir, ".config/gf"),
	}, nil
}

func getAllPatterns() ([]patternSource, error) {
	patternsMap := make(map[string]patternSource)

	// 1. Scan embedded patterns (lowest priority)
	entries, err := embeddedPatterns.ReadDir(".gflux")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				name := strings.TrimSuffix(entry.Name(), ".json")
				patternsMap[name] = patternSource{
					Name:   name,
					Path:   "embedded:.gflux/" + entry.Name(),
					Source: "embedded",
				}
			}
		}
	}

	// 2. Scan user patterns (medium priority)
	userDirs, _ := getUserPatternDirs()
	for _, dir := range userDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err == nil {
			for _, f := range files {
				name := strings.TrimSuffix(filepath.Base(f), ".json")
				patternsMap[name] = patternSource{
					Name:   name,
					Path:   f,
					Source: "user",
				}
			}
		}
	}

	// 3. Scan local patterns (highest priority)
	files, err := filepath.Glob(filepath.Join(".gflux", "*.json"))
	if err == nil {
		for _, f := range files {
			name := strings.TrimSuffix(filepath.Base(f), ".json")
			patternsMap[name] = patternSource{
				Name:   name,
				Path:   f,
				Source: "local",
			}
		}
	}

	var list []patternSource
	for _, ps := range patternsMap {
		list = append(list, ps)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})

	return list, nil
}

func loadPattern(patName string) (*pattern, error) {
	f, source, err := findPattern(patName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	pat := &pattern{}
	if err := json.NewDecoder(f).Decode(pat); err != nil {
		return nil, fmt.Errorf("pattern file malformed (%s): %s", source, err)
	}
	if pat.Pattern == "" {
		if len(pat.Patterns) == 0 {
			return nil, fmt.Errorf("pattern file has no patterns: %s", source)
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

func runAllPatterns(path, resultsDir string) {
	all, err := getAllPatterns()
	if err != nil || len(all) == 0 {
		fmt.Fprintln(os.Stderr, "no patterns found")
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
	fmt.Fprintf(os.Stderr, "[gflux --all] running %d patterns\n", len(all))
	if usingStdin {
		fmt.Fprintf(os.Stderr, "[gflux --all] input    : stdin  (%d bytes)\n", len(stdinBuf))
	} else {
		fmt.Fprintf(os.Stderr, "[gflux --all] input    : %s\n", path)
	}
	fmt.Fprintf(os.Stderr, "[gflux --all] results  : %s\n\n", resultsDir)
	success, empty, errored := 0, 0, 0
	for _, ps := range all {
		patName := ps.Name
		outputFile := filepath.Join(resultsDir, patName+".txt")
		pat, err := loadPattern(patName)
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
	path := filepath.Join(usr.HomeDir, ".gflux")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(usr.HomeDir, ".config/gflux")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(usr.HomeDir, ".gf")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	path = filepath.Join(usr.HomeDir, ".config/gf")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return path, nil
	}
	return filepath.Join(usr.HomeDir, ".gflux"), nil
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
	all, err := getAllPatterns()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range all {
		names = append(names, p.Name)
	}
	return names, nil
}

func listPatternsCmd() {
	var local, user, embedded []string

	// 1. Scan local patterns (./.gflux/)
	files, err := filepath.Glob(filepath.Join(".gflux", "*.json"))
	if err == nil {
		for _, f := range files {
			local = append(local, strings.TrimSuffix(filepath.Base(f), ".json"))
		}
	}
	sort.Strings(local)

	// 2. Scan user patterns (~/.gflux/)
	userDirs, _ := getUserPatternDirs()
	for _, dir := range userDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err == nil {
			for _, f := range files {
				user = append(user, strings.TrimSuffix(filepath.Base(f), ".json"))
			}
		}
	}
	user = deduplicateAndSort(user)

	// 3. Scan embedded patterns
	entries, err := embeddedPatterns.ReadDir(".gflux")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				embedded = append(embedded, strings.TrimSuffix(entry.Name(), ".json"))
			}
		}
	}
	sort.Strings(embedded)

	fmt.Println("Local patterns (./.gflux/):")
	if len(local) > 0 {
		for _, p := range local {
			fmt.Printf("  - %s\n", p)
		}
	} else {
		fmt.Println("  (none)")
	}
	fmt.Println()

	fmt.Println("User custom patterns (~/.gflux/):")
	if len(user) > 0 {
		for _, p := range user {
			fmt.Printf("  - %s\n", p)
		}
	} else {
		fmt.Println("  (none)")
	}
	fmt.Println()

	fmt.Println("Embedded default patterns:")
	if len(embedded) > 0 {
		for _, p := range embedded {
			fmt.Printf("  - %s\n", p)
		}
	} else {
		fmt.Println("  (none)")
	}
}

func deduplicateAndSort(s []string) []string {
	m := make(map[string]bool)
	for _, val := range s {
		m[val] = true
	}
	var res []string
	for val := range m {
		res = append(res, val)
	}
	sort.Strings(res)
	return res
}

func initCmd() {
	usr, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current user: %s\n", err)
		return
	}
	dir := filepath.Join(usr.HomeDir, ".gflux")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory %s: %s\n", dir, err)
		return
	}
	fmt.Printf("Created user pattern directory: %s\n", dir)

	autoCopy := false
	for _, arg := range flag.Args() {
		if arg == "-y" || arg == "--yes" || arg == "-force" || arg == "--force" {
			autoCopy = true
			break
		}
	}

	copyPatterns := autoCopy
	if !copyPatterns {
		if isTerminal(os.Stdin) {
			fmt.Print("Do you want to copy embedded default patterns to ~/.gflux? [y/N]: ")
			var response string
			_, err := fmt.Scanln(&response)
			if err == nil {
				response = strings.ToLower(strings.TrimSpace(response))
				if response == "y" || response == "yes" {
					copyPatterns = true
				}
			}
		} else {
			fmt.Println("Non-interactive terminal detected. Copying default patterns automatically.")
			copyPatterns = true
		}
	}

	if copyPatterns {
		entries, err := embeddedPatterns.ReadDir(".gflux")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading embedded patterns: %s\n", err)
			return
		}
		copiedCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				srcPath := ".gflux/" + entry.Name()
				destPath := filepath.Join(dir, entry.Name())

				srcFile, err := embeddedPatterns.Open(srcPath)
				if err != nil {
					continue
				}
				destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					srcFile.Close()
					continue
				}

				_, err = io.Copy(destFile, srcFile)
				srcFile.Close()
				destFile.Close()
				if err == nil {
					copiedCount++
				}
			}
		}
		fmt.Printf("Successfully copied %d default pattern files to %s\n", copiedCount, dir)
	} else {
		fmt.Println("Skipped copying default pattern files.")
	}
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func stdinIsPipe() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
