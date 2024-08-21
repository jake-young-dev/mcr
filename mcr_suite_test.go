package mcr_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMcr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mcr Suite")
}
