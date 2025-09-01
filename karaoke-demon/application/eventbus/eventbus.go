package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Event はイベント駆動の基本インターフェース
// 可能な限り軽量にするため Name のみを必須とする
// ペイロードは各イベント構造体のフィールドで表現する
// ハンドラは型アサーションでペイロードを参照する

type Event interface {
	Name() string
}

// Handler は単一イベントの処理関数
// エラーはログ側で扱う前提のため戻り値は設けない

type Handler func(ctx context.Context, e Event)

// EventBus はイベントの購読/発火を扱う
// 実装は InMemoryEventBus を提供する

type EventBus interface {
	Publish(ctx context.Context, e Event)
	Subscribe(eventName string, h Handler)
}

// InMemoryEventBus は goroutine セーフな簡易実装
// - 同期 Publish: ハンドラ毎に goroutine で非同期実行
// - Best-effort 実行: ハンドラ内のパニックは握りつぶさず上位に伝播しない

type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
	wg       sync.WaitGroup
}

func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{handlers: make(map[string][]Handler)}
}

func (b *InMemoryEventBus) Publish(ctx context.Context, e Event) {
	// イベント発火時のログ出力
	eventData, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("[EventBus] イベント発火: %s (JSON化エラー: %v)\n", e.Name(), err)
	} else {
		fmt.Printf("[EventBus] イベント発火: %s, 引数: %s\n", e.Name(), string(eventData))
	}

	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[e.Name()]...)
	b.mu.RUnlock()
	for _, h := range hs {
		b.wg.Add(1)
		go func(fn Handler) {
			defer b.wg.Done()
			fn(ctx, e)
		}(h)
	}
}

func (b *InMemoryEventBus) Subscribe(eventName string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], h)
}

// Wait は発火済みの全イベント処理完了を待機する
// テストやクリーンシャットダウン用
func (b *InMemoryEventBus) Wait() {
	b.wg.Wait()
}
