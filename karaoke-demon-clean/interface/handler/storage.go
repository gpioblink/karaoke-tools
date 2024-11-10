package handler

import (
	"context"
	"log"
	"strconv"

	"gpioblink.com/x/karaoke-demon-clean/application"
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

	/*
		$ makemyfat create test1.img 2GiB mp4 3 512MiB 1
		imagePath test1.img, fileSize 2147483648, fileExt mp4, numOfFiles 3, eachFileSize 536870912, isMBR true
		***** Root File List (MBR Shifted) *****
		0       MP4[536870912bytes]: LBA 0x00002814-0x00102814 0x0000000000502800-0x0000000020502800 clus=3
		1       MP4[536870912bytes]: LBA 0x00102814-0x00202814 0x0000000020502800-0x0000000040502800 clus=131075
		2       MP4[536870912bytes]: LBA 0x00202814-0x00302814 0x0000000040502800-0x0000000060502800 clus=262147
	*/

	// TODO: ハードコードによる決め打ちをなくす
	// アドレスを元にファイル番号を特定
	fileIdx := -1
	if addr >= 0x0000000000502800 && addr < 0x0000000020502800 {
		fileIdx = 0
	} else if addr < 0x0000000040502800 {
		fileIdx = 1
	} else if addr < 0x0000000060502800 {
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
