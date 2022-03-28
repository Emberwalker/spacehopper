package pkg_test

import (
	"github.com/Emberwalker/spacehopper/pkg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Matchers", func() {
	Context("Plain log matchers", func() {
		var matcher = pkg.CompileLogMatcher("Test")
		It("should match a matching log line", func() {
			Expect(matcher.MatchLine("Something Test something")).To(Equal(true))
		})
		It("should not match a unmatching log line", func() {
			Expect(matcher.MatchLine("Best")).To(Equal(false))
		})
		It("should not match exit codes", func() {
			Expect(matcher.MatchExitCode(0)).To(Equal(false))
		})
	})

	Context("Plain log matchers corner cases", func() {
		It("should not match something that looks like a catch-all regex", func() {
			var matcher = pkg.CompileLogMatcher("[Test]")
			Expect(matcher.MatchLine("Best")).To(Equal(false))
		})
		It("should match something that looks like a catch-all regex", func() {
			var matcher = pkg.CompileLogMatcher("[Test]")
			Expect(matcher.MatchLine("Something [Test] blah")).To(Equal(true))
		})
	})

	Context("Pattern log matchers", func() {
		var matcher = pkg.CompileLogPatternMatcher("[a|b]")
		It("should match a matching log line", func() {
			Expect(matcher.MatchLine("a")).To(Equal(true))
		})
		It("should not match a unmatching log line", func() {
			Expect(matcher.MatchLine("c")).To(Equal(false))
		})
		It("should not match exit codes", func() {
			Expect(matcher.MatchExitCode(0)).To(Equal(false))
		})
	})

	Context("Exit code matchers", func() {
		var matcher = pkg.CompileCodeMatcher(1)
		It("should match a matching code", func() {
			Expect(matcher.MatchExitCode(1)).To(Equal(true))
		})
		It("should not match a unmatching code", func() {
			Expect(matcher.MatchExitCode(2)).To(Equal(false))
		})
		It("should not match log lines", func() {
			Expect(matcher.MatchLine("Test")).To(Equal(false))
		})
	})
})
