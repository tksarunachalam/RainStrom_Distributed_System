package cli

import (
	"bufio"
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	hydfs "cs425/HyDfs"
	"cs425/HyDfs/models"
	"cs425/HyDfs/utils"
	stream "cs425/Stream"
	m "cs425/Stream/models"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func CliUtility(serverId int, ms *groupMembership.MembershipService,
	hs *hydfs.HyDfsService, leader *stream.Leader, worker *stream.WorkerService) {

	// Start CLI to handle user commands
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter command: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		args := strings.Split(input, " ")
		fmt.Println("the command is :", args)

		switch args[0] {
		case "list_mem":
			listMembership(ms.MemberShipList)
		case "lm":
			listMembership(ms.MemberShipList)

		case "list_self":
			listSelf(serverId)

		case "join":
			//utils.ClearFilesOnJoin("HyDfs/Files/Logs")
			// deleteFiles("HyDfs/Files")
			//deleteDirectories("HyDfs/Files/Logs")
			if serverId == groupMembership.IntroducerNodeId {
				fmt.Println("Introducer node cannot join the group.")
				continue
			}
			groupMembership.JoinRequestChan <- serverId
			//fmt.Printf("Node %d joined the group.\n", serverId)

		case "leave":
			ms.LeavingNode(serverId)
			//fmt.Printf("Node %d left the group.\n", serverId)

		case "enable_sus":
			groupMembership.SuspicionChan <- true
			fmt.Println("Suspicion detection enabled.")

		case "disable_sus":
			groupMembership.SuspicionChan <- false
			//fmt.Println("Suspicion detection disabled.")

		case "status_sus":
			if groupMembership.IsSuspicionEnabled() {
				fmt.Println("Suspicion detection is currently enabled.")
			} else {
				fmt.Println("Suspicion detection is currently disabled.")
			}

		case "show_sus":
			getSuspicions(ms.MemberShipList)

		case "exit":
			fmt.Println("Exiting CLI...")
			return

		case "show_msgs":
			print_buffer_msgs()

		case "runCordinator":
			hydfs.RunCoordinator(config.Conf.Server[serverId-1])

		case "rc", "create":
			// PerformWrite(args[1], args[2], true, hs)
			output := m.Output{
				TupleId: "abcdef",
				Key:     "hello",
				Value:   1,
				TaskId:  0,
			}
			data, _ := json.Marshal(output)
			PerformWriteData(data, args[2], true, hs)
		case "append":
			PerformWrite(args[1], args[2], false, hs)
		case "read", "get":
			readCommand := models.CreateCommand{
				LocalFileName: "HyDfs/Files/" + args[2],
				HydfsFileName: utils.HDFS_FILE_PATH + "/" + args[1],
			}
			resp := hs.DoReadOperationClient(readCommand)
			fmt.Println(resp.Message)

		case "getfromreplica", "getrc":
			vmId, _ := strconv.Atoi(args[1])
			readCommand := models.CreateCommand{
				VmId:          vmId,
				LocalFileName: "HyDfs/Files/" + args[3],
				HydfsFileName: utils.HDFS_FILE_PATH + "/" + args[2],
			}
			fmt.Println(hs.DoReadOperationClient(readCommand).Message)

		case "merge":
			hydfsFileName := utils.HDFS_FILE_PATH + "/" + args[1]
			err := hs.InitateMerge(models.MergeCommand{HydfsFileName: hydfsFileName})
			if err != nil {
				fmt.Println("Error merging files:", err)
			} else {
				fmt.Println("Merge Completed.")
			}

		case "fr":
			fmt.Println("--------------------File Routing---------------------------------------")
			fmt.Println(config.GetFileHashId("hello.html"))
			// fmt.Println(utils.FileRouting(config.GetFileHashId("hello.html")))
			fmt.Println("Introducer peer Id", config.GetServerIdInRing("127.0.0.1:8082"))
			fmt.Println("------------------------------------------------------------------------")

		case "store":
			fmt.Println("-------------------- store: Display HDFS Files Stored ---------------------------------------")
			fileinfo := utils.GetFilesUsingHash()
			for _, file := range fileinfo {
				fmt.Printf("File: %s, ID: %d\n", file.FileName, file.FileId)
			}
			fmt.Println("My Peer ID:", config.Conf.PeerId)

			fmt.Println("----------------------------------------------------------------------------------")

		case "ls":
			fmt.Println("-------------------------- ls: HDFS file's servers ---------------------------------------")
			fileHashId := config.GetFileHashId(utils.HDFS_FILE_PATH + "/" + args[1])
			peerIDs := utils.FileRouting(fileHashId, ms.MemberShipList)
			fmt.Printf("File %s (Hash ID: %d) is located at peer IDs: %v\n", args[1], fileHashId, peerIDs)
			fmt.Println("----------------------------------------------------------------------------------")

		case "multiappend":
			multiAppendReq, err := createMultiAppendRequest(args[1], args[2], args[3])
			if err != nil {
				fmt.Println("Invalid input format")
				break
			}
			err = hs.InitiateMultiAppend(multiAppendReq)
			fmt.Println("Multiappend completed.")

		case "job":
			//job app1-op1 app1-op2 test.csv output.txt Punched
			//job app1-op1 app1-op2 test2.csv output.txt Outlet

			//job app2-op1 app2-op2 test2.csv output.txt "Punched Telespar"
			//job app2-op1 app2-op2 test.csv output.txt "Punched Telespar"
			//job app2-op1 app2-op2 test3_1000.csv output.txt "Punched Telespar"
			//job app2-op1 app2-op2 test4_1000.csv output.txt "Punched Telespar"
			// Parse the input into arguments, handling quoted strings
			args := parseArguments(input)

			// Ensure the correct number of arguments
			if len(args) < 6 {
				fmt.Println("Invalid input format. Usage: job <op1_exe> <op2_exe> <hydfs_src_file> <hydfs_dest_filename> <pattern>")
				continue
			}

			// Extract arguments
			op1Exe := args[1]
			op2Exe := args[2]
			hydfsSrcFile := args[3]
			hydfsDestFile := args[4]
			pattern := args[5]

			// Create the Rain request
			req := createRainRequest(op1Exe, op2Exe, hydfsSrcFile, hydfsDestFile, pattern)
			fmt.Println("pattern", req.MetaData["pattern"])

			// Call the leader to create the job
			if leader != nil {
				leader.CreateJob(req)
				continue
			} else {
				err := worker.SendCreateJobRequestToLeader(req)
				fmt.Println("Job request sent to leader", err)

			}

		case "conn":
			worker.CreatePersistentConnections()
			if leader != nil {
				leader.ConnectLeaderToWorkers()
			}
			// if err := worker.ConnectToWorkers(); err != nil {
			// 	log.Println("Failed to connect to workers: %v", err)
			// }
		case "taskMap":
			if leader != nil {
				leader.PrintTaskMap()
				leader.PrintWorkerTaskQueue()
			}
		default:
			fmt.Println("Unknown command. Available commands are:")
			fmt.Println("list_mem, list_self, join, leave, enable_sus, disable_sus, status_sus, show_sus, exit")
		}
	}
}

