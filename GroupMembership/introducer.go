package groupMembership

import (
	config "cs425/Config"
	"cs425/GroupMembership/utils"
	"encoding/json"
	"log"
	"net"
)

var JoinRequestChan = make(chan int)

var ResponseFromIntroducer bool

// Send join request to introducer
func SendRequestToIntroducer(conn net.PacketConn, newNodeId int, introducerId int) {

	if introducerId != IntroducerNodeId {
		log.Printf("introducerId %d is not the introducer.", introducerId)
		return
	}
	allServers := utils.ReadConfig()

	// Connect to introducer
	addr, err := net.ResolveUDPAddr("udp", allServers[introducerId-1])
	if err != nil {
		log.Printf("Error resolving address for %s: %v\n", allServers[introducerId-1], err)
		return
	}

	var joinRequest JoinRequest = JoinRequest{MsgType: JOINREQUEST,
		NodeId: newNodeId}

	// Correctly encode the join request
	encoded, err := EncodeJoinRequest(joinRequest)
	if err != nil {
		log.Printf("Error encoding join request: %v\n", err)
		return
	}
	//clear this logs
	log.Println("writng to introdcuer with address", addr.String())
	_, err = conn.WriteTo(encoded, addr)
	if err != nil {
		log.Println("Failed to send request to introducer:", err)
	} else {
		log.Println("Sent request to introducer")
	}
}

// introducer recevies the request. Adds to membership list and dissimates the msgs to its buffer
func (ms *MembershipService) IsRequestToIntroducer(request []byte, conn net.PacketConn) bool {

	joinRequest, err := DecodeJoinRequest(request)
	if err == nil && joinRequest.MsgType == JOINREQUEST {
		log.Println("Introducer rcvd new join request from node: ", joinRequest.NodeId)
		ms.MemberShipListMutex.Lock()

		ms.MemberShipList.Add(AddMembershipListEntry(joinRequest.NodeId, ACTIVE, config.GetServerIdInRing(config.Conf.Server[joinRequest.NodeId-1])))
		ms.MemberShipListMutex.Unlock()

		log.Println("added to membership list : ", ms.MemberShipList)
		// Add the JOIN message to the buffer
		currentMsgsInBuffer = append(currentMsgsInBuffer, MessageEntry(JOIN, joinRequest.NodeId))

		go func(nodeId int) {
			ms.JoinNodeSignalChan <- nodeId
			log.Println("Successfully sent to JoinNodeSignalChan for node:", nodeId)
		}(joinRequest.NodeId)
		return true

		// select {
		// case ms.JoinNodeSignalChan <- joinRequest.NodeId:
		// 	// Successfully sent to channel
		// 	log.Println("enterring inside the channel")
		// 	return true
		// default:
		// 	// Handle the case if the channel write would block
		// 	log.Println("default case:")
		// 	return false
		// }

	}
	return false

}

// node recoves join response from introducer (and the updated mL)
func (ms *MembershipService) IsResponseFromIntroducer(request []byte) bool {
	var joinResponse JoinResponse

	// log.Println("debug 1: bytes recevied: ", string(request))
	err := json.Unmarshal(request, &joinResponse)
	if err != nil {
		log.Println("unmarshall error")
	}
	if err == nil && joinResponse.MsgType == JOINRESPONSE {
		log.Printf("\n New Node received response from introducer: %v", joinResponse.InitMembershipList)
		//assign the introducer memlist to the new node
		ms.MemberShipList = joinResponse.InitMembershipList
		return true
	} else if err != nil {
		log.Println("Error in converting the joinResponse", err)
	}
	return false

}
