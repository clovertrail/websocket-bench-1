package benchmark

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

const SignalRMessageTerminator = '\x1e'

type SignalRCoreHandshakeResp struct {
	AvailableTransports []string `json:"availableTransports"`
	ConnectionId        string   `json:"connectionId"`
}

type SignalRCommon struct {
	Type int `json:"type"`
}

type SignalRCoreInvocation struct {
	Type      int      `json:"type"`
	Target    string   `json:"target"`
	Arguments []string `json:"arguments"`
}

type MsgpackInvocation struct {
	MessageType  int32
	Header       map[string]string
	InvocationID string
	Target       string
	Params       []string
}

func (m *MsgpackInvocation) EncodeMsgpack(enc *msgpack.Encoder) error {
	enc.EncodeArrayLen(4)
	return enc.Encode(m.MessageType, m.Header, m.InvocationID, m.Target, m.Params)
}

func (m *MsgpackInvocation) DecodeMsgpack(dec *msgpack.Decoder) error {
	dec.DecodeArrayLen()
	messageType, err := dec.DecodeInt32()
	if err != nil {
		log.Printf("Failed to decode message %v\n", dec)
		return err
	}
	m.MessageType = messageType
	if messageType == 1 {
		return dec.Decode(&m.Header, &m.InvocationID, &m.Target, &m.Params)
	}
	return nil
}

type WithInterval struct {
	interval time.Duration
}

func (w *WithInterval) Interval() time.Duration {
	return w.interval
}

func appendLength(bytes []byte) []byte {
	buffer := make([]byte, 0, 5+len(bytes))
	length := len(bytes)
	for length > 0 {
		current := byte(length & 0x7F)
		length >>= 7
		if length > 0 {
			current |= 0x80
		}
		buffer = append(buffer, current)
	}
	if len(buffer) == 0 {
		buffer = append(buffer, 0)
	}
	buffer = append(buffer, bytes...)
	return buffer
}

func GenerateJsonRequest(target string, arguments []string) Message {
	msg, err := json.Marshal(&SignalRCoreInvocation{
		Type:      1,
		Target:    target,
		Arguments: arguments,
	})
	if err != nil {
		log.Println("ERROR: failed to encoding SignalR message", err)
		return nil
	}
	msg = append(msg, SignalRMessageTerminator)
	return PlainMessage{websocket.TextMessage, msg}
}

func GenerateMessagePackRequest(target string, arguments []string) Message {
	invocation := MsgpackInvocation{
		MessageType: 1,
		Header:      map[string]string{},
		Target:      target,
		Params:      arguments,
	}
	msg, err := msgpack.Marshal(&invocation)
	if err != nil {
		log.Fatalln("Fail to pack signalr core message", err)
		return nil
	}
	msg = appendLength(msg)
	return PlainMessage{websocket.BinaryMessage, msg}
}

type SignalRCoreTextMessageGenerator struct {
	WithInterval
	Target string
}

var _ MessageGenerator = (*SignalRCoreTextMessageGenerator)(nil)

func (g *SignalRCoreTextMessageGenerator) Generate(uid string, groupName string, invocationId int64) Message {
	arguments := []string{
		uid,
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	return GenerateJsonRequest(g.Target, arguments)
}

type JsonGroupSendMessageGenerator struct {
	WithInterval
	Target string
}

var _ MessageGenerator = (*JsonGroupSendMessageGenerator)(nil)

func (g *JsonGroupSendMessageGenerator) Generate(uid string, groupName string, invocationId int64) Message {
	arguments := []string{
		groupName,
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	return GenerateJsonRequest(g.Target, arguments)
}

type MessagePackMessageGenerator struct {
	WithInterval
	Target string
}

var _ MessageGenerator = (*MessagePackMessageGenerator)(nil)

func (g *MessagePackMessageGenerator) Generate(uid string, groupName string, invocationId int64) Message {
	params := []string{
		uid,
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	return GenerateMessagePackRequest(g.Target, params)
}

type MessagePackGroupSendMessageGenerator struct {
	WithInterval
	Target string
}

var _ MessageGenerator = (*MessagePackGroupSendMessageGenerator)(nil)

func (g *MessagePackGroupSendMessageGenerator) Generate(uid string, groupName string, invocationId int64) Message {
	params := []string{
		groupName,
		strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	return GenerateMessagePackRequest(g.Target, params)
}
