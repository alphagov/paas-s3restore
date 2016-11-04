package main_test

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"time"

	. "github.com/alphagov/paas-s3restore"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/awstesting/unit"
	"github.com/aws/aws-sdk-go/service/s3"
)

func s3r(args ...string) *gexec.Session {
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

	Describe("Bucket restore", func() {

		It("Picks correct object version to restore (oldest)", func() {
			err, restoredVersion := restore(defaultVersions(), time.Unix(150, 0))

			Expect(err).To(BeNil())
			Expect(restoredVersion).To(Equal("v1"))

		})

		It("Picks correct object version to restore (middle)", func() {
			err, restoredVersion := restore(defaultVersions(), time.Unix(250, 0))

			Expect(err).To(BeNil())
			Expect(restoredVersion).To(Equal("v2"))
		})

		It("Doesn't restore if not modified after restore time", func() {
			err, restoredVersion := restore(defaultVersions(), time.Unix(1000, 0))

			Expect(err).To(BeNil())
			Expect(restoredVersion).To(Equal(""))
		})

		It("Works correctly with empty version list", func() {
			err, restoredVersion := restore([]*s3.ObjectVersion{}, time.Unix(1000, 0))

			Expect(err).To(BeNil())
			Expect(restoredVersion).To(Equal(""))
		})

	})

})

func restore(versions []*s3.ObjectVersion, time time.Time) (error, string) {
	restoredVersion := ""
	s := s3.New(unit.Session)

	s.Handlers.Send.Clear()
	s.Handlers.Send.PushBack(func(r *request.Request) {
		params := r.Params.(*s3.CopyObjectInput)
		re := regexp.MustCompile(".*?versionId=")
		restoredVersion = re.ReplaceAllString(*params.CopySource, "")

		reader := ioutil.NopCloser(bytes.NewReader([]byte(`
			<?xml version="1.0" encoding="UTF-8"?>
			<CopyObjectResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
				<LastModified>2000-01-01T0:00:00Z</LastModified>
				<ETag>` + *params.Key + `</ETag>
			</CopyObjectResult>`)))
		r.HTTPResponse = &http.Response{StatusCode: 200, Body: reader}
	})

	mockS3 := &S3svc{Svc: s}
	testList := s3.ListObjectVersionsOutput{Versions: versions}
	err := mockS3.RestoreObjects("mybucket", &testList, time)

	return err, restoredVersion
}

func defaultVersions() []*s3.ObjectVersion {
	v1 := s3.ObjectVersion{
		Key:          aws.String("a"),
		IsLatest:     aws.Bool(false),
		LastModified: aws.Time(time.Unix(111, 0)),
		VersionId:    aws.String("v1"),
	}

	v2 := s3.ObjectVersion{
		Key:          aws.String("a"),
		IsLatest:     aws.Bool(false),
		LastModified: aws.Time(time.Unix(222, 0)),
		VersionId:    aws.String("v2"),
	}

	v3 := s3.ObjectVersion{
		Key:          aws.String("a"),
		IsLatest:     aws.Bool(true),
		LastModified: aws.Time(time.Unix(333, 0)),
		VersionId:    aws.String("v3"),
	}

	return []*s3.ObjectVersion{&v3, &v2, &v1}
}
