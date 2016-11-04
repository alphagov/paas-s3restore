package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	Version = "Placeholder"
)

type ParsedArgs struct {
	CommandName string
	Args        map[string]string
}

type S3svc struct {
	Svc *s3.S3
}

func NewS3svc() *S3svc {

	sess, err := session.NewSession()
	if err != nil {
		log.Fatal("failed to create session,", err)
	}

	return &S3svc{
		Svc: s3.New(sess),
	}
}

func parseTimestamp(timestamp string) (restoreTime time.Time) {

	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return time.Unix(i, 0)

}

func printUsage(command string, usage func()) func() {
	fmt.Fprintf(os.Stderr, "s3r version %s\n", Version)
	fmt.Fprintf(os.Stderr, "usage: s3r <command> <args>\n")
	switch command {
	case "restore":
		return func() {
			fmt.Fprintf(os.Stderr, " restore   Restore bucket objects\n")
			usage()
		}
	case "list":
		return func() {
			fmt.Fprintf(os.Stderr, " list   List object versions. Not implemented\n")
			usage()
		}
	default:
		fmt.Fprintf(os.Stderr, " restore   Restore bucket objects\n list   List object versions\n")
		return nil
	}
}

func parseArguments() ParsedArgs {
	restoreCommand := flag.NewFlagSet("restore", flag.ExitOnError)
	bkt := restoreCommand.String("bucket", "", "Source bucket. Default none. Required.")
	ts := restoreCommand.String("timestamp", "", "Restore point in time in UNIX timestamp format. Required.")
	prx := restoreCommand.String("prefix", "", "Object prefix. Default none.")

	listCommand := flag.NewFlagSet("list", flag.ExitOnError)
	since := listCommand.String("since", "", "Not implemented")

	if len(os.Args) == 1 {
		printUsage("", func() {})
		os.Exit(2)
	}

	switch os.Args[1] {
	case "restore":
		if err := restoreCommand.Parse(os.Args[2:]); err == nil {
			if *bkt == "" || *ts == "" {
				restoreCommand.Usage = printUsage("restore", restoreCommand.PrintDefaults)
				restoreCommand.Usage()
				os.Exit(2)
			}
		} else {
			log.Fatal(err)
		}
		return ParsedArgs{
			CommandName: "restore",
			Args: map[string]string{
				"bucket":    *bkt,
				"timestamp": *ts,
				"prefix":    *prx,
			},
		}

	case "list":
		if err := listCommand.Parse(os.Args[2:]); err == nil {
			restoreCommand.Usage = printUsage("list", listCommand.PrintDefaults)
			restoreCommand.Usage()
			fmt.Println(*since)
			os.Exit(2)
		} else {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
	return ParsedArgs{}
}

func (s *S3svc) ListVersions(bucket, prefix string) (*s3.ListObjectVersionsOutput, error) {

	listVersionsParams := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	listVersionResp, err := s.Svc.ListObjectVersions(listVersionsParams)
	if err != nil {
		return nil, err
	}
	return listVersionResp, nil
}

func (s *S3svc) CopyObject(bucket, key, version string) (*s3.CopyObjectOutput, error) {

	copyParams := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + key + "?versionId=" + version),
		Key:        aws.String(key),
	}
	copyResp, err := s.Svc.CopyObject(copyParams)
	if err != nil {
		return nil, err
	}
	return copyResp, nil
}

func (s *S3svc) RestoreObjects(bucket string, versions *s3.ListObjectVersionsOutput, restoreTime time.Time) error {

	var restored map[string]bool
	restored = make(map[string]bool)
	for _, version := range versions.Versions {
		if _, ok := restored[*version.Key]; !ok {
			// Amazon S3 returns object versions in the order in which they were stored,
			// with the most recently stored returned first.
			if restoreTime.After(*version.LastModified) {
				fmt.Printf("Restoring...\n %s\n", version)
				var copyResp *s3.CopyObjectOutput
				if !*version.IsLatest {
					var err error
					copyResp, err = s.CopyObject(bucket, *version.Key, *version.VersionId)
					if err != nil {
						return err
					}
				}
				restored[*version.Key] = true
				fmt.Printf("Restored:\n %s\n", copyResp)
			}
		}
	}
	return nil
}

func main() {
	s3svc := NewS3svc()
	args := parseArguments()

	switch args.CommandName {
	case "restore":
		bucket := args.Args["bucket"]
		prefix := args.Args["prefix"]
		timestamp := args.Args["timestamp"]

		listVersionResp, err := s3svc.ListVersions(bucket, prefix)
		if err != nil {
			log.Fatal(err.Error())
		}

		restoreTime := parseTimestamp(timestamp)
		err = s3svc.RestoreObjects(bucket, listVersionResp, restoreTime)
		if err != nil {
			log.Fatal(err)
		}

	case "list":
		log.Fatal("Not impleneted")
	}
}
