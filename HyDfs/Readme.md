# cs425 team 78 - HYDFS 



## Getting started
Clone the repository and install the required golang dependencies 

1. Run the process 
    - Go to the root folder 
    ```
    - go run main.go {serverId}
    - eg: go run main.go 5
    ```

## Run the Query in terminal 
1. Once main process in running. Follow "GroupMembership/Readme.md" for Joining the network. Enter you query in the terminal. 
2. Available commands are:
```
append, read, get, getrc, merge, store, ls, multiAppend
```

3. CREATE and GET
```
create {localFileName} {HydfsFileName}
eg : create business_1.txt hdfs-f1.txt 
read {HydfsFileName} {localFileName}
eg: read hdfs-f1.txt localFile.txt
```

4. APPEND and MULTIAPPEND
    ```
    append {localFileName} {HydfsFileName}
    eg: append sample2.txt hdfs-t1.txt
    multiappend {HydfsFileName} {f1,f2,f3} {vm1,vm2,vm3}

    multiappend hdfs-t1.txt sample2.txt,sample2.txt,sample2.txt 4,1,2
    ```

5. Initiate Merge append 
    ```
    merge {hydfsFileName}
    ```

6. STORE and ls
    ```
    - ls {hydfsFileName}
        List all Machines storing this filename
    - store
        List HydfsFiles present in this node

7. Above commands can be used in any of process in the network  

## More Information


- We Have built an filesystem with Hybrid architecture of HDFS and Cassandra

- Allows efficient concurrent Writes
- Uses internal Append Only Logs: https://github.com/arriqaaq/aol
- Compaction Algorithm runs periodically to flush AOL contents into FileSystem
- Maintains 3 replicas of each file
- Uses Underlying SWIM Style Failure Dectection Algorithm


