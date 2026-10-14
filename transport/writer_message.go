package transport

import (
	"google.golang.org/protobuf/proto"
)

func (w *Writer) SendMsg(message proto.Message) {
	if message == nil {
		return
	}
	payload, err := proto.Marshal(message)
	if err == nil {
		_, err = w.Write(payload)
	}
	if err != nil {
		Info("%v", err)
	}
}
