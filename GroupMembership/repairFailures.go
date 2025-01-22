package groupMembership

import (
	config "cs425/Config"
	"log"
)

//Node 4 - sender
// Node 5 - failed
//Node 6 - sender,reciver from 4
//Node 7- reciver from 4
//Node-8 receiver from 6

type ReplicateFail struct {
	FromFile int
	ToFile   int
	Sender   int
	Receiver int
}

func (ms *MembershipService) HandleFileTransferRequestsInFailureRepair(failedNode int, prevPeers []int, nextPeers []int) []ReplicateFail {

	ms.MemberShipListMutex.Lock()
	defer ms.MemberShipListMutex.Unlock()

	for i := 0; i < len(prevPeers)/2; i++ {
		prevPeers[i], prevPeers[len(prevPeers)-i-1] = prevPeers[len(prevPeers)-i-1], prevPeers[i]
	}

	// Define file transfer ranges for replication
	transferRanges := []ReplicateFail{
		// Transfer from 3rd previous peer to 2nd previous peer
		{FromFile: prevPeers[2], ToFile: prevPeers[1], Sender: prevPeers[0], Receiver: nextPeers[0]},

		// Transfer from 2nd previous peer to recent previous peer
		{FromFile: prevPeers[1], ToFile: prevPeers[0], Sender: prevPeers[0], Receiver: nextPeers[1]},

		// Transfer from recent previous peer to failed node’s 2nd successor
		{FromFile: prevPeers[0], ToFile: config.GetServerIdInRing(config.Conf.Server[failedNode-1]), Sender: nextPeers[0], Receiver: nextPeers[2]},
	}
	log.Printf("\nTransfer ranges for failed node %d: %v", config.GetServerIdInRing(config.Conf.Server[failedNode-1]), transferRanges)
	return transferRanges
}
func (ms *MembershipService) GetOwnFileRanges(replicateFailFiles []ReplicateFail) []ReplicateFail {

	var selfTransferRanges []ReplicateFail
	for _, replicateFailFile := range replicateFailFiles {

		if replicateFailFile.Sender == config.Conf.PeerId {
			selfTransferRanges = append(selfTransferRanges, replicateFailFile)
		}
	}
	log.Println("Self transfer range: ", selfTransferRanges)

	return selfTransferRanges

}

func (ms *MembershipService) HypotheticalIndexForPeerId(peerId int) int {
	left, right := 0, len(ms.MemberShipList)

	// Using bbinary search to find the hypothetical index for peerId.
	for left < right {
		mid := (left + right) / 2
		if ms.MemberShipList[mid].PeerId < peerId {
			left = mid + 1
		} else {
			right = mid
		}
	}
	return left // Left now represents the hypothetical index.
}

func (ms *MembershipService) HypotheticalNeighborsForPeerId(peerId int) ([]int, []int) {
	numPeers := len(ms.MemberShipList)

	// Create a hypothetical list with the new peerId inserted in sorted order
	hypotheticalList := make([]int, numPeers+1)
	inserted := false
	for i, j := 0, 0; i < numPeers; i, j = i+1, j+1 {
		if !inserted && ms.MemberShipList[i].PeerId > peerId {
			hypotheticalList[j] = peerId
			inserted = true
			j++ // Move to next position in hypotheticalList for remaining elements
		}
		hypotheticalList[j] = ms.MemberShipList[i].PeerId
	}
	if !inserted {
		hypotheticalList[numPeers] = peerId // Append peerId if it's the largest
	}

	// Find the position of peerId in the hypothetical list
	var peerIndex int
	for i, id := range hypotheticalList {
		if id == peerId {
			peerIndex = i
			break
		}
	}

	// Use circular indexing to find three preceding and succeeding peers
	precedingPeers := make([]int, 0, 3)
	succeedingPeers := make([]int, 0, 3)

	for i := 1; i <= 3; i++ {
		precedingIndex := (peerIndex - i + len(hypotheticalList)) % len(hypotheticalList)
		succeedingIndex := (peerIndex + i) % len(hypotheticalList)

		if precedingIndex < 0 || precedingIndex >= len(hypotheticalList) {
			log.Println("Invalid preceding index:", precedingIndex)
			return []int{}, []int{}
		}
		precedingPeers = append([]int{hypotheticalList[precedingIndex]}, precedingPeers...)
		succeedingPeers = append(succeedingPeers, hypotheticalList[succeedingIndex])
	}

	return precedingPeers, succeedingPeers
}
