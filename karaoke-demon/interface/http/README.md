# カラオケUSBツールAPI仕様

## POST /reservation

カラオケの予約を追加

### request

#### ローカルのファイルの場合

{
  "song_id": "選曲番号(数字6桁の文字列)",
  "video_type": "local",
  "video_filename": "ローカルディレクトリ内のファイル名.mp4"
}

#### ダウンロードが必要なファイルの場合

{
  "song_id": "選曲番号(数字6桁の文字列)",
  "video_type": "download",
  "video_filename": "ファイル名.mp4",
  "video_url": "https://<ダウンロード可能なmp4ファイルのリンク>"
}

### response

{
  status: "ok"
}

## GET /reservation

現在の予約状況とそれに付随するスロット情報を取得

### request

なし

### response

{
    reservations: [
        {
            "id": 3,
            "song_id": "012345",
            "video": {
                "video_status": "none|url_waiting|downloading|ready",
                "video_FileName": "xxxxx.mp4"
            },
            "slot": {
                "slot_status": "available|writing|set|locked|reading|released",
                "slot": 0
            },
            "karaoke": {
                "karaoke_status": "preparing_video|writing_to_slot|set_in_slot"
            }
        }
    ],
    length: 1,
    status: "success"
}

## GET /local-files

選曲番号から、予約時に利用可能なローカルファイルを取得

### request

クエリパラメータ

- song_id: 選曲番号(数字6桁の文字列)

### response

{
    files: [
        { name: "<選曲番号>-<タイトル>.mp4" },
        { name: "<選曲番号>-<タイトル>.mp4" }
    ],
    length: 2,
    status: "success"
}