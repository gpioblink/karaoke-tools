# 赤外線送信機能

このモジュールは、NECフォーマットの赤外線信号をGPIOピンに接続されたLEDから送信する機能を提供します。

## 機能

- NECプロトコルによる赤外線送信
- GPIO制御（/sys/class/gpio/使用）
- 組み込みLinux向けの軽量実装
- 既存システムとの統合

## 使用方法

### 1. 設定

環境変数または.envファイルでGPIOピンを設定：

```bash
IR_GPIO_PIN=18
```

### 2. FIFOインターフェース経由での使用

FIFOファイルにコマンドを送信：

```bash
# コマンド送信（アドレス0x00を使用）
echo "IR_COMMAND 02" > /tmp/karaoke-fifo

# データ送信（アドレスとコマンドを指定）
echo "IR_DATA 00 02" > /tmp/karaoke-fifo
```

### 3. プログラムからの使用

```go
import "gpioblink.com/x/karaoke-demon/interface/ir"

// 赤外線送信機を初期化
transmitter, err := ir.NewIRTransmitter(18)
if err != nil {
    log.Fatal(err)
}
defer transmitter.Close()

// コマンド送信
err = transmitter.SendCommand(0x02)

// データ送信
err = transmitter.SendData(0x00, 0x02)
```

### 4. テストプログラム

```bash
cd tool
go run ir_test.go 18
```

## NECプロトコル仕様

- キャリア周波数: 38kHz（LEDの点滅で実現）
- リーダー部分: 9ms ON + 4.5ms OFF
- データ部分: 0.56ms ON + 0.56ms OFF (0) / 0.56ms ON + 1.69ms OFF (1)
- ストップビット: 0.56ms ON

## よく使われるコマンド

```go
const (
    NEC_POWER     = 0x02  // 電源
    NEC_VOLUME_UP = 0x10  // 音量アップ
    NEC_VOLUME_DN = 0x11  // 音量ダウン
    NEC_CH_UP     = 0x20  // チャンネルアップ
    NEC_CH_DN     = 0x21  // チャンネルダウン
    NEC_0         = 0x19  // 数字0
    NEC_1         = 0x45  // 数字1
    NEC_2         = 0x46  // 数字2
    NEC_3         = 0x47  // 数字3
    NEC_4         = 0x44  // 数字4
    NEC_5         = 0x40  // 数字5
    NEC_6         = 0x43  // 数字6
    NEC_7         = 0x07  // 数字7
    NEC_8         = 0x15  // 数字8
    NEC_9         = 0x09  // 数字9
)
```

## ハードウェア接続

1. GPIOピンに赤外線LEDを接続
2. 適切な抵抗を直列に接続（通常100-330Ω）
3. LEDのアノードをGPIOピンに、カソードをGNDに接続

## 注意事項

- 組み込みLinux環境での動作を前提としています
- GPIO制御にはroot権限が必要な場合があります
- タイミング制御は`time.Sleep()`を使用しているため、高精度な制御には向いていません
- 赤外線LEDの特性に応じてタイミング調整が必要な場合があります 