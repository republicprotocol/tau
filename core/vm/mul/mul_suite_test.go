package mul_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMul(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mul Suite")
}
