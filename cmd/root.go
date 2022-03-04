package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/Emberwalker/spacehopper/pkg"

	"github.com/spf13/cobra"
)

var _verbose bool
var _codes []int32
var _strings, _patterns []string
var _maxAttempts int32

var rootCmd = &cobra.Command{
	Use:   "spacehopper",
	Short: "Reboot annoying CLI programs with irritating failure modes.",
	Long: `Spacehopper restarts commandline programs based on specific triggers, but otherwise passes through
	all standard input, output and exit codes verbatim.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		code, err := Run(_maxAttempts, _codes, _strings, _patterns, args)
		if err != nil {
			panic(err)
		}
		os.Exit(code)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&_verbose, "verbose", "v", false, "Enable verbose logging. May interfere with piped stderr streams.")
	rootCmd.Flags().Int32SliceVarP(&_codes, "codes", "c", []int32{}, "Error codes to restart the program on.")
	rootCmd.Flags().StringSliceVarP(&_strings, "strings", "s", []string{}, "Strings to restart the program on, anywhere in the stdout stream.")
	rootCmd.Flags().StringSliceVarP(&_patterns, "patterns", "p", []string{}, "Regex patterns to restart the program on, on any line of stdout output.")
	rootCmd.Flags().Int32VarP(&_maxAttempts, "max-attempts", "a", -1, "Maximum number of restarts. Defaults to infinite.")
}

func dbg(msg string, args ...interface{}) {
	if _verbose {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	}
}

func Run(maxAttempts int32, codes []int32, strings, patterns, args []string) (int, error) {
	matchers := buildMatchers(codes, strings, patterns)

	cmdPath, err := exec.LookPath(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to find binary: %v\n", args[0])
		return -1, err
	}

	attempts := 0
	exitCode := -250
loop:
	for {
		if maxAttempts != -1 && attempts >= int(maxAttempts) {
			dbg("Out of restart attempts; tried %s times", attempts)
			exitCode = -249
			break loop
		}
		attempts += 1

		cmd, stdout, stderr := spawn(cmdPath, args[1:])
		exited := make(chan struct{})
		restart := make(chan struct{})
		go monitorStream(restart, stdout, matchers)
		go monitorStream(restart, stderr, matchers)
		err = cmd.Start()
		if err != nil {
			return exitCode, err
		}

		go func() {
			waitErr := cmd.Wait()
			if _, isExitErr := waitErr.(*exec.ExitError); waitErr != nil && !isExitErr {
				panic(err)
			}
			close(exited)
		}()

	sel:
		select {
		case <-restart:
			cmd.Process.Kill()
			cmd.Wait()
			exitCode = -250
			break sel
		case <-exited:
			cmd.Wait()
			exitCode = cmd.ProcessState.ExitCode()
			codeMatched := false
			for _, matcher := range matchers {
				if matcher.MatchExitCode(exitCode) {
					codeMatched = true
				}
			}
			if codeMatched {
				break sel
			}
			break loop
		}
		close(restart)
	}

	return exitCode, nil
}

func buildMatchers(codes []int32, strings, patterns []string) []pkg.Matcher {
	var matchers = []pkg.Matcher{}
	for _, c := range codes {
		matchers = append(matchers, pkg.CompileCodeMatcher(int(c)))
	}
	for _, str := range strings {
		matchers = append(matchers, pkg.CompileLogMatcher(str))
	}
	for _, str := range patterns {
		matchers = append(matchers, pkg.CompileLogPatternMatcher(str))
	}
	return matchers
}

func spawn(cmdPath string, args []string) (cmd exec.Cmd, stdout io.ReadCloser, stderr io.ReadCloser) {
	cmd = *exec.Command(cmdPath, args...)
	cmd.Stdin = os.Stdin

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	io.TeeReader(stdout, os.Stdout)

	stderr, err = cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	io.TeeReader(stderr, os.Stderr)

	return
}

func monitorStream(channel chan struct{}, r io.Reader, matchers []pkg.Matcher) {
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		ln := scan.Text()
		for _, matcher := range matchers {
			if matcher.MatchLine(ln) {
				channel <- struct{}{}
				return
			}
		}
	}
}
