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
	Version = "dev"
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
				var copyResp *s3.CopyObjectOutput
				if !*version.IsLatest {
					fmt.Printf("Restoring...\n %s\n", version)
					var err error
					copyResp, err = s.CopyObject(bucket, *version.Key, *version.VersionId)
					if err != nil {
						return err
					}
					fmt.Printf("Restored:\n %s\n", copyResp)
				}
				restored[*version.Key] = true
			}
		}
	}
	return nil
}

func (s *S3svc) ListVersionsAtTimestamp(bucket string, versions *s3.ListObjectVersionsOutput, listTime time.Time) error {

	var listed map[string]bool
	listed = make(map[string]bool)
	fmt.Printf("%-40s %-34s %-30s\n", "object", "version", "timestamp")
	for _, version := range versions.Versions {
		if _, ok := listed[*version.Key]; !ok {
			// Amazon S3 returns object versions in the order in which they were stored,
			// with the most recently stored returned first.
			if listTime.After(*version.LastModified) {
				fmt.Printf("%-40s %-34s %-30s\n", *version.Key, *version.VersionId, *version.LastModified)
				listed[*version.Key] = true
			}
		}
	}
	return nil
}

func parseTimestamp(timestamp string) (restoreTime time.Time) {

	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Fatal("Failed parsing timestamp:", err)
	}
	return time.Unix(i, 0)

}

func printUsage(command string, usage func()) func() {
	fmt.Fprintf(os.Stderr, "s3r version %s\n", Version)
	fmt.Fprintf(os.Stderr, "usage: s3r <command> <args>\n")
	restoreHelp := " restore  Restore bucket objects\n"
	listHelp := " list     List versions at the point of time.\n"
	switch command {
	case "restore":
		return func() {
			fmt.Fprintf(os.Stderr, restoreHelp)
			usage()
		}
	case "list":
		return func() {
			fmt.Fprintf(os.Stderr, listHelp)
			usage()
		}
	default:
		fmt.Fprintf(os.Stderr, restoreHelp+listHelp)
		return nil
	}
}

func parseArguments() ParsedArgs {
	restoreCommand := flag.NewFlagSet("restore", flag.ExitOnError)
	rbkt := restoreCommand.String("bucket", "", "Source bucket. Default none. Required.")
	rts := restoreCommand.String("timestamp", "", "Restore point in time in UNIX timestamp format. Required.")
	rprx := restoreCommand.String("prefix", "", "Object prefix. Default none.")

	listCommand := flag.NewFlagSet("list", flag.ExitOnError)
	lts := listCommand.String("timestamp", "", "Time baseline of versions. Required.")
	lbkt := listCommand.String("bucket", "", "Source bucket. Default none. Required.")
	lprx := listCommand.String("prefix", "", "Object prefix. Default none.")

	if len(os.Args) == 1 {
		printUsage("", func() {})
		os.Exit(2)
	}

	switch os.Args[1] {
	case "restore":
		if err := restoreCommand.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if *rbkt == "" || *rts == "" {
			restoreCommand.Usage = printUsage("restore", restoreCommand.PrintDefaults)
			restoreCommand.Usage()
			os.Exit(2)
		}
		return ParsedArgs{
			CommandName: "restore",
			Args: map[string]string{
				"bucket":    *rbkt,
				"timestamp": *rts,
				"prefix":    *rprx,
			},
		}

	case "list":
		if err := listCommand.Parse(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		if *lbkt == "" || *lts == "" {
			listCommand.Usage = printUsage("list", listCommand.PrintDefaults)
			listCommand.Usage()
			os.Exit(2)
		}
		return ParsedArgs{
			CommandName: "list",
			Args: map[string]string{
				"bucket":    *lbkt,
				"timestamp": *lts,
				"prefix":    *lprx,
			},
		}
	default:
		fmt.Fprintf(os.Stderr, "%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
	return ParsedArgs{}
}

func main() {
	s3svc := NewS3svc()
	args := parseArguments()

	bucket := args.Args["bucket"]
	prefix := args.Args["prefix"]
	timestamp := args.Args["timestamp"]

	listVersionResp, err := s3svc.ListVersions(bucket, prefix)
	if err != nil {
		log.Fatal(err.Error())
	}
	parsedTimestamp := parseTimestamp(timestamp)
	switch args.CommandName {
	case "restore":

		err = s3svc.RestoreObjects(bucket, listVersionResp, parsedTimestamp)
		if err != nil {
			log.Fatal(err)
		}

	case "list":
		err = s3svc.ListVersionsAtTimestamp(bucket, listVersionResp, parsedTimestamp)
		if err != nil {
			log.Fatal(err)
		}
	}
}
