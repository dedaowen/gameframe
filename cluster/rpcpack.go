package cluster

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/dedaowen/gameframe/fnet"
	"github.com/dedaowen/gameframe/iface"
	"github.com/dedaowen/gameframe/logger"
)

type RpcData struct {
	MsgType RpcSignal              `json:"msgtype"`
	Key     string                 `json:"key,omitempty"`
	Target  string                 `json:"target,omitempty"`
	Args    []interface{}          `json:"args,omitempty"`
	Result  map[string]interface{} `json:"result,omitempty"`
}

type RpcPackege struct {
	Len  int32
	Data []byte
}

type RpcRequest struct {
	Fconn   iface.IWriter
	Rpcdata *RpcData
}

type RpcDataPack struct{}

func NewRpcDataPack() *RpcDataPack {
	return &RpcDataPack{}
}

func (this *RpcDataPack) GetHeadLen() int32 {
	return 4
}

func (this *RpcDataPack) Unpack(headdata []byte) (interface{}, error) {
	headbuf := bytes.NewReader(headdata)

	rp := &RpcPackege{}

	// 读取Len
	if err := binary.Read(headbuf, binary.LittleEndian, &rp.Len); err != nil {
		return nil, err
	}

	// 封包太大
	if rp.Len > fnet.MaxPacketSize {
		return nil, errors.New("rpc packege too big!!!")
	}

	return rp, nil
}

//func (this *RpcDataPack) Pack(msgId uint32, pkg interface{}) (out []byte, err error) {
//	outbuff := bytes.NewBuffer([]byte{})
//	// 进行编码
//	dataBytes := []byte{}
//	data := pkg.(*RpcData)
//	if data != nil {
//		dataBytes, err = json.Marshal(data)
//	}
//
//	if err != nil {
//		fmt.Println(fmt.Sprintf("json marshaling error:  %s", err))
//	}
//	// 写Len
//	if err = binary.Write(outbuff, binary.LittleEndian, uint32(len(dataBytes))); err != nil {
//		return
//	}
//
//	//all pkg data
//	if err = binary.Write(outbuff, binary.LittleEndian, dataBytes); err != nil {
//		return
//	}
//
//	out = outbuff.Bytes()
//	return
//
//}

func (this *RpcDataPack) Pack(msgId uint32, pkg interface{}) (out []byte, err error) {
	outbuff := bytes.NewBuffer([]byte{})
	// 进行编码
	databuff := bytes.NewBuffer([]byte{})
	data := pkg.(*RpcData)
	if data != nil {
		enc := gob.NewEncoder(databuff)
		err = enc.Encode(data)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("rpcpack gob marshaling error:  %s", err))
		return
	}
	// 写Len
	if err = binary.Write(outbuff, binary.LittleEndian, uint32(databuff.Len())); err != nil {
		return
	}

	//all pkg data
	if err = binary.Write(outbuff, binary.LittleEndian, databuff.Bytes()); err != nil {
		return
	}

	out = outbuff.Bytes()
	return

}
