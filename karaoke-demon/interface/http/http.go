package http

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"gpioblink.com/x/karaoke-demon/application"
	"gpioblink.com/x/karaoke-demon/domain/song"
)

type HttpInterface struct {
	musicService application.MusicService
	server       *http.Server
	version      string
	buildDate    string
	buildUser    string
}

// 予約リクエスト用の構造体
type ReservationRequest struct {
	SongID        string `json:"song_id"`
	VideoType     string `json:"video_type"`
	VideoFilename string `json:"video_filename"`
	VideoURL      string `json:"video_url,omitempty"`
}

// 予約レスポンス用の構造体
type ReservationResponse struct {
	Status string `json:"status"`
}

// 予約一覧レスポンス用の構造体
type ReservationListResponse struct {
	Reservations []ReservationInfo `json:"reservations"`
	Length       int               `json:"length"`
	Status       string            `json:"status"`
}

type ReservationInfo struct {
	ID      int         `json:"id"`
	SongID  string      `json:"song_id"`
	Video   VideoInfo   `json:"video"`
	Slot    SlotInfo    `json:"slot"`
	Karaoke KaraokeInfo `json:"karaoke"`
}

type VideoInfo struct {
	DownloadStatus string `json:"download_status"`
	VideoStatus    string `json:"video_status"`
	VideoFileName  string `json:"video_FileName"`
}

type SlotInfo struct {
	SlotStatus string `json:"slot_status"`
	Slot       int    `json:"slot"`
}

type KaraokeInfo struct {
	KaraokeStatus string `json:"karaoke_status"`
}

// ローカルファイルレスポンス用の構造体
type LocalFilesResponse struct {
	Files  []FileInfo `json:"files"`
	Length int        `json:"length"`
	Status string     `json:"status"`
}

type FileInfo struct {
	Name string `json:"name"`
}

// バージョン情報レスポンス用の構造体
type VersionResponse struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	BuildUser string `json:"build_user"`
	Status    string `json:"status"`
}

func NewHttpInterface(service *application.MusicService, version, buildDate, buildUser string) *HttpInterface {
	mux := http.NewServeMux()

	httpInterface := &HttpInterface{
		musicService: *service,
		version:      version,
		buildDate:    buildDate,
		buildUser:    buildUser,
		server: &http.Server{
			Addr:    ":8787",
			Handler: corsMiddleware(mux),
		},
	}

	// API endpoints
	mux.HandleFunc("/reservation", httpInterface.handleReservation)
	mux.HandleFunc("/local-files", httpInterface.handleLocalFiles)
	mux.HandleFunc("/version", httpInterface.handleVersion)

	return httpInterface
}

