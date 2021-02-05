package runner_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGoStressRunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GoStressRunner Suite")
}
