package cmd_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/Emberwalker/spacehopper/cmd"
)

var _ = Describe("Cmd/Root", func() {
	It("should not error on no retries required", func() {
		code, err := cmd.Run(1, []int32{}, []string{}, []string{}, []string{"sh", "-c", "echo Hello"})
		Expect(err).To(BeNil())
		Expect(code).To(Equal(0))
	})
	It("should not error if a retry based on code is needed and succeeds", func() {
		os.Remove("/tmp/test1")
		code, err := cmd.Run(2, []int32{1}, []string{}, []string{}, []string{
			"sh",
			"-c",
			`[[ -f /tmp/test1 ]] && exit 0 || touch /tmp/test1 && exit 1`,
		})
		Expect(err).To(BeNil())
		Expect(code).To(Equal(0))
	})
	It("should not error if a retry based on string is needed and succeeds", func() {
		os.Remove("/tmp/test2")
		code, err := cmd.Run(2, []int32{}, []string{"FAIL"}, []string{}, []string{
			"sh",
			"-c",
			`[[ -f /tmp/test2 ]] && exit 0 || touch /tmp/test2 && echo "FAIL" && exit 1`,
		})
		Expect(err).To(BeNil())
		Expect(code).To(Equal(0))
	})
	It("should not error if a retry based on regex is needed and succeeds", func() {
		os.Remove("/tmp/test2")
		code, err := cmd.Run(2, []int32{}, []string{}, []string{"F.I."}, []string{
			"sh",
			"-c",
			`[[ -f /tmp/test2 ]] && exit 0 || touch /tmp/test2 && echo "FAIL" && exit 1`,
		})
		Expect(err).To(BeNil())
		Expect(code).To(Equal(0))
	})
	It("should error if a out of retries", func() {
		os.Remove("/tmp/test2")
		code, err := cmd.Run(2, []int32{}, []string{}, []string{"F.I."}, []string{
			"sh",
			"-c",
			`exit 1`,
		})
		Expect(err).To(BeNil())
		Expect(code).To(Equal(1))
	})
})
