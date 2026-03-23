package main

import (
	"fmt"
	"os/exec"
)

func setMemorySize(vmName, size string) error {
	cmd := exec.Command("sudo", "virsh", "setmaxmem", vmName, size, "--config")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set maximum memory: %v", err)
	}

	cmd = exec.Command("sudo", "virsh", "setmem", vmName, size, "--config")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set current memory: %v", err)
	}

	return nil
}
