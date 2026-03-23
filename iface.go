package main

import (
	"fmt"
	"os/exec"
)

func attachInterface(vmName, ifType, source, model string) error {
	args := []string{"attach-interface", vmName, "--type", ifType}

	if source != "" {
		args = append(args, "--source", source)
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	args = append(args, "--config")

	cmd := exec.Command("sudo", append([]string{"virsh"}, args...)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to attach interface: %v", err)
	}

	return nil
}
