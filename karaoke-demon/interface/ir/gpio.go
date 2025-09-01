package ir

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// GPIOController GPIO制御を行う構造体
type GPIOController struct {
	pin           int
	valuePath     string
	directionPath string
}

// NewGPIOController 新しいGPIO制御インスタンスを作成
func NewGPIOController(pin int) (*GPIOController, error) {
	controller := &GPIOController{
		pin:           pin,
		valuePath:     fmt.Sprintf("/sys/class/gpio/gpio%d/value", pin),
		directionPath: fmt.Sprintf("/sys/class/gpio/gpio%d/direction", pin),
	}

	// GPIOピンをエクスポート
	if err := controller.export(); err != nil {
		return nil, fmt.Errorf("failed to export GPIO %d: %w", pin, err)
	}

	// 方向を出力に設定
	if err := controller.setDirection("out"); err != nil {
		return nil, fmt.Errorf("failed to set direction for GPIO %d: %w", pin, err)
	}

	return controller, nil
}

// export GPIOピンをエクスポート
func (g *GPIOController) export() error {
	exportPath := "/sys/class/gpio/export"
	exportFile, err := os.OpenFile(exportPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer exportFile.Close()

	_, err = exportFile.WriteString(strconv.Itoa(g.pin))
	return err
}

// setDirection GPIOピンの方向を設定
func (g *GPIOController) setDirection(direction string) error {
	// 方向設定ファイルが利用可能になるまで少し待機
	time.Sleep(100 * time.Millisecond)

	file, err := os.OpenFile(g.directionPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(direction)
	return err
}

// SetHigh GPIOピンをHIGHに設定
func (g *GPIOController) SetHigh() error {
	return g.writeValue("1")
}

// SetLow GPIOピンをLOWに設定
func (g *GPIOController) SetLow() error {
	return g.writeValue("0")
}

// writeValue GPIOピンに値を書き込み
func (g *GPIOController) writeValue(value string) error {
	file, err := os.OpenFile(g.valuePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(value)
	return err
}

// Close GPIOピンをアンエクスポート
func (g *GPIOController) Close() error {
	unexportPath := "/sys/class/gpio/unexport"
	unexportFile, err := os.OpenFile(unexportPath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer unexportFile.Close()

	_, err = unexportFile.WriteString(strconv.Itoa(g.pin))
	return err
}
