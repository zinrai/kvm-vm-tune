package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kvm-vm-tune",
	Short: "KVM VM Resource Management CLI Tool",
	Long:  `A CLI tool for managing KVM virtual machine resources including CPU, memory, and disk.`,
}

var cpuCmd = &cobra.Command{
	Use:   "cpu <vm_name>",
	Short: "Change CPU count for a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runCPUCommand,
}

var memoryCmd = &cobra.Command{
	Use:   "memory <vm_name>",
	Short: "Change memory size for a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runMemoryCommand,
}

var expandDiskCmd = &cobra.Command{
	Use:   "expand-disk <vm_name>",
	Short: "Expand disk for a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runExpandDiskCommand,
}

var attachDiskCmd = &cobra.Command{
	Use:   "attach-disk <vm_name>",
	Short: "Attach a disk to a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runAttachDiskCommand,
}

var createDiskCmd = &cobra.Command{
	Use:   "create-disk",
	Short: "Create a new disk image",
	Run:   runCreateDiskCommand,
}

var attachIfaceCmd = &cobra.Command{
	Use:   "attach-iface <vm_name>",
	Short: "Attach a network interface to a VM",
	Args:  cobra.ExactArgs(1),
	Run:   runAttachIfaceCommand,
}

var (
	// Common flags
	dryRun bool

	// CPU flags
	cpuCount int

	// Memory flags
	memorySize string

	// Expand disk flags
	expandImagePath string
	expandDevice    string
	expandPartition int
	expandSize      string

	// Attach disk flags
	attachDiskPath  string
	attachTarget    string
	attachCache     string
	attachDriver    string
	attachSubdriver string

	// Create disk flags
	createDiskPath string
	createSize     string
	createFormat   string

	// Attach interface flags
	ifaceType   string
	ifaceSource string
	ifaceModel  string
)

func init() {
	rootCmd.AddCommand(cpuCmd, memoryCmd, expandDiskCmd, attachDiskCmd, createDiskCmd, attachIfaceCmd)

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run:   func(cmd *cobra.Command, args []string) { printVersion() },
	})

	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Print the command without executing it")

	// CPU flags
	cpuCmd.Flags().IntVar(&cpuCount, "count", 0, "Number of CPUs to set")
	cpuCmd.MarkFlagRequired("count")

	// Memory flags
	memoryCmd.Flags().StringVar(&memorySize, "size", "", "Memory size to set (e.g., 4G)")
	memoryCmd.MarkFlagRequired("size")

	// Expand disk flags
	expandDiskCmd.Flags().StringVar(&expandImagePath, "image", "", "Path to the virtual machine image file")
	expandDiskCmd.Flags().StringVar(&expandDevice, "device", "vda", "Disk device (e.g., vda, sda)")
	expandDiskCmd.Flags().IntVar(&expandPartition, "partition", 1, "Partition number to expand")
	expandDiskCmd.Flags().StringVar(&expandSize, "size", "", "New size for the disk (e.g., 40G)")
	expandDiskCmd.MarkFlagRequired("size")

	// Attach disk flags
	attachDiskCmd.Flags().StringVar(&attachDiskPath, "disk-path", "", "Path to the disk image to attach")
	attachDiskCmd.Flags().StringVar(&attachTarget, "target", "", "Target device name (e.g., vdb)")
	attachDiskCmd.Flags().StringVar(&attachCache, "cache", "none", "Cache mode")
	attachDiskCmd.Flags().StringVar(&attachDriver, "driver", "qemu", "Driver type")
	attachDiskCmd.Flags().StringVar(&attachSubdriver, "subdriver", "qcow2", "Subdriver type")
	attachDiskCmd.MarkFlagRequired("disk-path")
	attachDiskCmd.MarkFlagRequired("target")

	// Create disk flags
	createDiskCmd.Flags().StringVar(&createDiskPath, "path", "", "Path where the new disk image will be created")
	createDiskCmd.Flags().StringVar(&createSize, "size", "", "Size of the new disk (e.g., 10G)")
	createDiskCmd.Flags().StringVar(&createFormat, "format", "qcow2", "Format of the disk image (e.g., qcow2, raw)")
	createDiskCmd.MarkFlagRequired("path")
	createDiskCmd.MarkFlagRequired("size")

	// Attach interface flags
	attachIfaceCmd.Flags().StringVar(&ifaceType, "type", "", "Interface type (e.g., bridge, network)")
	attachIfaceCmd.Flags().StringVar(&ifaceSource, "source", "", "Source name (e.g., br0, default)")
	attachIfaceCmd.Flags().StringVar(&ifaceModel, "model", "virtio", "Interface model (e.g., virtio)")
	attachIfaceCmd.MarkFlagRequired("type")
	attachIfaceCmd.MarkFlagRequired("source")
}

func runCPUCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if dryRun {
		fmt.Printf("Would set CPU count to %d for VM '%s'\n", cpuCount, vmName)
		return
	}

	if err := setCPUCount(vmName, cpuCount); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change CPU count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("CPU count changed to %d for VM '%s'.\n", cpuCount, vmName)
}

func runMemoryCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if dryRun {
		fmt.Printf("Would set memory size to %s for VM '%s'\n", memorySize, vmName)
		return
	}

	if err := setMemorySize(vmName, memorySize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change memory size: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Memory size changed to %s for VM '%s'.\n", memorySize, vmName)
}

func runExpandDiskCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if expandImagePath == "" {
		disks, err := getVMDiskPaths(vmName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get disk information for VM '%s': %v\n", vmName, err)
			os.Exit(1)
		}
		if len(disks) == 0 {
			fmt.Fprintf(os.Stderr, "No disks found for VM '%s'\n", vmName)
			os.Exit(1)
		}
		expandImagePath = disks[0]
	}

	fmt.Printf("Selected disk: %s (device: %s)\n", expandImagePath, expandDevice)

	isRunning, err := isVMRunning(vmName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check VM status: %v\n", err)
		os.Exit(1)
	}
	if isRunning {
		fmt.Fprintf(os.Stderr, "VM '%s' is currently running. Please stop the VM before making changes.\n", vmName)
		os.Exit(1)
	}

	if dryRun {
		fmt.Printf("Would resize disk %s to %s for VM '%s'\n", expandImagePath, expandSize, vmName)
		return
	}

	belongsToVM, err := verifyDiskBelongsToVM(vmName, expandImagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to verify disk ownership: %v\n", err)
		os.Exit(1)
	}
	if !belongsToVM {
		fmt.Fprintf(os.Stderr, "The specified disk does not belong to VM '%s'\n", vmName)
		os.Exit(1)
	}

	if err := resizeAndExpandDisk(expandImagePath, expandDevice, expandPartition, expandSize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resize and expand disk: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Disk expansion completed successfully for VM '%s'.\n", vmName)
}

func runAttachDiskCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if dryRun {
		fmt.Printf("Would attach disk %s to VM '%s' as %s\n", attachDiskPath, vmName, attachTarget)
		return
	}

	if _, err := os.Stat(attachDiskPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Disk image does not exist: %s\n", attachDiskPath)
		os.Exit(1)
	}

	options := map[string]string{
		"cache":     attachCache,
		"driver":    attachDriver,
		"subdriver": attachSubdriver,
	}

	if err := attachDisk(vmName, attachDiskPath, attachTarget, options); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach disk: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Disk %s successfully attached to VM '%s' as %s\n", attachDiskPath, vmName, attachTarget)
}

func runCreateDiskCommand(cmd *cobra.Command, args []string) {
	if dryRun {
		fmt.Printf("Would create a new %s disk image at %s with size %s\n", createFormat, createDiskPath, createSize)
		return
	}

	if err := createDiskImage(createDiskPath, createSize, createFormat); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create disk image: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Disk image created successfully at %s with size %s\n", createDiskPath, createSize)
}

func runAttachIfaceCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	if dryRun {
		fmt.Printf("Would attach a %s interface to VM '%s' using source '%s' and model '%s'\n",
			ifaceType, vmName, ifaceSource, ifaceModel)
		return
	}

	if err := attachInterface(vmName, ifaceType, ifaceSource, ifaceModel); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to attach network interface: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Network interface (%s) successfully attached to VM '%s'\n", ifaceType, vmName)
}
