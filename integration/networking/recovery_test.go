package networking_test

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/cloudfoundry-incubator/garden"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Networking recovery", func() {
	Context("with two containers in the same subnet", func() {
		var (
			ctr1           garden.Container
			ctr2           garden.Container
			bridgeEvidence string
		)
		BeforeEach(func() {
			client = startGarden()

			containerNetwork := fmt.Sprintf("10.%d.0.0/24", GinkgoParallelNode())
			var err error
			ctr1, err = client.Create(garden.ContainerSpec{Network: containerNetwork})
			Ω(err).ShouldNot(HaveOccurred())
			ctr2, err = client.Create(garden.ContainerSpec{Network: containerNetwork})
			Ω(err).ShouldNot(HaveOccurred())

			bridgeEvidence = fmt.Sprintf("inet 10.%d.0.254/24 scope global w%db-", GinkgoParallelNode(), GinkgoParallelNode())
			cmd := exec.Command("ip", "a")
			Ω(cmd.CombinedOutput()).Should(ContainSubstring(bridgeEvidence))
		})

		Context("when garden is killed and restarted using SIGKILL", func() {
			BeforeEach(func() {
				gardenProcess.Signal(syscall.SIGKILL)
				Eventually(gardenProcess.Wait(), "10s").Should(Receive())

				client = startGarden()
				Ω(client.Ping()).ShouldNot(HaveOccurred())
			})

			It("the subnet's bridge no longer exists", func() {
				cmd := exec.Command("ip", "a")
				Ω(cmd.CombinedOutput()).ShouldNot(ContainSubstring(bridgeEvidence))
			})
		})

		Context("when garden is shut down cleanly and restarted, and the containers are deleted", func() {
			BeforeEach(func() {
				gardenProcess.Signal(syscall.SIGTERM)
				Eventually(gardenProcess.Wait(), "10s").Should(Receive())

				client = startGarden()
				Ω(client.Ping()).ShouldNot(HaveOccurred())

				cmd := exec.Command("ip", "a")
				Ω(cmd.CombinedOutput()).Should(ContainSubstring(bridgeEvidence))

				Ω(client.Destroy(ctr1.Handle())).Should(Succeed())
				Ω(client.Destroy(ctr2.Handle())).Should(Succeed())
			})

			It("the subnet's bridge no longer exists", func() {
				cmd := exec.Command("ip", "a")
				Ω(cmd.CombinedOutput()).ShouldNot(ContainSubstring(bridgeEvidence))
			})
		})

		Context("when garden is shut down and restarted", func() {
			BeforeEach(func() {
				gardenProcess.Signal(syscall.SIGTERM)
				Eventually(gardenProcess.Wait(), "10s").Should(Receive())

				client = startGarden()
				Ω(client.Ping()).ShouldNot(HaveOccurred())
			})

			It("the subnet's bridge still exists", func() {
				cmd := exec.Command("ip", "a")
				Ω(cmd.CombinedOutput()).Should(ContainSubstring(bridgeEvidence))
			})

			It("containers are still pingable", func() {
				info1, ierr := ctr1.Info()
				Ω(ierr).ShouldNot(HaveOccurred())

				out, err := exec.Command("/bin/ping", "-c 2", info1.ContainerIP).Output()
				Ω(out).Should(ContainSubstring(" 0% packet loss"))
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("a container can still reach external networks", func() {
				Ω(checkInternet(ctr1)).Should(Succeed())
			})
		})
	})

})