// corsMiddleware adds CORS headers to allow requests from any origin
func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func (h *HttpInterface) handleReservation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		h.handlePostReservation(w, r)
	case http.MethodGet:
		h.handleGetReservation(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HttpInterface) handlePostReservation(w http.ResponseWriter, r *http.Request) {
	var req ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("failed to decode reservation request: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// バリデーション
	if req.SongID == "" {
		http.Error(w, "song_id is required", http.StatusBadRequest)
		return
	}
	if req.VideoType == "" {
		http.Error(w, "video_type is required", http.StatusBadRequest)
		return
	}
	if req.VideoFilename == "" {
		http.Error(w, "video_filename is required", http.StatusBadRequest)
		return
	}

	// 選曲番号の形式チェック（6桁の数字）
	if len(req.SongID) != 6 {
		http.Error(w, "song_id must be 6 digits", http.StatusBadRequest)
		return
	}
	if _, err := strconv.Atoi(req.SongID); err != nil {
		http.Error(w, "song_id must be numeric", http.StatusBadRequest)
		return
	}

	// ビデオタイプのバリデーション
	if req.VideoType != "local" && req.VideoType != "download" {
		http.Error(w, "video_type must be 'local' or 'download'", http.StatusBadRequest)
		return
	}

	// ダウンロードタイプの場合、URLが必要
	if req.VideoType == "download" && req.VideoURL == "" {
		http.Error(w, "video_url is required for download type", http.StatusBadRequest)
		return
	}

	// ビデオ情報を作成
	videoInfo := &application.VideoInfo{
		Type:     req.VideoType,
		Filename: req.VideoFilename,
		URL:      req.VideoURL,
	}

	// ビデオ情報付きで予約処理
	err := h.musicService.ReserveSongWithVideo(song.RequestNo(req.SongID), videoInfo)
	if err != nil {
		log.Printf("failed to reserve song with video: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ReservationResponse{Status: "ok"}
	json.NewEncoder(w).Encode(response)
}

func (h *HttpInterface) handleGetReservation(w http.ResponseWriter, r *http.Request) {
	reservations, slots, err := h.musicService.GetReservationWithSlotInfo()
	if err != nil {
		log.Printf("failed to get reservations with slot info: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// レスポンス用のデータ構造に変換
	reservationInfos := make([]ReservationInfo, 0, len(reservations))
	for _, res := range reservations {
		vid := res.Video()
		so := vid.Song()

		info := ReservationInfo{
			ID:     int(res.Seq()),
			SongID: string(so.RequestNo()),
			Video: VideoInfo{
				DownloadStatus: string(vid.State()), // TODO: ダウンロード中の詳細状態を取得する
				VideoStatus:    string(vid.State()),
				VideoFileName:  string(vid.Location()),
			},
			Slot: SlotInfo{
				SlotStatus: "nodata",
				Slot:       -1,
			},
			Karaoke: KaraokeInfo{
				KaraokeStatus: string(res.State()),
			},
		}

		// 予約の状態をそのまま使用
		info.Video.DownloadStatus = string(res.State())

		// スロット情報を検索して設定
		for _, slot := range slots {
			if slot.Reservation() != nil && slot.Reservation().Seq() == res.Seq() {
				info.Slot.Slot = slot.Id()
				// スロットの状態をそのまま使用
				info.Slot.SlotStatus = string(slot.State())

				// ↓ videoはreservationの中に入っているので、ここでは取得しない
				// // VideoStatusにはvideo.goで定義されているStateをそのまま使用
				// if slot.Video() != nil {
				// 	info.Video.VideoStatus = string(slot.Video().State())
				// 	info.Video.VideoFileName = slot.Video().Location()
				// }

				break
			}
		}

		reservationInfos = append(reservationInfos, info)
	}

	response := ReservationListResponse{
		Reservations: reservationInfos,
		Length:       len(reservationInfos),
		Status:       "success",
	}

	json.NewEncoder(w).Encode(response)
}

func (h *HttpInterface) handleLocalFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	songID := r.URL.Query().Get("song_id")
	if songID == "" {
		http.Error(w, "song_id parameter is required", http.StatusBadRequest)
		return
	}

	// 選曲番号の形式チェック（6桁の数字）
	if len(songID) != 6 {
		http.Error(w, "song_id must be 6 digits", http.StatusBadRequest)
		return
	}
	if _, err := strconv.Atoi(songID); err != nil {
		http.Error(w, "song_id must be numeric", http.StatusBadRequest)
		return
	}

	// application層の機能を使用してローカルファイルを検索
	filenames, err := h.musicService.FindLocalFilesByRequestNo(songID)
	if err != nil {
		log.Printf("failed to find local files: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// ファイル名をFileInfo構造体に変換
	files := make([]FileInfo, len(filenames))
	for i, filename := range filenames {
		files[i] = FileInfo{Name: filename}
	}

	response := LocalFilesResponse{
		Files:  files,
		Length: len(files),
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HttpInterface) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := VersionResponse{
		Version:   h.version,
		BuildDate: h.buildDate,
		BuildUser: h.buildUser,
		Status:    "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HttpInterface) Run() error {
	log.Printf("Starting HTTP server on %s", h.server.Addr)
	return h.server.ListenAndServe()
}

// TODO: 終了時のシグナルハンドリングを実装
// func (h *HttpInterface) Stop() error {
// 	log.Println("Stopping HTTP server...")
// 	return h.server.Shutdown(context.Background())
// }
