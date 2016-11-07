package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPaasS3restore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PaasS3restore Suite")
}
