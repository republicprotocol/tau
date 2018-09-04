package pedersen_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPedersen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pedersen Suite")
}
