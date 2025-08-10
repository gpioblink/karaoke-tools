package eventbus

// イベント名の定数
const (
	EventReservationCreated    = "reservation.created"
	EventReservationDequeued   = "reservation.dequeued"
	EventSlotReadingAdvanced   = "slot.reading.advanced"
	EventSlotAvailableAppeared = "slot.available.appeared"
	EventVideoURLReceived      = "video.url.received"
	EventVideoDownloaded       = "video.downloaded"
	EventVideoAttachedToSlot   = "video.attached"
	EventKaraokeStarted        = "karaoke.started"
	EventKaraokeFinished       = "karaoke.finished"
)

// ReservationCreated は新規予約
type ReservationCreated struct {
	SongID        string
	WithVideoURL  string
	VideoTitle    string
	ForceOverride bool
}

func (e ReservationCreated) Name() string { return EventReservationCreated }

// ReservationDequeued は先頭予約が消化された
type ReservationDequeued struct {
	PrevReservationID int
}

func (e ReservationDequeued) Name() string { return EventReservationDequeued }

// SlotReadingAdvanced は読み取り中スロットが進んだ
type SlotReadingAdvanced struct {
	ReadingSlotID int
	TotalSlots    int
}

func (e SlotReadingAdvanced) Name() string { return EventSlotReadingAdvanced }

// SlotAvailableAppeared は新たに available になったスロット
type SlotAvailableAppeared struct {
	SlotID int
}

func (e SlotAvailableAppeared) Name() string { return EventSlotAvailableAppeared }

// VideoURLReceived はWebhook等で動画URLが届いた
type VideoURLReceived struct {
	ReservationID int
	SongID        string
	VideoURL      string
	VideoTitle    string
	ForceOverride bool
}

func (e VideoURLReceived) Name() string { return EventVideoURLReceived }

// VideoDownloaded は動画の保存完了
type VideoDownloaded struct {
	ReservationID int
	LocalPath     string
}

func (e VideoDownloaded) Name() string { return EventVideoDownloaded }

// VideoAttachedToSlot はスロットに動画が差し替えられた
type VideoAttachedToSlot struct {
	SlotID int
}

func (e VideoAttachedToSlot) Name() string { return EventVideoAttachedToSlot }

// KaraokeStarted/Finished は再生状態

type KaraokeStarted struct {
	ReservationID int
	SlotID        int
}

func (e KaraokeStarted) Name() string { return EventKaraokeStarted }

type KaraokeFinished struct {
	ReservationID int
	SlotID        int
}

func (e KaraokeFinished) Name() string { return EventKaraokeFinished }
