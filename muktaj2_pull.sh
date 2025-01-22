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

# Command to run on each VM
populate_command="cd cs425 &&  git checkout main && git pull --rebase https://muktaj2:gitlab.engr.illinois.edu/muktaj2/cs425.git"

# Function to populate logs on a VM
populate_logs_on_vm() {
    local vm=$1
    echo "Running command on $vm..."
    ssh "muktaj2@$vm" "$populate_command"

    if [ $? -eq 0 ]; then
        echo "Git pull completed on $vm"
    else
        echo "Failed to do git pull on $vm"
    fi
}

# Loop through each VM and run the populate command
for vm in "${vms[@]}"; do
    populate_logs_on_vm "$vm" &
done

# Wait for all background jobs to complete
wait
echo "All git pulls are completed."
