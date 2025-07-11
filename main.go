package main

import (
	"bufio"
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/google/shlex"
	flag "github.com/spf13/pflag"
)

var Colors = []string{
	"\033[1;31m", // Red
	"\033[1;32m", // Green
	"\033[1;33m", // Yellow
	"\033[1;34m", // Blue
	"\033[1;35m", // Magenta
	"\033[1;36m", // Cyan
	"\033[0;91m", // Bright Red
	"\033[0;92m", // Bright Green
	"\033[0;94m", // Bright Blue
	"\033[0;95m", // Bright Magenta
	"\033[0;96m", // Bright Cyan
}

var Reset = "\033[0m"

var maxWidth int
var args Args

type Task struct {
	Label   string
	Color   int
	Command string
}

type TaskResult struct {
	Task     Task
	ExitCode int
	Err      error
}

type Args struct {
	Tasks       []Task
	File        string
	Quiet       bool
	AbortOnFail bool
	Raw         bool
	NoColor     bool
	Max         int
}

func getColor(label string) int {
	h := fnv.New32a()
	h.Write([]byte(label))
	return int(h.Sum32()) % len(Colors)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

func parseTasks(lines []string) []Task {
	var tasks []Task
	labelCount := map[string]int{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		label := ""
		cmd := line

		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			if end > 0 {
				label = line[1:end]
				cmd = strings.TrimSpace(line[end+1:])
			}
		}

		if label == "" {
			fields := strings.Fields(cmd)
			base := filepath.Base(fields[0])
			ext := filepath.Ext(base)
			label = strings.TrimSuffix(base, ext)
		}

		origLabel := label
		count := labelCount[origLabel]
		if count > 0 {
			label = fmt.Sprintf("%s (%d)", origLabel, count+1)
		}
		labelCount[origLabel] = count + 1

		tasks = append(tasks, Task{
			Label:   label,
			Command: cmd,
			Color:   getColor(label),
		})
	}

	return tasks
}

func calcMaxWidth(tasks []Task) int {
	width := 0
	for _, t := range tasks {
		if len(t.Label) > width {
			width = len(t.Label)
		}
	}
	return width
}

func writeOut(task Task, message string, w io.Writer) {
	if args.Raw {
		fmt.Fprintln(w, message)
		return
	}

	color := ""
	reset := ""
	if !args.NoColor {
		color = Colors[task.Color]
		reset = Reset
	}
	padding := strings.Repeat(" ", maxWidth-len(task.Label))
	fmt.Fprintf(w, "%s%s%s |%s %s\n", color, task.Label, padding, reset, message)
}

func parseArgs() {
	flag.StringVarP(&args.File, "from", "f", "",
		"Load tasks from file")
	flag.BoolVarP(&args.Quiet, "quiet", "q", false,
		"Suppress command run messages and summary")
	flag.BoolVarP(&args.Raw, "raw", "r", false,
		"Disable output labels and suppress extra logs (implies quiet mode)")
	flag.BoolVar(&args.NoColor, "no-color", false,
		"Disable color output")
	flag.IntVar(&args.Max, "max", 0,
		"Maximum concurrent tasks (0 = unlimited)")
	flag.BoolVarP(&args.AbortOnFail, "abort-on-fail", "a", false,
		"Stop all tasks if any fail")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  stampede [options] '[label] command' '[label] command' ...
  stampede --from tasks.txt

Examples:
  # Run commands with optional labels (labels in square brackets)
  stampede "[Google] ping -c 3 8.8.8.8" "[Cloudflare] ping -c 3 1.1.1.1"
  
  # Run commands without labels; labels will be inferred from executable names
  stampede "ping -c 3 8.8.8.8" "ping -c 3 1.1.1.1"

  # Load commands from file (one per line, optional labels allowed)
  stampede --from commands.txt

Flags:
`)
		flag.PrintDefaults()
	}

	flag.Parse()

	if args.Raw {
		args.Quiet = true
	}

	var lines []string
	if args.File != "" {
		fileLines, err := readLines(args.File)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading file:", err)
			os.Exit(1)
		}
		lines = fileLines
	} else {
		lines = flag.Args()
	}

	args.Tasks = parseTasks(lines)

	if len(args.Tasks) == 0 {
		fmt.Fprintln(os.Stderr, "No tasks provided.\n")
		flag.Usage()
		os.Exit(1)
	}
}

func runTask(ctx context.Context, task Task, wg *sync.WaitGroup, sem chan struct{}, exitOnFail *int32, results chan<- TaskResult) {
	defer wg.Done()

	sem <- struct{}{}
	defer func() { <-sem }()

	if atomic.LoadInt32(exitOnFail) == 1 {
		results <- TaskResult{task, -1, fmt.Errorf("aborted")}
		return
	}

	words, err := shlex.Split(task.Command)
	if err != nil {
		results <- TaskResult{task, -1, err}
		return
	}
	if len(words) == 0 {
		results <- TaskResult{task, -1, fmt.Errorf("empty command")}
		return
	}

	cmd := exec.CommandContext(ctx, words[0], words[1:]...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		results <- TaskResult{task, -1, err}
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		results <- TaskResult{task, -1, err}
		return
	}

	if !args.Quiet {
		writeOut(task, "Running: "+task.Command, os.Stdout)
	}

	if err := cmd.Start(); err != nil {
		results <- TaskResult{task, -1, err}
		return
	}

	var wgOut sync.WaitGroup
	wgOut.Add(2)
	go copyOutput(task, stdout, os.Stdout, &wgOut)
	go copyOutput(task, stderr, os.Stderr, &wgOut)
	wgOut.Wait()

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	if args.AbortOnFail && exitCode != 0 {
		atomic.StoreInt32(exitOnFail, 1)
	}

	results <- TaskResult{task, exitCode, err}
}

func copyOutput(task Task, r io.Reader, w io.Writer, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		writeOut(task, scanner.Text(), w)
	}
}

func main() {
	parseArgs()
	maxWidth = calcMaxWidth(args.Tasks)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\n\nReceived signal: %v. Finishing running tasks...\n", sig)
		cancel()
	}()

	sem := make(chan struct{}, args.Max)
	if args.Max <= 0 {
		sem = make(chan struct{}, len(args.Tasks))
	}

	var wg sync.WaitGroup
	results := make(chan TaskResult, len(args.Tasks))
	var exitOnFail int32 = 0

	for _, task := range args.Tasks {
		wg.Add(1)
		go runTask(ctx, task, &wg, sem, &exitOnFail, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	successCount := 0
	failCount := 0
	failLabels := []string{}

	for res := range results {
		if res.ExitCode == 0 {
			successCount++
		} else {
			failCount++
			failLabels = append(failLabels, res.Task.Label)
			writeOut(res.Task, fmt.Sprintf("Error: %s", res.Err), os.Stderr)
		}
	}

	if !args.Quiet {
		fmt.Printf("\nTasks finished: %d / %d succeeded, %d failed\n", successCount, len(args.Tasks), failCount)
		if failCount > 0 {
			fmt.Printf("Failed tasks: %s\n", strings.Join(failLabels, ", "))
		} else {
			fmt.Println("All tasks completed successfully!")
		}
	}

	if failCount > 0 {
		os.Exit(1)
	}
}
