
# RainStorm on HYDFS

A Streaming System that uses distributed file system supporting efficient concurrent writes, fault tolerance, and exactly-once semantics for distributed applications.

---

## Getting Started

Clone the repository and install the required Go dependencies.

1. **Run the RainStorm Process**  
   Navigate to the root folder and run the following command:
   ```bash
   go run main.go {serverId}
   # Example:
   go run main.go 5
   ```

2. **Join the Network**  
   Follow the instructions in `GroupMembership/Readme.md` to join the network.

---

## Running Queries

Once the main process is running, enter your query in the terminal. The available commands are:

- `append`, `read`, `get`, `getrc`, `merge`, `store`, `ls`, `multiAppend`

### Commands
#### Create and Read
   Follow the instructions in `HyDfs/Readme.md` for more instructions on how to use these commands.

---

## RainStorm Invocation

RainStorm is invoked as:
```bash
rainstorm <op1_exe> <op2_exe> <hydfs_src_file> <hydfs_dest_filename> <num_tasks>
```

- `num_tasks`: For all tests, use `num_tasks=2` (Tested on 7 VM's due to CPU utilization error issue).
- `hydfs_src_file`: Choose one of `test_10000.csv`, `test_5000.csv`, or `test_1000.csv`.

---

## Features

- **Hybrid Architecture**: Combines HDFS and Cassandra for scalability and reliability.
- **Efficient Concurrent Writes**: Internal Append-Only Logs (AOL) with periodic compaction.
- **Failure Detection**: Uses SWIM-style failure detection.
- **Replication**: Maintains 3 replicas of each file.

---

## References
- Internal Append Only Logs: [arriqaaq/aol](https://github.com/arriqaaq/aol)
