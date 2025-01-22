package groupMembership

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// for the message types
const (
	FAIL         string = "FAILED"        //node has failed to send ack for the ping
	LEAVE        string = "LEFT"          //node has left
	SUSPECT      string = "SUSPECTED"     //node is suspected after no intial ack
	JOIN         string = "JOIN"          // node is joining
	ALIVE        string = "ACTIVE"        // node is alive after suspect
	JOINREQUEST  string = "JOIN_REQUEST"  //send join request to introducer
	JOINRESPONSE string = "JOIN_RESPONSE" //get join response from introducer
	SWITCH       string = "SWITCH"        //switch to suspect
)

// for the signals - all other actions will be passed as the message of the ping signal
const (
	PING string = "PING"
	ACK  string = "ACK"
)

type Signal struct {
	SignalType   string
	TargetNodeId int
	Messages     []*Message
}

type Message struct {
	MsgType   string
	NodeId    int
	TimeStamp time.Time
}
type MessagesList []*Message

type JoinRequest struct {
	MsgType string
	NodeId  int
}
type JoinResponse struct {
	MsgType            string
	InitMembershipList MembershipList
}

func (signal *Signal) String() string {
	var messages []string
	for _, msg := range signal.Messages {
		messages = append(messages, (msg.MsgType + string(msg.NodeId)))
	}
	return fmt.Sprintf("SignalType: %s, Messages: [%s]", signal.SignalType, strings.Join(messages, ", "))
}

func (m *Message) String() string {
	return fmt.Sprintf("Message{Type: %s, NodeId: %d, TimeStamp: %s}", m.MsgType, m.NodeId, m.TimeStamp.Format("2006-01-02 15:04:05"))
}

// String method for MessagesList
func (ml MessagesList) String() string {
	var sb strings.Builder
	for i, msg := range ml {
		sb.WriteString(fmt.Sprintf("Message %d: %s\n", i+1, msg.String()))
	}
	return sb.String()
}

// Signal contains the type: ping/ack signal type, the target node Ids, and the list of messages for the target node to update in their membership list (eg: FAIL 3, LEAVE 2 etc.)
func CreateSignal(signalType string, targetNodeId int, messages []*Message) *Signal {
	return &Signal{
		SignalType:   signalType,
		TargetNodeId: targetNodeId,
		Messages:     messages,
	}
}

// Message contains the message type as defined in the const above, and it also contains the node id for the particular message type (eg: Leave 2, which indicates node 2 has left)
func MessageEntry(msgType string, nodeId int) *Message {
	return &Message{
		MsgType:   msgType,
		NodeId:    nodeId,
		TimeStamp: time.Now(),
	}
}

// encoding the signal - marshalled
func EncodeSignal(signal Signal) ([]byte, error) {
	encodedBytes, err := json.Marshal(signal)
	if err != nil {
		return nil, err
	}
	return encodedBytes, nil
}

// decoding the signal
func DecodeSignal(encodedSignal []byte) (Signal, error) {
	var signal Signal
	err := json.Unmarshal(encodedSignal, &signal)
	if err != nil {
		return signal, err
	}
	return signal, err
}

// encoding the join request - marshalled
func EncodeJoinRequest(request JoinRequest) ([]byte, error) {
	encodedBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	return encodedBytes, nil
}

// decoding join response
func DecodeJoinRequest(request []byte) (JoinRequest, error) {
	var joinRequest JoinRequest
	err := json.Unmarshal(request, &joinRequest)
	if err != nil {
		return joinRequest, err
	}
	return joinRequest, err
}

// getting join response from introducer
func (ms *MembershipService) getJoinResponse() ([]byte, error) {
	joinResp := JoinResponse{MsgType: JOINRESPONSE,
		InitMembershipList: ms.MemberShipList,
	}
	resp, err := json.Marshal(joinResp)

	if err != nil {
		log.Println("Error encoding the join response: ", err)
		return nil, err
	} else {
		return resp, nil
	}
}

// to handle conflict of messages from different nodes recieved by one node
func RemoveConflictMessages(msgs []*Message) []*Message {
	// Create a map to store the latest message for each node
	latestMessages := make(map[int]*Message)

	// Iterate through the messages
	for _, msg := range msgs {
		// Check if there is already a message for this node
		if existingMsg, exists := latestMessages[msg.NodeId]; exists {
			// If the existing message is a conflict, keep the latest one based on timestamp
			if (existingMsg.MsgType == "FAILED" && msg.MsgType == "JOIN") ||
				(existingMsg.MsgType == "JOIN" && msg.MsgType == "FAILED") {
				if msg.TimeStamp.After(existingMsg.TimeStamp) {
					latestMessages[msg.NodeId] = msg
				}
			} else if msg.TimeStamp.After(existingMsg.TimeStamp) && SuspicionModeEnabled == false {
				// If no conflict, just keep the latest message
				latestMessages[msg.NodeId] = msg
			}
		} else {
			// If no message exists for this node, add the current message
			latestMessages[msg.NodeId] = msg
		}
	}

	// Create a new slice to store the latest messages
	result := make([]*Message, 0, len(latestMessages))
	for _, msg := range latestMessages {
		result = append(result, msg)
	}

	return result
}

func PrintCurrentMsgsInBuffer() {
	fmt.Printf("\n Msgs in buffer: ", currentMsgsInBuffer)
}
