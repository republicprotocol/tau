package algebra_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAlgebra(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Algebra Suite")
}
