package vm_test

import (
	"github.com/republicprotocol/co-go"
	"github.com/republicprotocol/smpc-go/core/buffer"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/republicprotocol/smpc-go/core/vm"
)

var _ = Describe("Virtual Machine", func() {

	// initVMs for a secure multi-party computation network. The VMs will
	// communicate to execute processes.
	initVMs := func(n, k uint, cap int) ([]VM, []buffer.ReaderWriter, []buffer.ReaderWriter) {
		// Initialize the VMs
		ins := make([]buffer.ReaderWriter, n)
		outs := make([]buffer.ReaderWriter, n)
		vms := make([]VM, n)
		for i := uint(0); i < n; i++ {
			ins[i] = buffer.NewReaderWriter(bufferCap)
			outs[i] = buffer.NewReaderWriter(bufferCap)
			vms[i] = New(ins[i], outs[i], n, k, cap int)
		}
		return vms, ins, outs
	}

	// runVMs until the done channel is closed.
	runVMs := func(done <-chan struct{}, vms []VM) {
		co.ParForAll(vms, func(i int) {
			vms[i].Run(done)
		})
	}

	Context("when running the virtual machines in a fully connected network", func() {

		table := []struct {
			n, k      uint
			bufferCap int
		}{
			{3, 2, BufferLimit}, {3, 2, BufferLimit * 2}, {3, 2, BufferLimit * 3}, {3, 2, BufferLimit * 4},
			{6, 4, BufferLimit}, {6, 4, BufferLimit * 2}, {6, 4, BufferLimit * 3}, {6, 4, BufferLimit * 4},
			{12, 8, BufferLimit}, {12, 8, BufferLimit * 2}, {12, 8, BufferLimit * 3}, {12, 8, BufferLimit * 4},
			{24, 16, BufferLimit}, {24, 16, BufferLimit * 2}, {24, 16, BufferLimit * 3}, {24, 16, BufferLimit * 4},
		}

		for _, entry := range table {
			entry := entry

			Context(fmt.Sprintf("when n = %v, k = %v and buffer capacity = %v", entry.n, entry.k, entry.bufferCap), func() {
				It("should add public numbers", func(doneT Done) {
				})

				It("should add private numbers", func(doneT Done) {
				})

				It("should add public numbers with private numbers", func(doneT Done) {
				})

				It("should generate private random numbers", func(doneT Done) {
				})

				It("should multiply private numbers", func(doneT Done) {
				})
			})
		}
	})

})
