// this is mock implementation of song repository. Data is not really stored.
// TODO: get song information from the Internet

package song

import "gpioblink.com/x/karaoke-demon-clean/domain/song"

type MemoryRepository struct {
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{}
}

func (m *MemoryRepository) FindByRequestNo(requestNo string) (*song.Song, error) {
	songInfo, err := song.NewSongInfo(requestNo)
	if err != nil {
		return nil, err
	}
	return songInfo, nil
}
