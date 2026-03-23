package main

import (
	"fmt"
	"os/exec"
)

func setCPUCount(vmName string, count int) error {
	cmd := exec.Command("sudo", "virsh", "setvcpus", vmName, fmt.Sprintf("%d", count), "--config", "--maximum")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set maximum CPU count: %v", err)
	}

	cmd = exec.Command("sudo", "virsh", "setvcpus", vmName, fmt.Sprintf("%d", count), "--config")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set current CPU count: %v", err)
	}

	return nil
}
