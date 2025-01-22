package groupMembership

import (
	"cs425/GroupMembership/utils"
	"encoding/json"
	"log"
	"net"
)

var SuspicionChan = make(chan bool)

var SuspicionModeEnabled bool

type SwitchSuspicionRequest struct {
	MsgType         string
	SwitchSuspicion bool
}

func SendSuspicionSignalToOtherNodes(conn net.PacketConn, suspicion bool) {

	allServers := utils.ReadConfig()

	// Connect to introducer
	for i, node := range allServers {

		if i+1 == ServerId {
			continue
		}
		addr, err := net.ResolveUDPAddr("udp", node)
		if err != nil {
			log.Printf("Error resolving address for %s: %v\n", node, err)
			return
		}
		encodedReq, err := json.Marshal(SwitchSuspicionRequest{MsgType: SWITCH, SwitchSuspicion: suspicion})
		if err != nil {
			log.Println("Error in encoding switch suspicion", err)
			return
		}
		_, err = conn.WriteTo(encodedReq, addr)
		if err != nil {
			log.Println("Failed send switch suspicion request to node:", addr, err)
			continue
		}
	}

}

func ProcessSwitchSuspicionSignalRequest(request []byte) (int, bool) {

	var switchSuspicionResponse SwitchSuspicionRequest

	// log.Println("debug 1: bytes recevied: ", string(request))
	err := json.Unmarshal(request, &switchSuspicionResponse)
	if err != nil {
		//normal ping ack msg not switch
		return 0, false
	}
	if switchSuspicionResponse.MsgType == SWITCH {
		return 1, switchSuspicionResponse.SwitchSuspicion
	}
	return -1, false
}

func IsSuspicionEnabled() bool {
	return SuspicionModeEnabled
}
