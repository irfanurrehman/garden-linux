package containerizer_test

import (
	"errors"
	"os"

	"github.com/cloudfoundry-incubator/garden-linux/containerizer"
	"github.com/cloudfoundry-incubator/garden-linux/containerizer/fake_container_configurer"
	"github.com/cloudfoundry-incubator/garden-linux/containerizer/fake_container_execer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Containerizer", func() {
	Describe("Create", func() {
		var cz *containerizer.Containerizer
		var containerExecer *fake_container_execer.FakeContainerExecer

		BeforeEach(func() {
			containerExecer = &fake_container_execer.FakeContainerExecer{}

			cz = &containerizer.Containerizer{
				Execer:      containerExecer,
				InitBinPath: "initd",
			}
		})

		It("Runs the initd process in a container", func() {
			Ω(cz.Create()).Should(Succeed())
			Ω(containerExecer.ExecCallCount()).Should(Equal(1))
			binPath, args := containerExecer.ExecArgsForCall(0)
			Ω(binPath).Should(Equal("initd"))
			Ω(args).Should(BeEmpty())
		})

		PIt("exports PID environment variable", func() {})

		Context("when execer fails", func() {
			It("returns an error", func() {
				containerExecer.ExecReturns(0, errors.New("Oh my gawsh"))
				Ω(cz.Create()).Should(MatchError("containerizer: Failed to create container: Oh my gawsh"))
			})
		})
	})

	Describe("Child", func() {
		var cz *containerizer.Containerizer
		var containerConfigurer *fake_container_configurer.FakeContainerConfigurer
		var workingDirectory string

		BeforeEach(func() {
			var err error

			workingDirectory, err = os.Getwd()
			Ω(err).ShouldNot(HaveOccurred())

			containerConfigurer = &fake_container_configurer.FakeContainerConfigurer{}

			cz = &containerizer.Containerizer{
				Configurer: containerConfigurer,
				RootFSPath: "/tmp/rootfs",
				ChdirPath:  "/",
			}
		})

		AfterEach(func() {
			Ω(os.Chdir(workingDirectory)).Should(Succeed())
		})

		It("bind mounts a rootfs", func() {
			Ω(cz.Child()).Should(Succeed())
			Ω(containerConfigurer.BindMountRootfsCallCount()).Should(Equal(1))
			args := containerConfigurer.BindMountRootfsArgsForCall(0)
			Ω(args).Should(Equal("/tmp/rootfs"))
		})

		It("pivots root", func() {
			Ω(cz.Child()).Should(Succeed())
			Ω(containerConfigurer.PivotRootCallCount()).Should(Equal(1))
			args := containerConfigurer.PivotRootArgsForCall(0)
			Ω(args).Should(Equal("/tmp/rootfs"))
		})

		It("changes user and group to root", func() {
			Ω(cz.Child()).Should(Succeed())
			Ω(containerConfigurer.ChangeUserCallCount()).Should(Equal(1))
			uid, gid := containerConfigurer.ChangeUserArgsForCall(0)
			Ω(uid).Should(Equal(0))
			Ω(gid).Should(Equal(0))
		})

		It("changes directory to the new root", func() {
			wd, err := os.Getwd()
			Ω(err).ShouldNot(HaveOccurred())

			cz.Child()

			newWd, err := os.Getwd()
			Ω(err).ShouldNot(HaveOccurred())

			Ω(newWd).ShouldNot(Equal(wd))
			Ω(newWd).Should(Equal("/"))
		})

		Context("when bind mount fails", func() {
			BeforeEach(func() {
				containerConfigurer.BindMountRootfsReturns(errors.New("Opps"))
			})

			It("returns an error", func() {
				Ω(cz.Child()).Should(MatchError("containerizer: Failed to bind mount rootfs: Opps"))
			})

			It("does not run pivot root", func() {
				cz.Child()
				Ω(containerConfigurer.PivotRootCallCount()).Should(Equal(0))
			})
		})

		Context("when pivot root fails", func() {
			BeforeEach(func() {
				containerConfigurer.PivotRootReturns(errors.New("Opps"))
			})

			It("returns an error", func() {
				Ω(cz.Child()).Should(MatchError("containerizer: Failed to pivot root: Opps"))
			})
		})

		Context("when change user/groups fails", func() {
			BeforeEach(func() {
				containerConfigurer.ChangeUserReturns(errors.New("Opps"))
			})

			It("returns an error", func() {
				Ω(cz.Child()).Should(MatchError("containerizer: Failed to change user: Opps"))
			})
		})

		Context("when changing directory fails", func() {
			It("returns an error", func() {
				cz = &containerizer.Containerizer{
					Configurer: containerConfigurer,
					RootFSPath: "/tmp/rootfs",
					ChdirPath:  "/a/long/non/existant/path",
				}
				err := cz.Child()
				Ω(err.Error()).Should(HavePrefix(("containerizer: Failed to change directory after pivot root:")))
			})
		})
	})
})
