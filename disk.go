package main

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
)

type domainDisk struct {
	Device string `xml:"device,attr"`
	Source struct {
		File string `xml:"file,attr"`
	} `xml:"source"`
	Target struct {
		Dev string `xml:"dev,attr"`
	} `xml:"target"`
}

type domain struct {
	Devices struct {
		Disks []domainDisk `xml:"disk"`
	} `xml:"devices"`
}

func getVMDiskPaths(vmName string) ([]string, error) {
	cmd := exec.Command("sudo", "virsh", "dumpxml", vmName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute virsh dumpxml: %v", err)
	}

	var d domain
	if err := xml.Unmarshal(output, &d); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	var diskPaths []string
	for _, disk := range d.Devices.Disks {
		if disk.Device == "disk" {
			diskPaths = append(diskPaths, disk.Source.File)
		}
	}

	if len(diskPaths) == 0 {
		return nil, fmt.Errorf("no disks found for VM '%s'", vmName)
	}

	return diskPaths, nil
}

func isVMRunning(vmName string) (bool, error) {
	cmd := exec.Command("sudo", "virsh", "list", "--name", "--state-running")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to get list of running VMs: %v", err)
	}

	runningVMs := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, vm := range runningVMs {
		if vm == vmName {
			return true, nil
		}
	}
	return false, nil
}

func verifyDiskBelongsToVM(vmName, imagePath string) (bool, error) {
	diskPaths, err := getVMDiskPaths(vmName)
	if err != nil {
		return false, fmt.Errorf("failed to get disk paths for VM '%s': %v", vmName, err)
	}

	for _, path := range diskPaths {
		if path == imagePath {
			return true, nil
		}
	}

	return false, nil
}

func resizeAndExpandDisk(imagePath, device string, partition int, newSize string) error {
	newImagePath := imagePath + ".new"

	cmd := exec.Command("sudo", "qemu-img", "create", "-f", "qcow2", "-o", "preallocation=metadata", newImagePath, newSize)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create new image: %v", err)
	}

	cmd = exec.Command("sudo", "virt-resize", "--expand", fmt.Sprintf("/dev/%s%d", device, partition), imagePath, newImagePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to resize disk: %v", err)
	}

	cmd = exec.Command("sudo", "mv", newImagePath, imagePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to replace original image: %v", err)
	}

	return nil
}

func createDiskImage(path, size, format string) error {
	if format == "" {
		format = "qcow2"
	}

	cmd := exec.Command("sudo", "qemu-img", "create", "-f", format, path, size)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create disk image: %v", err)
	}

	return nil
}

func attachDisk(vmName, diskPath, targetDev string, options map[string]string) error {
	if options == nil {
		options = make(map[string]string)
	}

	if _, ok := options["cache"]; !ok {
		options["cache"] = "none"
	}

	if _, ok := options["driver"]; !ok {
		options["driver"] = "qemu"
	}

	if _, ok := options["subdriver"]; !ok {
		options["subdriver"] = "qcow2"
	}

	args := []string{"attach-disk", vmName, diskPath, targetDev}

	if cache, ok := options["cache"]; ok {
		args = append(args, "--cache", cache)
	}

	if driver, ok := options["driver"]; ok {
		args = append(args, "--driver", driver)
	}

	if subdriver, ok := options["subdriver"]; ok {
		args = append(args, "--subdriver", subdriver)
	}

	args = append(args, "--current")

	cmd := exec.Command("sudo", append([]string{"virsh"}, args...)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to attach disk: %v", err)
	}

	return nil
}
