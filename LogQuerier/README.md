# cs425 team 78 - DISTRIBUTED GREP



## Getting started
Clone the repository and install the required golang dependencies 

1. Install Golang 
2. To generate the ProtoBuf models (only required when the i/p model changes)
    - Install Go gRPC Library:
    - Install the Protocol Buffers Plugin for Go:
        ```
        - go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        - go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        ```
    - To generate the models from .Proto file run 
        ```
        - protoc --go_out=. --go-grpc_out=. path/to/your/protos/greeter.proto
        ```

## Running the Query 
1. client/servers.conf has list of all server's address and the port exposed 
2. Start the server
    - Go to the 'server' folder 
    ```
    - go run server.go {serverId}
    - eg: go run server.go server1
    ```
3. Run the query through the Client 
    - Go to the 'client' folder
    - In the servers.conf file add the server configurations 
    ```
    localhost:8080
    localhost:8082
    ```
    - Run the client 
    ```
    - go run client.go {pattern_to_be_matched} {flags supported by grep}
    ```
4. Sample Output
    ```
    tks@TKSs-MBP client % go run client.go CS          

    Response from server1:
    CS425 Distributed systems
    CS425 at line no 3

    Response from server2:
    CS425
    WARN 101057-09-05 18:57:32 Server connection failedCS425
    INFO 101057-09-05 18:57:32 Server is disconnectedCS425
    INFO 101057-09-05 18:57:32 New client connectedCS425
    INFO 101057-09-05 18:57:32 Server connection failedCS425

    No of matching lines in server 1 : 2

    No of matching lines in server 2 : 5

    total matched lines: 7
    ```




&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;




## More Information


- [ ] The distributed grep query will run on all the N machines including the one the client runs
- Output from all the N nodes are displayed on the client terminal 
- The system is fault taulrent. If any of the node is down, the client collects the output from other live machines
- The client support flags supported by Grep such as -E, -n, -c
- The communication between the nodes happen through **Remote Procedure Call** 
- The implementation leverages gRPC - [High performance open source framework](https://grpc.io/)



