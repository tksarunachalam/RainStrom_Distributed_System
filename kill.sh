#!/bin/bash

# Array of VMs
vms=(
    "fa24-cs425-7801.cs.illinois.edu"
    "fa24-cs425-7802.cs.illinois.edu"
    "fa24-cs425-7803.cs.illinois.edu"
    "fa24-cs425-7804.cs.illinois.edu"
    "fa24-cs425-7805.cs.illinois.edu"
    "fa24-cs425-7806.cs.illinois.edu"
    "fa24-cs425-7807.cs.illinois.edu"
    "fa24-cs425-7808.cs.illinois.edu"
    "fa24-cs425-7809.cs.illinois.edu"
    "fa24-cs425-7810.cs.illinois.edu"
)

# Command to kill processes on each VM
kill_command="pkill -f 'your_process_name_here'"

# Function to kill processes on a VM
kill_process_on_vm() {
    local vm=$1
    echo "Killing processes on $vm..."
    ssh "muktaj2@$vm" "$kill_command"

    if [ $? -eq 0 ]; then
        echo "Processes killed successfully on $vm"
    else
        echo "Failed to kill processes on $vm"
    fi
}

# Check if arguments are provided
if [ $# -eq 0 ]; then
    echo "Usage: ./kill.sh <vm_indices>"
    echo "Example: ./kill.sh 1 2"
    exit 1
fi

# Loop through the provided VM indices and run the kill command
for index in "$@"; do
    # Convert the argument to zero-based index
    vm_index=$((index - 1))

    # Check if the index is valid
    if [ $vm_index -ge 0 ] && [ $vm_index -lt ${#vms[@]} ]; then
        kill_process_on_vm "${vms[$vm_index]}" &
    else
        echo "Invalid VM index: $index"
    fi
done

# Wait for all background jobs to complete
wait
echo "Kill commands executed on specified VMs."
