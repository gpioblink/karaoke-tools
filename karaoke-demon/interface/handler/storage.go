package handler

import (
	"context"
	"log"
	"strconv"

	"gpioblink.com/x/karaoke-demon/application"
)

var UpdateReading HandlerFunc = func(ctx context.Context, service application.MusicService, req Request) {
	if len(req.params) != 2 {
		return
	}

	addrStr := req.params[0]
	addr, err := strconv.ParseUint(addrStr, 10, 64)
	if err != nil {
		return
	}

	// lengthStr := req.params[1]
	// length, err := strconv.ParseUint(lengthStr, 10, 64)
	// if err != nil {
	// 	return
	// }

	// TODO: ハードコードによる決め打ちをなくす
	// アドレスを元にファイル番号を特定
	fileIdx := -1
	if addr >= 0x000000000049c800 && addr < 0x000000002049c800 {
		fileIdx = 0
	} else if addr >= 0x000000002049c800 && addr < 0x000000004049c800 {
		fileIdx = 1
	} else if addr >= 0x000000004049c800 && addr < 0x000000006049c800 {
		fileIdx = 2
	} else {
		return
	}

	err = service.UpdateSlotStateReadingByReadingSlotId(fileIdx)
	if err != nil {
		log.Fatalf("failed to update reading: %v", err)
		return
	}
}
