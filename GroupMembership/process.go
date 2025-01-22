package groupMembership

import (
	"cs425/GroupMembership/utils"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var currentMsgsInBuffer []*Message

// init membershipList with introducer nodeId
// var MemberShipList MembershipList = MembershipList{
// 	&MembershipEntry{NodeId: IntroducerNodeId, TimeStamp: time.Now(), Status: ACTIVE,
// 		PeerId: IntroducerPeerId},
// }

var Servers []string
var ServerId int

// var memberShipListMutex sync.Mutex
var ackChanMapMutex sync.Mutex // Mutex to guard ackChanMap
// var ackChanMap map[int]chan int // Map to store ackChan for each NodeId

type AckChannelInfo struct {
	AckChan   chan int // Channel for receiving ACKs
	PingCount int      // Track how many pings are sent
}

var ackChanMap map[int]*AckChannelInfo // NodeID to AckChannelInfo

func InitProcess() *MembershipService {

	ms := new(MembershipService)
	ms.MemberShipList = MembershipList{
		&MembershipEntry{NodeId: IntroducerNodeId, TimeStamp: time.Now(), Status: ACTIVE,
			PeerId: IntroducerPeerId},
	}

	ms.FailNodeSignalChan = make(chan int)
	ms.JoinNodeSignalChan = make(chan int)
	ms.LeaveNodeSignalChan = make(chan int)

	return ms

}

// RunProcess starts the UDP process for a node.
func (ms *MembershipService) RunProcess(servers []string, serverId int) {
	Servers = servers
	ServerId = serverId

	// Open a log file and set it as the log output
	logFile, err := os.OpenFile("groupMembership.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	logFile.Sync()
	log.SetOutput(logFile) // Redirect log output to the file

	// Confirm the log redirection by printing a message
	fmt.Println("Log output successfully redirected to", logFile.Name())

	// Initialize the map to store ackChans
	ackChanMap = make(map[int]*AckChannelInfo)

	// Local address for receiving messages
	conn, err := net.ListenPacket("udp", Servers[serverId-1])
	if err != nil {
		fmt.Printf("Error listening on %s: %v\n", Servers[serverId-1], err)
		return
	}
	defer conn.Close()

	go func() {
		for {
			select {
			case node := <-JoinRequestChan:
				log.Printf("Join request received for node: %d", node)
				SendRequestToIntroducer(conn, node, IntroducerNodeId) // Notify introducer about the join request

			case sus := <-SuspicionChan: // Triggered only when there's a new value in SuspicionChan
				if sus != SuspicionModeEnabled {
					log.Println("Switch suspicion received:", sus)
					// Set the suspicion mode based on the channel value
					SuspicionModeEnabled = sus
					// Notify other nodes about the change
					SendSuspicionSignalToOtherNodes(conn, sus)
				}
			}
		}
	}()

	// Goroutine to receive messages and handle PING/ACK
	go ms.listenUdpSignals(serverId, conn)

	// Goroutine to send pings to neighbors every 10 seconds
	go ms.sendSignalsToNeighbors(serverId, conn)

	// Keep running indefinitely
	select {}
}

// Listens to all UDP incoming packets
func (ms *MembershipService) listenUdpSignals(serverId int, conn net.PacketConn) {

	buffer := make([]byte, 1024)
	for {

		// Handle other incoming UDP messages
		n, addr, err := conn.ReadFrom(buffer)
		if err != nil {
			log.Println("Error reading from connection:", err)
			return
		}

		if ms.IsRequestToIntroducer(buffer[:n], conn) {
			encodedReq, _ := ms.getJoinResponse()
			log.Println("Introducer sending mem list to new node: "+string(encodedReq)+" address: ", addr.String())
			_, err = conn.WriteTo(encodedReq, addr)
			continue
		}
		if ms.IsResponseFromIntroducer(buffer[:n]) {
			log.Println("Received response from Introducer, Node joined ")
			ResponseFromIntroducer = true
			// time.Sleep(4 * time.Second)

		}
		intCheck, switchSignal := ProcessSwitchSuspicionSignalRequest(buffer[:n])
		if intCheck == 1 {
			SuspicionChan <- switchSignal
			log.Println("Suspicion Mode switched")
		}

		signal, err := DecodeSignal(buffer[:n])
		if err != nil {
			log.Printf("Error decoding signal at Node %d from %s: error: \n", serverId, addr, err)
			continue
		}

		//log.Printf("Node %d received signal %s from %s", serverId, signal.SignalType, addr.String())
		ms.processSignal(&signal, addr, conn)
	}

}

// Periodically pings to neighbors
func (ms *MembershipService) sendSignalsToNeighbors(serverId int, conn net.PacketConn) {
	ticker := time.NewTicker(time.Duration(PingFrequency) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			neighbors := ms.MemberShipList.GetNeighbours(serverId)

			neighbors = utils.ConvertListToSet(neighbors)
			log.Printf("Node %d pinging neighbors: %v", serverId, neighbors)

			// log.Printf("\n Msgs in buffer: ", currentMsgsInBuffer)
			// Create a copy of the current messages in buffer to avoid race conditions
			bufferCopy := make(MessagesList, len(currentMsgsInBuffer))
			copy(bufferCopy, currentMsgsInBuffer)

			for _, neighbor := range neighbors {
				if neighbor != ServerId {
					ackChan := make(chan int) // Create a new ACK channel

					ackChanMapMutex.Lock()
					// Add a new entry for the neighbor's ACK channel
					ackChanMap[neighbor] = &AckChannelInfo{
						AckChan:   ackChan,
						PingCount: 1, // Initialize with 1 since we are about to send a ping
					}
					ackChanMapMutex.Unlock()

					go ms.pingNode(conn, neighbor, bufferCopy, ackChan)
				}
			}
			currentMsgsInBuffer = nil
			// time.Sleep(time.Duration(PingFrequency) * time.Second)
		}

	}
}

// processSignal handles received signals.
func (ms *MembershipService) processSignal(signal *Signal, addr net.Addr, conn net.PacketConn) {
	// log.Println("Signal recevied:", signal)
	//log.Printf("\n Msgs in buffer: ", currentMsgsInBuffer)

	filteredMsgs := ms.MemberShipList.RemoveMsgIfNoActionNeeded(signal.Messages)
	currentMsgsInBuffer = append(currentMsgsInBuffer, filteredMsgs...)
	currentMsgsInBuffer = RemoveConflictMessages(currentMsgsInBuffer)
	//log.Printf("\n After clearing: ", currentMsgsInBuffer)

	switch signal.SignalType {
	case "PING":
		//ping received
		if signal.Messages != nil {
			log.Println("Signal- ", signal.String(), signal.Messages)
		}
		// Send acknowledgment upon receving a ping from neighbor
		signalAck := &Signal{SignalType: ACK}
		signalByte, err := EncodeSignal(*signalAck)
		if err != nil {
			log.Println("Error encoding signal to bytes")
			return
		}

		_, err = conn.WriteTo(signalByte, addr)
		if err != nil {
			log.Println("Error sending acknowledgment:", err)
		}
		// log.Printf("Received ping from %s, sent acknowledgment", addr.String())

	case "ACK":
		indexFound := getIndexFromAddr(addr)
		log.Println("ACK received from Node: ", indexFound)
		ackChanMapMutex.Lock()
		if ackInfo, ok := ackChanMap[indexFound]; ok {
			// Send the ACK through the correct channel
			ackInfo.AckChan <- indexFound // Notify that ACK has been received
			ackInfo.PingCount--

			// Only delete the entry when no more pings are pending
			if ackInfo.PingCount <= 0 {
				close(ackInfo.AckChan)         // Close the channel to signal completion
				delete(ackChanMap, indexFound) // Remove the entry from the map
			}
		} else {
			log.Printf("No ackChan found for Node %d", indexFound)
		}
		ackChanMapMutex.Unlock()
	}
	for _, msg := range currentMsgsInBuffer {
		ms.PerformActionOnMessage(*msg)
	}
}

// pingNode sends a ping signal to a neighbor.
func (ms *MembershipService) pingNode(conn net.PacketConn, neighbor int, messagesInBuffer []*Message, ackChan chan int) {

	addr, err := net.ResolveUDPAddr("udp", Servers[neighbor-1])
	if err != nil {
		log.Printf("Error resolving address for %s: %v\n", Servers[neighbor-1], err)
		return
	}

	// Use the provided message for the ping
	signal := CreateSignal("PING", neighbor, messagesInBuffer)
	signalByte, err := EncodeSignal(*signal)
	if err != nil {
		log.Println("Error converting signal to bytes")
		return
	}

	_, err = conn.WriteTo(signalByte, addr)
	if err != nil {
		log.Printf("Error sending ping to %s: %v\n", Servers[neighbor-1], err)
		return
	}

	log.Printf("\nPing sent to Node: %d with msgs: %v", neighbor, messagesInBuffer)

	// Wait for acknowledgment with a timeout of failTimeout seconds
	select {
	case <-ackChan: // Correctly received ACK
		log.Printf("Acknowledgment received from Node %d", neighbor)
		ms.markNodeAlive(neighbor) // Rejuvenate the node if suspected
	case <-time.After(time.Duration(failTimeout+2) * time.Second):
		log.Printf("Timeout waiting for ACK from Node %d", neighbor)
		ms.markNodeFailed(neighbor) // Timeout reached, mark node as suspected or failed
	}

}

// Handling suspects and failure of nodes
func (ms *MembershipService) markNodeFailed(neighbor int) {

	ms.MemberShipListMutex.Lock()
	// defer func() {
	// 	memberShipListMutex.Unlock()
	// 	log.Println("FAiled:::Unlocked membership list mutex")
	// }()

	// If suspicion is enabled, mark as suspected instead of failed
	if IsSuspicionEnabled() {
		// memberShipListMutex.Lock()
		// defer memberShipListMutex.Unlock()

		if ms.MemberShipList.IsSuspected(neighbor) {
			log.Printf("Node %d is already suspected, avoiding duplicate suspicion.\n", neighbor)
			return
		}

		fmt.Printf("Node %d marked as suspected\n", neighbor)
		log.Printf("Node %d marked as suspected\n", neighbor)
		//Print Membership List
		if ms.MemberShipList.Update(neighbor, SUSPECTED) {
			log.Println("Membership list updated for suspected node")
		} else {
			log.Println("Error updating membership list for suspected node")
		}
		PrintMembershipList(ms.MemberShipList)
		currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(SUSPECT, neighbor))

		// Start a timer to eventually mark as failed if no ACK is received
		go func() {
			time.Sleep(time.Duration(SuspectWaitTimeout) * time.Second) // Prespecified suspicion timeout
			// memberShipListMutex.Lock()
			// defer memberShipListMutex.Unlock()

			if ms.MemberShipList.IsSuspected(neighbor) { // Ensure it's still suspected
				log.Printf("Node %d suspected and now marked as failed after timeout\n", neighbor)
				fmt.Printf("Node %d suspected and now marked as failed after timeout\n", neighbor)
				currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(FAIL, neighbor))
				if ms.MemberShipList.contains(neighbor) {

					ms.FailNodeSignalChan <- neighbor
				}
				ms.MemberShipList.Remove(neighbor)
			}
		}()
	} else {
		// Immediately mark as failed if suspicion is disabled
		// log.Printf("Node %d marked as failed (no suspicion)\n", neighbor)
		fmt.Printf("Node %d marked as failed (no suspicion)\n", neighbor)
		// memberShipListMutex.Lock()
		// defer func() {
		// 	memberShipListMutex.Unlock()
		// 	log.Println("FAiled:::Unlocked membership list mutex")
		// }()
		if ms.MemberShipList.contains(neighbor) {

			ms.FailNodeSignalChan <- neighbor
		}

		ms.MemberShipList.Remove(neighbor)
		currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(FAIL, neighbor))
	}
	ms.MemberShipListMutex.Unlock()
	log.Println("FAiled:::Unlocked membership list mutex")

	// ms.FailNodeSignalChan <- neighbor

}

// Node is alive aftermarking suspected
func (ms *MembershipService) markNodeAlive(neighbor int) {
	ms.MemberShipListMutex.Lock()
	defer ms.MemberShipListMutex.Unlock()

	if ms.MemberShipList.IsSuspected(neighbor) {
		// log.Printf("Node %d suspected, but now marked as alive\n", neighbor)
		fmt.Printf("Node %d suspected, but now marked as alive\n", neighbor)
		// Spread "alive" message to other nodes
		currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(ALIVE, neighbor))
		ms.MemberShipList.Rejuvenate(neighbor) // Mark as alive in the list
	}
}

func (ms *MembershipService) LeavingNode(node int) {
	log.Printf("Node %d wants to leave\n", node)

	// Lock the membership list before removing
	ms.MemberShipListMutex.Lock()
	defer ms.MemberShipListMutex.Unlock()
	//send ping msg to other nodes to do the leave update and remove from mL
	currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(LEAVE, node))

	log.Printf("Node %d left the Membership List\n", node)
}

func getIndexFromAddr(addr net.Addr) int {
	for index, str := range Servers {
		if addr.String() == str {
			return index + 1 // Adjust for zero-based index
		}
	}
	return -1 // Return -1 if not found
}
