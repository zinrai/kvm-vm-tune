package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type DomainDisk struct {
	Device string `xml:"device,attr"`
	Source struct {
		File string `xml:"file,attr"`
	} `xml:"source"`
	Target struct {
		Dev string `xml:"dev,attr"`
	} `xml:"target"`
}

type Domain struct {
	Devices struct {
		Disks []DomainDisk `xml:"disk"`
	} `xml:"devices"`
}

var rootCmd = &cobra.Command{
	Use:   "kvm-vm-tune",
	Short: "KVM VM Resource Management CLI Tool",
	Long:  `A CLI tool for managing KVM virtual machine resources including CPU, memory, and disk.`,
}

var cpuCmd = &cobra.Command{
	Use:   "cpu <cpu_count> <vm_name>",
	Short: "Change CPU count for a VM",
	Args:  cobra.ExactArgs(2),
	Run:   runCPUCommand,
}

var memoryCmd = &cobra.Command{
	Use:   "memory <memory_size> <vm_name>",
	Short: "Change memory size for a VM",
	Args:  cobra.ExactArgs(2),
	Run:   runMemoryCommand,
}

var diskCmd = &cobra.Command{
	Use:   "disk <vm_name>",
	Short: "Expand disk for a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runDiskCommand,
}

var (
	imagePath string
	device    string
	partition int
	size      string
	dryRun    bool
)

func init() {
	rootCmd.AddCommand(cpuCmd, memoryCmd, diskCmd)

	diskCmd.Flags().StringVar(&imagePath, "image", "", "Path to the virtual machine image file")
	diskCmd.Flags().StringVar(&device, "device", "vda", "Disk device (e.g., vda, sda)")
	diskCmd.Flags().IntVar(&partition, "partition", 1, "Partition number to expand")
	diskCmd.Flags().StringVar(&size, "size", "", "New size for the disk (e.g., 40G)")
	diskCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the command without executing it")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runCPUCommand(cmd *cobra.Command, args []string) {
	cpuCount, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid CPU count: %v\n", err)
		os.Exit(1)
	}
	vmName := args[1]

	if dryRun {
		printCommand("virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpuCount), "--config", "--maximum")
		printCommand("virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpuCount), "--config")
		return
	}

	if err := changeCPUCount(vmName, cpuCount); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change CPU count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("CPU count changed to %d for VM '%s'.\n", cpuCount, vmName)
}

func runMemoryCommand(cmd *cobra.Command, args []string) {
	memorySize := args[0]
	vmName := args[1]

	if dryRun {
		printCommand("virsh", "setmaxmem", vmName, memorySize, "--config")
		printCommand("virsh", "setmem", vmName, memorySize, "--config")
		return
	}

	if err := changeMemorySize(vmName, memorySize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change memory size: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Memory size changed to %s for VM '%s'.\n", memorySize, vmName)
}

func runDiskCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if imagePath == "" {
		disks, err := getVMDisks(vmName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get disk information for VM '%s': %v\n", vmName, err)
			os.Exit(1)
		}
		if len(disks) == 0 {
			fmt.Fprintf(os.Stderr, "No disks found for VM '%s'\n", vmName)
			os.Exit(1)
		}
		imagePath = disks[0].Source.File
	}

	fmt.Printf("Selected disk: %s (device: %s)\n", imagePath, device)

	if !isVMStopped(vmName) {
		fmt.Fprintf(os.Stderr, "VM '%s' is currently running. Please stop the VM before making changes.\n", vmName)
		os.Exit(1)
	}

	if size == "" {
		fmt.Fprintf(os.Stderr, "Please specify the new size using the --size option\n")
		os.Exit(1)
	}

	if dryRun {
		printResizeCommand(imagePath, device, partition, size)
		return
	}

	fmt.Println("WARNING: This operation will modify the disk image.")
	fmt.Print("Do you want to continue? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.ToLower(strings.TrimSpace(response))

	if response != "y" && response != "yes" {
		fmt.Println("Operation cancelled.")
		os.Exit(0)
	}

	if err := resizeAndExpandDisk(imagePath, device, partition, size); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resize and expand disk: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Disk expansion completed successfully.")
}

func getVMDisks(vmName string) ([]DomainDisk, error) {
	cmd := exec.Command("sudo", "virsh", "dumpxml", vmName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute virsh dumpxml: %v", err)
	}

	var domain Domain
	if err := xml.Unmarshal(output, &domain); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	if len(domain.Devices.Disks) == 0 {
		return nil, fmt.Errorf("no disks found for VM '%s'", vmName)
	}

	return domain.Devices.Disks, nil
}

func isVMStopped(vmName string) bool {
	cmd := exec.Command("sudo", "virsh", "list", "--name", "--state-running")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to get list of running VMs: %v\n", err)
		return false
	}

	runningVMs := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, vm := range runningVMs {
		if vm == vmName {
			return false
		}
	}
	return true
}

func resizeAndExpandDisk(imagePath, device string, partition int, newSize string) error {
    newImagePath := imagePath + ".new"

    createCmd := exec.Command("sudo", "qemu-img", "create", "-f", "qcow2", "-o", "preallocation=metadata", newImagePath, newSize)
    fmt.Println("Creating new image...")
    fmt.Println("Executing command:", createCmd.String())
    output, err := createCmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create new image: %v, output: %s", err, string(output))
    }

    resizeCmd := exec.Command("sudo", "virt-resize", "--expand", fmt.Sprintf("/dev/%s%d", device, partition), imagePath, newImagePath)
    fmt.Println("Resizing disk...")
    fmt.Println("Executing command:", resizeCmd.String())
    output, err = resizeCmd.CombinedOutput()
    if err != nil {
        rmCmd := exec.Command("sudo", "rm", newImagePath)
        rmOutput, rmErr := rmCmd.CombinedOutput()
        if rmErr != nil {
            return fmt.Errorf("virt-resize failed: %v, output: %s. Additionally, failed to remove new image: %v, output: %s",
                err, string(output), rmErr, string(rmOutput))
        }
        return fmt.Errorf("virt-resize failed: %v, output: %s. New image was removed.", err, string(output))
    }

    fmt.Printf("virt-resize output:\n%s\n", string(output))

    mvCmd := exec.Command("sudo", "mv", newImagePath, imagePath)
    fmt.Println("Replacing original image with resized image...")
    fmt.Println("Executing command:", mvCmd.String())
    if err := mvCmd.Run(); err != nil {
        rmCmd := exec.Command("sudo", "rm", newImagePath)
        rmOutput, rmErr := rmCmd.CombinedOutput()
        if rmErr != nil {
            return fmt.Errorf("failed to replace original image: %v. Additionally, failed to remove new image: %v, output: %s",
                err, rmErr, string(rmOutput))
        }
        return fmt.Errorf("failed to replace original image: %v. New image was removed.", err)
    }

    fmt.Println("Disk expansion completed.")
    return nil
}

