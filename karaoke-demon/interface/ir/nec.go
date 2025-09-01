package ir

import (
	"fmt"
	"time"
)

// NECProtocol NECプロトコル制御構造体
type NECProtocol struct {
	gpio *GPIOController
}

// NECConstants NECプロトコルの定数
const (
	// タイミング定数（マイクロ秒）
	NECLeaderOnTime  = 9000 // 9ms
	NECLeaderOffTime = 4500 // 4.5ms
	NECBitOnTime     = 560  // 0.56ms
	NECBit0OffTime   = 560  // 0.56ms
	NECBit1OffTime   = 1690 // 1.69ms
	NECStopBitTime   = 560  // 0.56ms

	// データ長
	NECDataBits = 32
)

// NewNECProtocol 新しいNECプロトコル制御インスタンスを作成
func NewNECProtocol(gpioPin int) (*NECProtocol, error) {
	gpio, err := NewGPIOController(gpioPin)
	if err != nil {
		return nil, fmt.Errorf("failed to create GPIO controller: %w", err)
	}

	return &NECProtocol{
		gpio: gpio,
	}, nil
}

// SendNECData NECフォーマットでデータを送信
func (n *NECProtocol) SendNECData(address, command uint8) error {
	// 32ビットデータを構築
	data := n.buildNECData(address, command)

	// リーダー部分を送信
	if err := n.sendLeader(); err != nil {
		return fmt.Errorf("failed to send leader: %w", err)
	}

	// データ部分を送信
	if err := n.sendData(data); err != nil {
		return fmt.Errorf("failed to send data: %w", err)
	}

	// ストップビットを送信
	if err := n.sendStopBit(); err != nil {
		return fmt.Errorf("failed to send stop bit: %w", err)
	}

	return nil
}

// buildNECData NECデータを構築
func (n *NECProtocol) buildNECData(address, command uint8) uint32 {
	// NECフォーマット: [Address][Address Inverse][Command][Command Inverse]
	addressInv := ^address
	commandInv := ^command

	data := uint32(address)<<24 | uint32(addressInv)<<16 | uint32(command)<<8 | uint32(commandInv)
	return data
}

// sendLeader リーダー部分を送信
func (n *NECProtocol) sendLeader() error {
	// 9ms ON
	if err := n.gpio.SetHigh(); err != nil {
		return err
	}
	n.sleepMicroseconds(NECLeaderOnTime)

	// 4.5ms OFF
	if err := n.gpio.SetLow(); err != nil {
		return err
	}
	n.sleepMicroseconds(NECLeaderOffTime)

	return nil
}

// sendData データ部分を送信
func (n *NECProtocol) sendData(data uint32) error {
	for i := 0; i < NECDataBits; i++ {
		bit := (data >> (NECDataBits - 1 - i)) & 1
		if err := n.sendBit(bit == 1); err != nil {
			return fmt.Errorf("failed to send bit %d: %w", i, err)
		}
	}
	return nil
}

// sendBit 1ビットを送信
func (n *NECProtocol) sendBit(isOne bool) error {
	// 0.56ms ON
	if err := n.gpio.SetHigh(); err != nil {
		return err
	}
	n.sleepMicroseconds(NECBitOnTime)

	// OFF時間を決定（0または1によって異なる）
	offTime := NECBit0OffTime
	if isOne {
		offTime = NECBit1OffTime
	}

	if err := n.gpio.SetLow(); err != nil {
		return err
	}
	n.sleepMicroseconds(offTime)

	return nil
}

// sendStopBit ストップビットを送信
func (n *NECProtocol) sendStopBit() error {
	// 0.56ms ON
	if err := n.gpio.SetHigh(); err != nil {
		return err
	}
	n.sleepMicroseconds(NECStopBitTime)

	// LOWに戻す
	if err := n.gpio.SetLow(); err != nil {
		return err
	}

	return nil
}

// sleepMicroseconds マイクロ秒単位でスリープ
func (n *NECProtocol) sleepMicroseconds(us int) {
	time.Sleep(time.Duration(us) * time.Microsecond)
}

// Close リソースを解放
func (n *NECProtocol) Close() error {
	return n.gpio.Close()
}

// SendNECCommand 簡易的なNECコマンド送信（アドレス0x00を使用）
func (n *NECProtocol) SendNECCommand(command uint8) error {
	return n.SendNECData(0x00, command)
}
