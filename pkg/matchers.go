package pkg

import (
	"fmt"
	"regexp"
)

type Matcher interface {
	MatchLine(string) bool
	MatchExitCode(int) bool
}

type logMatcher struct {
	pattern regexp.Regexp
}

type codeMatcher struct {
	code int
}

func (m logMatcher) MatchLine(s string) bool {
	return m.pattern.MatchString(s)
}

func (m logMatcher) MatchExitCode(int) bool {
	return false
}

func (m logMatcher) String() string {
	return fmt.Sprintf("LogMatcher(%v)", m.pattern.String())
}

func (m codeMatcher) MatchLine(string) bool {
	return false
}

func (m codeMatcher) MatchExitCode(code int) bool {
	return m.code == code
}

func (m codeMatcher) String() string {
	return fmt.Sprintf("CodeMatcher(%v)", m.code)
}

func CompileLogMatcher(pattern string) Matcher {
	return logMatcher{
		pattern: *regexp.MustCompile(".*" + regexp.QuoteMeta(pattern) + ".*"),
	}
}

func CompileLogPatternMatcher(pattern string) Matcher {
	return logMatcher{
		pattern: *regexp.MustCompile(pattern),
	}
}

func CompileCodeMatcher(code int) Matcher {
	return codeMatcher{code}
}
