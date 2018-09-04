package vss_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVss(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vss Suite")
}
