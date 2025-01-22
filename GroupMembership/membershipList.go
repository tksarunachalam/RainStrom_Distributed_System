package groupMembership

import (
	config "cs425/Config"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

type MembershipList []*MembershipEntry

type Status string

// status in the membership list
const (
	ACTIVE    Status = "ACTIVE"
	FAILED    Status = "FAILED"
	SUSPECTED Status = "SUSPECTED"
	LEFT      Status = "LEFT"
)

type MembershipEntry struct {
	//Membership lit entry contains:
	//Node Id of the machine: Ip address + port number which is mapped to the machine number (1,2,3....)
	//Timestamp
	//Status

	NodeId    int
	TimeStamp time.Time
	Status    Status
	PeerId    int
	//Address Str

}
type MembershipService struct {
	MemberShipList      MembershipList
	LeaveNodeSignalChan chan int
	FailNodeSignalChan  chan int
	JoinNodeSignalChan  chan int

	MemberShipListMutex sync.Mutex
}

// to get the index of a particular node from the memebrship list
func (m MembershipList) getIndexOfNode(node int) int {
	for index, value := range m {
		if value.NodeId == node {
			return index
		}
	}
	return -1
}

// check if the node is present in the membership list or not
func (m MembershipList) contains(node int) bool {

	if m.getIndexOfNode(node) == -1 {
		return false
	} else {
		return true
	}
}

// returning the membership list
func (ms *MembershipService) GetMembershipList() MembershipList {
	return ms.MemberShipList
}

// for creating the membership list entry
func AddMembershipListEntry(nodeId int, status Status, peerId int) *MembershipEntry {
	return &MembershipEntry{
		NodeId:    nodeId,
		TimeStamp: time.Now(),
		Status:    status,
		PeerId:    peerId,
	}
}

// Add function adds a membership entry to the list
func (mL *MembershipList) Add(entry *MembershipEntry) {

	*mL = append(*mL, entry)
	// Sort the list by NodeId after adding the new entry
	sort.Slice(*mL, func(i, j int) bool {
		return (*mL)[i].PeerId < (*mL)[j].PeerId
	})
}
func (mL *MembershipList) AddOrUpdate(message *Message) {
	index := mL.getIndexOfNode(message.NodeId)
	if index != -1 {
		//already present update the status to live
		if message.MsgType != JOIN {
			(*mL)[index].Status = Status(message.MsgType)
			(*mL)[index].TimeStamp = message.TimeStamp
		} else {
			(*mL)[index].Status = Status(ACTIVE)
			(*mL)[index].PeerId = config.GetServerIdInRing(config.Conf.Server[message.NodeId-1])
		}
	} else {
		//not present add to list
		mL.Add(AddMembershipListEntry(message.NodeId, ACTIVE, config.GetServerIdInRing(config.Conf.Server[message.NodeId-1])))
	}

}

// Update updates the status of an existing membership entry by NodeId.
func (mL MembershipList) Update(nodeId int, status Status) bool {

	for _, entry := range mL {
		if entry.NodeId == nodeId {
			entry.Status = status
			return true
		}
	}
	return false // Entry not found
}

// Remove removes a membership entry by NodeId.
func (mL *MembershipList) Remove(nodeId int) bool {

	for i, entry := range *mL {
		if entry.NodeId == nodeId {
			*mL = append((*mL)[:i], (*mL)[i+1:]...) // Remove entry
			return true
		}
	}
	return false // Entry not found
}

// function to get the 3 successor neighbours of a node for the pings using the order from membership list
// We are structuring our membership list to be ordered
func (mL MembershipList) GetNeighbours(nodeId int) (neighbours []int) {

	if mL.contains(nodeId) {
		index := mL.getIndexOfNode(nodeId)
		for j := 1; j <= KNeighbors; j++ {
			neighbours = append(neighbours, mL[(j+index)%len(mL)].NodeId)
		}
	} else {
		log.Printf("\nNode Id: %d not found in membership list", nodeId)
		PrintMembershipList(mL)
	}
	return neighbours
}

func (mL MembershipList) GetNeighboursPeerId(peerId int) (neighbours []int) {

	if mL.containsPeer(peerId) {
		index := mL.GetIndexOfPeer(peerId)
		for j := 1; j <= 2; j++ {
			neighbours = append(neighbours, mL[(j+index)%len(mL)].PeerId)
		}
	} else {
		log.Printf("\nPeer Id: %d not found in membership list", peerId)
	}
	return neighbours
}

func (mL MembershipList) containsPeer(peerId int) bool {
	for _, peer := range mL {
		if peer.PeerId == peerId {
			return true
		}
	}
	return false
}

func (mL MembershipList) GetIndexOfPeer(peerId int) int {
	for index, peer := range mL {
		if peer.PeerId == peerId {
			return index
		}
	}
	return -1 // Return -1 if the peer is not found
}

// function that takes action for the particular node id's mL based on the action messages it receives from other nodes
func (ms *MembershipService) PerformActionOnMessage(message Message) bool {
	// memberShipListMutex.Lock()
	// defer memberShipListMutex.Unlock()
	switch message.MsgType {
	case FAIL:
		log.Printf("Processing REMOVE message for Node %d", message.NodeId)
		//
		if ms.MemberShipList.contains(message.NodeId) {

			ms.FailNodeSignalChan <- message.NodeId
		}
		ms.MemberShipList.Remove(message.NodeId)
	case JOIN:
		log.Printf("Processing JOIN message for Node %d", message.NodeId)
		//fmt.Printf("Processing JOIN message for Node %d\n", message.NodeId)
		ms.MemberShipList.AddOrUpdate(&message)
	case LEAVE:
		log.Printf("Processing LEAVE message for Node %d", message.NodeId)
		ms.MemberShipList.Remove(message.NodeId)
	case SUSPECT:
		log.Printf("Processing SUSPECT message for Node %d", message.NodeId)
		ms.MemberShipList.Update(message.NodeId, SUSPECTED)
	case ALIVE:
		log.Println("Processing ALIVE message for Node %d", message.NodeId)
		ms.MemberShipList.AddOrUpdate(&message)
	default:
		log.Printf("Unknown message type %s for Node %d", message.MsgType, message.NodeId)
	}
	return true
}

// removes message from buffer if no action is neeeded
func (mL MembershipList) RemoveMsgIfNoActionNeeded(messages []*Message) (filteredMsgs []*Message) {
	for _, message := range messages {
		//node already removed from membership list no need to procees. can remove msg
		if message.MsgType == FAIL && !mL.contains(message.NodeId) {
			continue
		}
		//node already present in membership list
		if message.MsgType == JOIN && mL.contains(message.NodeId) {
			mL[mL.getIndexOfNode(message.NodeId)].TimeStamp = message.TimeStamp
			continue
		}

		if message.MsgType == SUSPECT && mL.contains(message.NodeId) &&
			strings.Contains(string(mL[mL.getIndexOfNode(message.NodeId)].Status), "SUSPECT") {
			continue
		}
		filteredMsgs = append(filteredMsgs, message)
	}
	return filteredMsgs

}

func (ml *MembershipList) IsSuspected(nodeId int) bool {

	// Check if node is in suspected state
	for _, entry := range *ml {
		if entry.NodeId == nodeId && entry.Status == SUSPECTED {
			return true
		}
	}
	return false
}

func (ml *MembershipList) Rejuvenate(nodeId int) {
	for _, entry := range *ml {
		if entry.NodeId == nodeId && entry.Status == SUSPECTED {
			entry.Status = ACTIVE
			entry.TimeStamp = time.Now() // Update the timestamp
			log.Printf("Node %d rejuvenated in membership list", nodeId)
			break
		}
	}
}

// printing mebership list for debugging
func PrintMembershipList(memList MembershipList) {
	for _, entry := range memList {
		fmt.Printf("\nNodeId: %d, PeerId : %d ,TimeStamp: %v, Status: %s", entry.NodeId, entry.PeerId, entry.TimeStamp.Format("15:04:05"), entry.Status)

	}
}
