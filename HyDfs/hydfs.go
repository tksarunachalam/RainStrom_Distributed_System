package hydfs

import (
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	appendLogs "cs425/HyDfs/AOL"
	"cs425/HyDfs/utils"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

type HyDfsService struct {
	Ms                  *groupMembership.MembershipService
	LeaveNodeSignalChan chan int
	FailNodeSignalChan  chan int
	JoinNodeSignalChan  chan int
}

func InitHydfsService(ms *groupMembership.MembershipService) (*HyDfsService, error) {

	hs := new(HyDfsService)
	hs.Ms = ms
	hs.FailNodeSignalChan = make(chan int)
	hs.JoinNodeSignalChan = make(chan int)
	hs.LeaveNodeSignalChan = make(chan int)

	go hs.getJoinNodeSignal()
	go hs.getFailedNodeSignal()
	go hs.startAppendReadLogs()

	return hs, nil

}

func (hy *HyDfsService) startAppendReadLogs() {
	for {
		time.Sleep(utils.SLEEP_DURATION * time.Second)
		// log.Println("hdfs to read all")
		appendLogs.ReadAllLogs()

	}
}

func (hy *HyDfsService) getJoinNodeSignal() {
	for {
		if joinedNode, ok := <-hy.Ms.JoinNodeSignalChan; ok {
			log.Println("Hydfs service detected new join: ", joinedNode)
			if config.Conf.NodeId == groupMembership.IntroducerNodeId {
				hy.JoinNodeSignalChan <- joinedNode
			}
			//TODO:: call the file transfer on JOINS
			//func transferFileOnJoin
		} else if !ok {
			// If channel is closed, exit
			return
		}
		runtime.Gosched()
	}
}

func (hy *HyDfsService) getFailedNodeSignal() {
	var failedNodes []int
	var failedNodesMutex sync.Mutex
	processedNodes := make(map[int]struct{})

	for {
		// Collect failed nodes for 15 seconds
		select {
		case failedNode, ok := <-hy.Ms.FailNodeSignalChan:
			if !ok {
				// If channel is closed, exit
				return
			}

			log.Println("Hydfs service detected node fail: ", failedNode)
			if config.Conf.NodeId == groupMembership.IntroducerNodeId {
				hy.FailNodeSignalChan <- failedNode
			}

			failedNodesMutex.Lock()
			if _, exists := processedNodes[failedNode]; !exists {
				failedNodes = append(failedNodes, failedNode)
				processedNodes[failedNode] = struct{}{}
			}
			failedNodesMutex.Unlock()

		case <-time.After(20 * time.Second):
			// After 15 seconds, process all failed nodes
			failedNodesMutex.Lock()
			// Remove duplicates using a map as a set
			uniqueNodesMap := make(map[int]struct{})
			uniqueNodes := []int{}

			for _, node := range failedNodes {
				if _, exists := uniqueNodesMap[node]; !exists {
					uniqueNodesMap[node] = struct{}{}
					uniqueNodes = append(uniqueNodes, node)
				}
			}

			// Assign the unique nodes back to failedNodes
			failedNodes = uniqueNodes
			log.Println("Failed nodes set collected: ", failedNodes)
			fileTransferMap := make(map[int]map[string]struct{})
			for _, failedNode := range failedNodes {
				// Proceed with the existing failure handling logic
				log.Println("Processing failed node:", failedNode)

				// Get hypothetical neighbors for the failed node
				precedingPeers, succeedingPeers := hy.Ms.HypotheticalNeighborsForPeerId(config.GetServerIdInRing(config.Conf.Server[failedNode-1]))
				log.Printf("Preceding peers: %v, Succeeding peers: %v", precedingPeers, succeedingPeers)
				if len(precedingPeers) == 0 || len(succeedingPeers) == 0 {
					log.Printf("Failed to get preceding and succeeding peers for failed node %d", failedNode)
					continue
				}

				// Pass the lists to HandleFileTransferRequestsInFailureRepair
				transferRanges := hy.Ms.HandleFileTransferRequestsInFailureRepair(failedNode, precedingPeers, succeedingPeers)
				transferRanges = hy.Ms.GetOwnFileRanges(transferRanges)

				// Get files within the specified range and start the file transfer
				for _, transfer := range transferRanges {
					filesInRange, err := utils.GetFilesInRange(transfer.FromFile, transfer.ToFile)
					log.Println("Files in range", filesInRange)
					log.Println("Transferring files from ", transfer.FromFile, " to ", transfer.ToFile)

					if err != nil {
						log.Printf("Error retrieving files in range %d-%d: %v", transfer.FromFile, transfer.ToFile, err)
						continue
					}
					if _, exists := fileTransferMap[transfer.Receiver]; !exists {
						fileTransferMap[transfer.Receiver] = make(map[string]struct{})
					}
					for _, file := range filesInRange {
						fileTransferMap[transfer.Receiver][file] = struct{}{}
					}

					// Call the file transfer function
					// fmt.Println("Starting file send on failure")
					// hy.TransferFilesOnFailure(filesInRange, transfer.Sender, transfer.Receiver)
				}
			}
			for receiver, filesSet := range fileTransferMap {
				filesList := make([]string, 0, len(filesSet))
				for file := range filesSet {
					filesList = append(filesList, file)
				}
				fmt.Println("Starting file send on failure to receiver:", receiver)
				hy.TransferFilesOnFailure(filesList, config.Conf.PeerId, receiver)
			}

			// Clear the list of failed nodes after processing
			failedNodes = nil
			processedNodes = make(map[int]struct{})

			failedNodesMutex.Unlock()
		}
	}
}
