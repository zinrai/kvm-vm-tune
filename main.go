package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zinrai/libvirtwrap-go/pkg/disk"
	"github.com/zinrai/libvirtwrap-go/pkg/vm"
	"github.com/zinrai/libvirtwrap-go/pkg/virsh"
)

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

	myVM := vm.New(vmName)

	if dryRun {
		fmt.Printf("Would set CPU count to %d for VM '%s'\n", cpuCount, vmName)
		return
	}

	if err := myVM.SetCPUCount(cpuCount); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change CPU count: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("CPU count changed to %d for VM '%s'.\n", cpuCount, vmName)
}

func runMemoryCommand(cmd *cobra.Command, args []string) {
	memorySize := args[0]
	vmName := args[1]

	myVM := vm.New(vmName)

	if dryRun {
		fmt.Printf("Would set memory size to %s for VM '%s'\n", memorySize, vmName)
		return
	}

	if err := myVM.SetMemorySize(memorySize); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change memory size: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Memory size changed to %s for VM '%s'.\n", memorySize, vmName)
}

func runDiskCommand(cmd *cobra.Command, args []string) {
	vmName := args[0]

	myVM := vm.New(vmName)

	if imagePath == "" {
		disks, err := virsh.GetVMDiskPaths(vmName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get disk information for VM '%s': %v\n", vmName, err)
			os.Exit(1)
		}
		if len(disks) == 0 {
			fmt.Fprintf(os.Stderr, "No disks found for VM '%s'\n", vmName)
			os.Exit(1)
		}
		imagePath = disks[0]
	}

	fmt.Printf("Selected disk: %s (device: %s)\n", imagePath, device)

	isRunning, err := myVM.IsRunning()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check VM status: %v\n", err)
		os.Exit(1)
	}
	if isRunning {
		fmt.Fprintf(os.Stderr, "VM '%s' is currently running. Please stop the VM before making changes.\n", vmName)
		os.Exit(1)
	}

	if size == "" {
		fmt.Fprintf(os.Stderr, "Please specify the new size using the --size option\n")
		os.Exit(1)
	}

	if dryRun {
		fmt.Printf("Would resize disk %s to %s for VM '%s'\n", imagePath, size, vmName)
		return
	}

	belongsToVM, err := myVM.VerifyDiskBelongsToVM(imagePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to verify disk ownership: %v\n", err)
		os.Exit(1)
	}
	if !belongsToVM {
		fmt.Fprintf(os.Stderr, "The specified disk does not belong to VM '%s'\n", vmName)
		os.Exit(1)
	}

	if err := disk.ResizeAndExpandDisk(imagePath, device, partition, size); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to resize and expand disk: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Disk expansion completed successfully for VM '%s'.\n", vmName)
}