func getSuspicions(memList groupMembership.MembershipList) {
	var suspectedNodes []int
	for _, member := range memList {
		if member.Status == "SUSPECTED" {
			suspectedNodes = append(suspectedNodes, member.NodeId)
		}
	}
	if len(suspectedNodes) > 0 {
		fmt.Println("Suspected nodes:", suspectedNodes)
	} else {
		fmt.Println("No nodes are suspected.")
	}
}

func listMembership(memList groupMembership.MembershipList) {
	fmt.Println("Membership list:")
	groupMembership.PrintMembershipList(memList)
	// for _, member := range groupMembership.GetMembershipList() {
	// 	fmt.Printf("NodeId: %d TimeStamp: %v, Status: %s,\n", member.NodeId, member.TimeStamp.Format("15:04:05"), member.Status)
	// }
}

func listSelf(serverID int) {
	fmt.Printf("Self's Node ID: %d\n", serverID)
}

func print_buffer_msgs() {
	groupMembership.PrintCurrentMsgsInBuffer()
}

func deleteDirectories(dirPath string) {
	log.Println("In delete directories")
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", err)
			return err
		}
		log.Println("No error In delete directories")
		// Check if it's a directory (not a file) and delete it
		if info.IsDir() && path != dirPath { // Ensure we don’t delete the root directory
			log.Println("it is a directory to del in del directories")
			err = os.RemoveAll(path) // Recursively delete directory and its contents
			if err != nil {
				fmt.Println("Error deleting directory:", err)
			} else {
				fmt.Println("Deleted directory:", path)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking through directory:", err)
	}
}

