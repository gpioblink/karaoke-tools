# カラオケUSBツールAPI仕様

## POST /reservation

### request

{
    "song_id": "012345", //必須
    "video": {
        "videoUrl": "https://~~~~.mp4",
        "videoTitle": "曲名MAD",
        "forceOverride"： false // ローカルに動画がダウンロード済みでも、ここで指定されたものを優先して使う
    }
}

### response

{
  status: "success", message: "曲の予約に成功しました"
}

## GET /reservation

{
    reservations: [
        {
            "id": 3,
            "song_id": "012345",
            "song_name": "あいうえお",
            "video": {
                "download_status": "waiting|downloading|downloaded|failed|none",
                "videoStatus": "downloaded-video|local-video|none",
                "videoPath": "/home/root/xxxxx"
                "videoUrl": "https://~~~~.mp4",
                "videoTitle": "曲名MAD",
            },
            "slot": {
                "slot_status": "waiting|allocated|wont_allocated|failed",
                "slot": 0
            },
            "karaoke": {
                "karaoke_status": "wait|playing|played"
            }
        }
    ],
    length: 1,
    status: "success"
}
