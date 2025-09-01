package ir

import (
	"log"
	"strconv"
)

// IRTransmitter 赤外線送信インターフェース
type IRTransmitter struct {
	nec *NECProtocol
}

// NewIRTransmitter 新しい赤外線送信インスタンスを作成
func NewIRTransmitter(gpioPin int) (*IRTransmitter, error) {
	nec, err := NewNECProtocol(gpioPin)
	if err != nil {
		return nil, err
	}

	return &IRTransmitter{
		nec: nec,
	}, nil
}

// SendCommand 赤外線コマンドを送信
func (i *IRTransmitter) SendCommand(command uint8) error {
	log.Printf("Sending IR command: 0x%02X", command)
	return i.nec.SendNECCommand(command)
}

// SendData アドレスとコマンドを指定して赤外線データを送信
func (i *IRTransmitter) SendData(address, command uint8) error {
	log.Printf("Sending IR data - Address: 0x%02X, Command: 0x%02X", address, command)
	return i.nec.SendNECData(address, command)
}

// SendCommandString 文字列として渡されたコマンドを送信
func (i *IRTransmitter) SendCommandString(commandStr string) error {
	command, err := strconv.ParseUint(commandStr, 16, 8)
	if err != nil {
		return err
	}
	return i.SendCommand(uint8(command))
}

// SendDataString 文字列として渡されたアドレスとコマンドを送信
func (i *IRTransmitter) SendDataString(addressStr, commandStr string) error {
	address, err := strconv.ParseUint(addressStr, 16, 8)
	if err != nil {
		return err
	}

	command, err := strconv.ParseUint(commandStr, 16, 8)
	if err != nil {
		return err
	}

	return i.SendData(uint8(address), uint8(command))
}

// Close リソースを解放
func (i *IRTransmitter) Close() error {
	return i.nec.Close()
}
