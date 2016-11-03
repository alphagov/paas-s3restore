package main_test

import (
//	"fmt"
	"os/exec"
//	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/gbytes"
)

func s3r(args ...string) (*gexec.Session) {
	s3rbinary, err := gexec.Build("github.com/alphagov/paas-s3restore")
	Expect(err).ToNot(HaveOccurred())

	cmd := exec.Command(s3rbinary, args...)
	s3restore, err := gexec.Start(cmd, nil, nil)
	Expect(err).ToNot(HaveOccurred())

	return (s3restore)
}

var _ = Describe("S3restore", func() {

	Describe("User interface", func() {

		It("Errors with no params", func() {
			s3run := s3r()
			Eventually(s3run).Should(gexec.Exit())
			Expect(s3run.ExitCode()).To(Equal(2))
			Expect(s3run.Err).To(gbytes.Say("usage: "))
		})

		It("Identifies invalid command", func() {
			command := "test"
			s3run := s3r(command)
			Eventually(s3run).Should(gexec.Exit())
			Expect(s3run.ExitCode()).To(Equal(2))
			Expect(s3run.Err).To(gbytes.Say("\"" + command + "\" is not valid command."))
		})

		It("Identifies restore command", func() {
			command := "restore"
			s3run := s3r(command)
			Eventually(s3run).Should(gexec.Exit())
			Expect(s3run.ExitCode()).To(Equal(2))
			Expect(s3run.Err).To(gbytes.Say("timestamp"))
		})

		It("Doesn't implement list", func() {
			command := "list"
			s3run := s3r(command)
			Eventually(s3run).Should(gexec.Exit())
			Expect(s3run.ExitCode()).To(Equal(2))
			Expect(s3run.Err).To(gbytes.Say("Not implemented"))
		})

	})

})
//fmt.Printf("%#v",*s3r)