func changeCPUCount(vmName string, cpuCount int) error {
	cmdMax := exec.Command("sudo", "virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpuCount), "--config", "--maximum")
	if err := cmdMax.Run(); err != nil {
		return fmt.Errorf("failed to set maximum CPU count: %v", err)
	}

	cmdCurrent := exec.Command("sudo", "virsh", "setvcpus", vmName, fmt.Sprintf("%d", cpuCount), "--config")
	if err := cmdCurrent.Run(); err != nil {
		return fmt.Errorf("failed to set current CPU count: %v", err)
	}

	return nil
}

func changeMemorySize(vmName, memorySize string) error {
	cmd := exec.Command("sudo", "virsh", "setmaxmem", vmName, memorySize, "--config")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set maximum memory: %v, output: %s", err, string(output))
	}

	cmd = exec.Command("sudo", "virsh", "setmem", vmName, memorySize, "--config")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set current memory: %v, output: %s", err, string(output))
	}

	return nil
}

func printCommand(name string, args ...string) {
	fmt.Printf("Command: sudo %s %s\n", name, strings.Join(args, " "))
}

func printResizeCommand(imagePath, device string, partition int, newSize string) {
	newImagePath := imagePath + ".new"

	fmt.Printf("Command: sudo qemu-img create -f qcow2 -o preallocation=metadata %s %s\n", newImagePath, newSize)
	fmt.Printf("Command: sudo virt-resize --expand /dev/%s%d %s %s\n", device, partition, imagePath, newImagePath)
	fmt.Printf("Command: sudo mv %s %s\n", newImagePath, imagePath)
}
