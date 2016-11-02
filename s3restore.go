// +build example

package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	region = "eu-west-1"
)

func parseTimestamp(timestamp string) (restoreTime time.Time) {

	i, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	return time.Unix(i, 0)

}

func main() {

	bucket := flag.String("bucket", "", "Source bucket")
	timestamp := flag.String("timestamp", "", "Restore point in time")
	key := flag.String("object", "", "Object name")
	flag.Parse()

	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		log.Fatal("failed to create session,", err)
	}

	svc := s3.New(sess)

	listVersionsParams := &s3.ListObjectVersionsInput{
		Bucket: aws.String(*bucket),
		Prefix: aws.String(*key),
	}

	listVersionResp, err := svc.ListObjectVersions(listVersionsParams)
	if err != nil {
		log.Fatal(err.Error())
	}
	restoreTime := parseTimestamp(*timestamp)

	for _, version := range listVersionResp.Versions {
		if (*key == *version.Key) && (restoreTime.After(*version.LastModified)) {
			fmt.Println(version)

			copyParams := &s3.CopyObjectInput{
				Bucket:     aws.String(*bucket),
				CopySource: aws.String(*bucket + "/" + *key + "?versionId=" + *version.VersionId),
				Key:        aws.String(*key + ".restored"),
			}
			copyResp, err := svc.CopyObject(copyParams)
			fmt.Println(copyResp)
			if err != nil {
				fmt.Println(err.Error())
				return

			}
			fmt.Printf("%s restored to %s\n", *key, *key+".restored")
			break
		}
	}
}
