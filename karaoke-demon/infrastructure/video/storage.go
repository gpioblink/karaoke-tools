package video

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gpioblink.com/x/karaoke-demon/domain/song"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

type StorageRepository struct {
	basePath        string
	fillerVideoPath string
}

func NewStorageRepository(basePath string, fillerVideoPath string) *StorageRepository {
	return &StorageRepository{
		basePath:        basePath,
		fillerVideoPath: fillerVideoPath,
	}
}

func (s *StorageRepository) FindByRequestNo(requestNo string) (*video.Video, error) {
	// 選曲番号から曲情報を取得
	songInfo, err := song.NewSongInfo(requestNo)
	if err != nil {
		return nil, err
	}
	// 動画ディレクトリ内の選曲番号から始まるファイル名の曲を探す
	filePath, err := findFileWithPrefix(s.basePath, requestNo)

	if err != nil {
		filePath = s.fillerVideoPath
	}

	v, videoErr := video.NewVideo(songInfo, filePath)
	if videoErr != nil {
		return nil, videoErr
	}
	return v, nil // TODO: ビデオが存在しない場合の返し方を検討
}

func findFileWithPrefix(dir string, prefix string) (string, error) {
	fmt.Printf("[FindFileWithPrefix] dir: %s, prefix: %s\n", dir, prefix)
	// ディレクトリ内のファイルを取得
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	// ファイル名がprefixから始まるファイルを探す
	for _, entry := range files {
		if strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("file starting with %s is not found", prefix)
}
