# kvm-vm-tune

`kvm-vm-tune` is a CLI tool for managing KVM virtual machine resources, including CPU, memory, and disk.

## Features

- Change CPU count for a VM
- Modify memory size of a VM
- Expand disk size for a VM
- Dry-run option to preview commands without execution

## Requirements

- sudo privileges are required to run the commands.
- The following tools must be installed on your system:
  - virsh
  - qemu-img
  - virt-resize

## Installation

To install `kvm-vm-tune`, clone the repository and build the tool:

```bash
$ go build
```

## Usage

### Change CPU count

```
$ kvm-vm-tune cpu <cpu_count> <vm_name>
```

### Change memory size

```
$ kvm-vm-tune memory <memory_size> <vm_name>
```

### Expand disk

```
$ kvm-vm-tune disk <vm_name> --size <new_size> [--device <device>] [--partition <partition_number>] [--image <image_path>]
```

### Dry-run

Add the `--dry-run` flag to any command to preview the commands without executing them.

## Examples

1. Change CPU count to 4 for VM named "myvm":
   ```
   $ kvm-vm-tune cpu 4 myvm
   ```

2. Change memory size to 8G for VM named "myvm":
   ```
   $ kvm-vm-tune memory 8G myvm
   ```

3. Expand disk size to 40G for VM named "myvm":
   ```
   $ kvm-vm-tune disk myvm --size 40G
   ```

4. Dry-run disk expansion:
   ```
   $ kvm-vm-tune disk myvm --size 40G --dry-run
   ```

## License

This project is licensed under the MIT License - see the [LICENSE](https://opensource.org/license/mit) for details.