func PerformWrite(localFileName string, hydfsFileName string, isCreate bool, hs *hydfs.HyDfsService) {
	createCommand := models.CreateCommand{
		LocalFileName: "HyDfs/Files/" + localFileName,
		HydfsFileName: utils.HDFS_FILE_PATH + "/" + hydfsFileName,
		IsCreate:      isCreate,
	}
	log.Println("createCommand: ", createCommand)
	err := hs.DoWriteOperationClient(createCommand)
	if err != nil {
		fmt.Println("Error while writing file: %s", err.Error())
	} else {
		fmt.Println("File written successfully")
	}
}
func PerformWriteData(data []byte, hydfsFileName string, isCreate bool, hs *hydfs.HyDfsService) {
	createCommand := models.CreateCommand{
		FileData:        data,
		HydfsFileName:   utils.HDFS_FILE_PATH + "/" + hydfsFileName,
		IsCreate:        isCreate,
		IsStreamResults: true,
	}
	err := hs.DoWriteOperationClient(createCommand)
	if err != nil {
		fmt.Println("Error while writing file: %s", err.Error())
	} else {
		fmt.Println("File written successfully")
	}
}

func createMultiAppendRequest(hydfsFileName string, arg1 string, arg2 string) (models.MultiAppendCommand, error) {
	localFiles := strings.Split(arg1, ",")
	//iterate through locaFiles and add HYDFS FILEPATH to the prefix
	for i, file := range localFiles {
		localFiles[i] = "HyDfs/Files/" + file
	}

	// Assuming args[3] is a comma-separated string of integers
	vmStrings := strings.Split(arg2, ",")
	var vms []int
	for _, vmStr := range vmStrings {
		vm, err := strconv.Atoi(vmStr)
		if err != nil {
			fmt.Printf("Invalid integer value: %s\n", vmStr)
			return models.MultiAppendCommand{}, err
		}
		vms = append(vms, vm)
	}
	return models.MultiAppendCommand{
		HydfsFileName: utils.HDFS_FILE_PATH + "/" + hydfsFileName,
		LocalFiles:    localFiles,
		Vms:           vms,
	}, nil
}

//	func deleteFiles(dirPath string) {
//		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
//			if err != nil {
//				fmt.Println("Error accessing path:", err)
//				return err
//			}
//			// Check if it's a file (not a directory) and delete it
//			if !info.IsDir() {
//				err = os.Remove(path)
//				if err != nil {
//					fmt.Println("Error deleting file:", err)
//				}
//			}
//			return nil
//		})
//		if err != nil {
//			fmt.Println("Error walking through directory:", err)
//		}
//	}
func parseArguments(input string) []string {
	var args []string
	var currentArg strings.Builder
	inQuotes := false

	for _, char := range input {
		switch {
		case char == '"' && !inQuotes: // Start quoted section
			inQuotes = true
		case char == '"' && inQuotes: // End quoted section
			inQuotes = false
		case char == ' ' && !inQuotes: // Split on spaces outside quotes
			if currentArg.Len() > 0 {
				args = append(args, currentArg.String())
				currentArg.Reset()
			}
		default: // Add character to current argument
			currentArg.WriteRune(char)
		}
	}

	// Add the last argument if it exists
	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args
}
