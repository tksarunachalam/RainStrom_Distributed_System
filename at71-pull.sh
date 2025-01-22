#!/bin/bash

for vm in 172.22.95.2 172.22.157.3 172.22.159.3 172.22.95.3 172.22.157.4 172.22.159.4 172.22.95.4 172.22.157.5 172.22.159.5 172.22.95.5  # Loop through all 10 VMs
do
    ssh at71@$vm "cd MP1-G78 && git pull origin main"  # SSH into each VM, navigate to the repository, and run 'git pull'
done 

git config --global url."https://oauth2:@gitlab.engr.illinois.edu".insteadOf https://gitlab.engr.illinois.edu

# https://gitlab.engr.illinois.edu/
