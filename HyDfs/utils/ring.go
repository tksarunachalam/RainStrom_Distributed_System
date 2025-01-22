package utils

import groupMembership "cs425/GroupMembership"

// func GetPeerNeighbours(peerId int) []int {
// 	//get neighbors based on the peerid and mem list

// 	neighbors := groupMembership.MemberShipList.GetNeighboursPeerId(peerId)

// 	//return slice of neighbor peer id
// 	return neighbors

// }

func FileRouting(fileHashId int, mL groupMembership.MembershipList) []int {
	//Check and comapre the file id using the peer id's in the mem list
	// file will belong to the peer Id which is the first most greater id than the file id else the top of the list element

	var routedFile []int

	for i := 0; i < len(mL); i++ {
		if mL[i].PeerId > fileHashId {
			routedFile = append(routedFile, mL[i].PeerId)
			routedFile = append(routedFile, mL[(i+1)%len(mL)].PeerId)
			routedFile = append(routedFile, mL[(i+2)%len(mL)].PeerId)
			break
		}
	}
	if len(routedFile) == 0 && len(mL) > 0 {
		routedFile = append(routedFile, mL[0].PeerId)
		routedFile = append(routedFile, mL[1%len(mL)].PeerId)
		routedFile = append(routedFile, mL[2%len(mL)].PeerId)
	}

	//returns the list the server id with the file and replicas
	return routedFile
}

func GetNodeIdsOfFile(fileHashId int, mL groupMembership.MembershipList) []int {
	var nodeIds []int
	filePeerIds := FileRouting(fileHashId, mL)
	for _, peerId := range filePeerIds {
		//get the worker id of the peer id
		nodeIds = append(nodeIds, mL[mL.GetIndexOfPeer(peerId)].NodeId)
	}
	return nodeIds

}
