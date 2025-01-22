# cs425 team 78 - Failure Detection



## Getting started
Clone the repository and install the required golang dependencies 

1. Run the process 
    - Go to the root folder 
    ```
    - go run main.go {serverId}
    - eg: go run main.go 5
    ```

## Run the Query in terminal 
1. Once main process in running. Enter you query in the terminal 
2. Available commands are:
```
list_mem, list_self, join, leave, enable_sus, disable_sus, status_sus, show_sus, exit
```

3. Join the process using join command

4. List the membership of node
    ```
    Enter command: list_mem
    Membership list:
    NodeId: 1 TimeStamp: 20:42:39, Status: ACTIVE,
    NodeId: 2 TimeStamp: 20:42:22, Status: ACTIVE,
    NodeId: 3 TimeStamp: 20:42:43, Status: ACTIVE,
    NodeId: 4 TimeStamp: 20:42:45, Status: ACTIVE,
    Enter command: 
    ```

5. Swtich to suspicion mode 
    ```
    Enter command: enable_sus
    Suspicion detection enabled.
    ```

6. Kill any process
    - Failed process will be detected by the nodes monitoring the process
    - Failure message is propogated using piggy back method
    - Membership list is updated and failed process is removed



## More Information


- We use SWIM Style failure detection mechaninms
- Each node sends PING/ACKs 
- System can switch to suspicion mode without downtime
- Messages/Singals are communicated over **UDP** 
- [SWIM Research paper](https://www.cs.cornell.edu/projects/Quicksilver/public_pdfs/SWIM.pdf#:~:text=failure%20detection%20time,%20stable%20rate%20of%20false%20positives%20and%20low)



