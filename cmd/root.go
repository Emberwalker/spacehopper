package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/Emberwalker/spacehopper/pkg"

	"github.com/spf13/cobra"

	gcmd "github.com/go-cmd/cmd"
)

var Version string = "devel"
var Verbose bool
var _codes []int32
var _strings, _patterns []string
var _maxAttempts int32

var rootCmd = &cobra.Command{
	Use:   "spacehopper",
	Short: "Reboot annoying CLI programs with irritating failure modes.",
	Long: `Spacehopper restarts commandline programs based on specific triggers, but otherwise passes through
	all standard input, output and exit codes verbatim.`,
	Args:    cobra.MinimumNArgs(1),
	Version: Version,
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
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose logging. May interfere with piped stderr streams.")
	rootCmd.Flags().Int32SliceVarP(&_codes, "codes", "c", []int32{}, "Error codes to restart the program on.")
	rootCmd.Flags().StringSliceVarP(&_strings, "strings", "s", []string{}, "Strings to restart the program on, anywhere in the stdout stream.")
	rootCmd.Flags().StringSliceVarP(&_patterns, "patterns", "p", []string{}, "Regex patterns to restart the program on, on any line of stdout output.")
	rootCmd.Flags().Int32VarP(&_maxAttempts, "max-attempts", "a", -1, "Maximum number of restarts. Defaults to infinite.")
}

func dbg(msg string, args ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	}
}

func Run(maxAttempts int32, codes []int32, strs, patterns, args []string) (int, error) {
	matchers := buildMatchers(codes, strs, patterns)

	attempts := 0
	exitCode := -250

loop:
	for {
		if maxAttempts != -1 && attempts >= int(maxAttempts) {
			dbg("Out of restart attempts; tried %v times", attempts)
			exitCode = -249
			break loop
		}
		attempts += 1
		dbg("Attempt %v", attempts)

		opts := gcmd.Options{
			Buffered:  false,
			Streaming: true,
		}
		cmd := gcmd.NewCmdOptions(opts, args[0], args[1:]...)

		restart := make(chan struct{})
		stdoutDone := make(chan struct{})
		stderrDone := make(chan struct{})

		go monitorStream(cmd.Stdout, os.Stdout, restart, stdoutDone, matchers)
		go monitorStream(cmd.Stderr, os.Stderr, restart, stderrDone, matchers)

		statusChan := cmd.StartWithStdin(os.Stdin)

		select {
		case <-restart:
			cmd.Stop()
			<-statusChan
			close(restart)
			continue loop
		case <-statusChan:
			// Give log matchers a chance to wrap up and check restart requests
			<-stdoutDone
			<-stderrDone
			select {
			case <-restart:
				continue loop
			default:
				// Pass
			}

			close(restart)
			exitCode = cmd.Status().Exit
			for _, matcher := range matchers {
				if matcher.MatchExitCode(exitCode) {
					continue loop
				}
			}
			break loop
		}
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

func monitorStream(stream chan string, osStream io.Writer, restart, done chan struct{}, matchers []pkg.Matcher) {
root:
	for {
		s, more := <-stream
		if more {
			fmt.Fprintln(osStream, s)
			for _, matcher := range matchers {
				if matcher.MatchLine(s) {
					restart <- struct{}{}
					break root
				}
			}
		} else {
			close(done)
			break root
		}
	}
}
