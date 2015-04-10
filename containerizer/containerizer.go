package containerizer

import (
	"fmt"
	"os"
)

//go:generate counterfeiter -o fake_container_execer/FakeContainerExecer.go . ContainerExecer
type ContainerExecer interface {
	Exec(binPath string, args ...string) (int, error)
}

//go:generate counterfeiter -o fake_container_configurer/FakeContainerConfigurer.go . ContainerConfigurer
type ContainerConfigurer interface {
	BindMountRootfs(rootFSPath string) error
	PivotRoot(rootFSPath string) error
	ChangeUser(uid, gid int) error
}

type InitDaemon interface {
	Start() error
}

type Containerizer struct {
	InitBinPath string
	Execer      ContainerExecer
	RootFSPath  string
	ChdirPath   string
	Configurer  ContainerConfigurer
}

func (c *Containerizer) Create() error {
	_, err := c.Execer.Exec(c.InitBinPath)
	if err != nil {
		return fmt.Errorf("containerizer: Failed to create container: %s", err)
	}

	return nil
}

func (c *Containerizer) Child() error {
	if err := c.Configurer.BindMountRootfs(c.RootFSPath); err != nil {
		return fmt.Errorf("containerizer: Failed to bind mount rootfs: %s", err)
	}

	if err := c.Configurer.PivotRoot(c.RootFSPath); err != nil {
		return fmt.Errorf("containerizer: Failed to pivot root: %s", err)
	}

	if err := os.Chdir(c.ChdirPath); err != nil {
		return fmt.Errorf("containerizer: Failed to change directory after pivot root: %s", err)
	}

	// TODO: TTY stuff (ptmx)

	if err := c.Configurer.ChangeUser(0, 0); err != nil {
		return fmt.Errorf("containerizer: Failed to change user: %s", err)
	}

	// TODO: Call child-after-pivot hook scripts

	// TODO: Unmount old root

	// TODO: Barrier(s) for synchronization with tha parent

	// TODO: Run the daemon

	return nil
}
