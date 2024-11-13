package slot

import (
	"fmt"
	"os"
	"os/exec"

	"gpioblink.com/x/karaoke-demon/domain/reservation"
	"gpioblink.com/x/karaoke-demon/domain/slot"
	"gpioblink.com/x/karaoke-demon/domain/video"
)

type FatRepository struct {
	memoryRepository *MemoryRepository
	imagePath        string
	dummyFilePath    string
}

const IMAGE_SIZE = "1.8GiB"
const VIDEO_EXT = "mp4"
const VIDEO_NUM = "3"
const VIDEO_SIZE = "512MiB"

func NewFatRepository(imagePath string, dummyFilePath string) (*FatRepository, error) {
	fmt.Println("Setup image file...")

	// イメージファイルが存在する場合は削除
	if Exists(imagePath) {
		os.Remove(imagePath)
	}

	// makemyfatコマンドにより空イメージファイルの作成
	// makemyfat create test1.img 2GiB mp4 3 512MiB 1

	//print command
	fmt.Println("Execute:", "makemyfat", "create", imagePath, IMAGE_SIZE, VIDEO_EXT, VIDEO_NUM, VIDEO_SIZE, "1")

	if err := exec.Command("makemyfat", "create",
		imagePath, IMAGE_SIZE, VIDEO_EXT, VIDEO_NUM, VIDEO_SIZE, "1").Run(); err != nil {
		// イメージファイルの作成に失敗した場合はエラーを出力
		fmt.Println("Failed to create image file.")
		return nil, err
	}

	fmt.Println("Insert initial video files...")

	// ビデオ数の分だけダミーファイルを書き込み
	for i := 0; i < 3; i++ {
		if err := exec.Command("makemyfat", "insert",
			imagePath, dummyFilePath, fmt.Sprintf("%d", i)).Run(); err != nil {
			// イメージファイルの追加に失敗した場合はエラーを出力
			fmt.Println("Failed to insert video.")
			return nil, err
		}
	}

	// Memory実装の拡張として実装する
	return &FatRepository{
		memoryRepository: NewMemoryRepository(dummyFilePath),
		imagePath:        imagePath,
		dummyFilePath:    dummyFilePath,
	}, nil
}

func (f *FatRepository) Len() int {
	return f.memoryRepository.Len()
}

func (f *FatRepository) AttachReservationById(slotId int, reservation *reservation.Reservation) error {
	err := f.memoryRepository.AttachReservationById(slotId, reservation)
	if err != nil {
		return err
	}

	return nil
}

func (f *FatRepository) ChangeVideoById(slotId int, video *video.Video) error {
	// FIXME: 書き込みに失敗した場合のリカバリ処理が必要
	err := f.writeVideo(slotId, video)
	if err != nil {
		return err
	}
	err = f.memoryRepository.ChangeVideoById(slotId, video)
	if err != nil {
		return err
	}
	return nil
}

func (f *FatRepository) GetFirstSlotByState(state slot.State) (*slot.Slot, error) {
	return f.memoryRepository.GetFirstSlotByState(state)
}

func (f *FatRepository) DettachReservationById(slotId int) error {
	return f.memoryRepository.DettachReservationById(slotId)
}

func (f *FatRepository) SetStateById(slotId int, state slot.State) error {
	return f.memoryRepository.SetStateById(slotId, state)
}

func (f *FatRepository) SetWritingFlagById(slotId int, isWriting bool) error {
	return f.memoryRepository.SetWritingFlagById(slotId, isWriting)
}

func (f *FatRepository) SetSeqById(slotId int, seq int) error {
	return f.memoryRepository.SetSeqById(slotId, seq)
}

func (f *FatRepository) FindById(id int) (*slot.Slot, error) {
	return f.memoryRepository.FindById(id)
}

func (f *FatRepository) List() ([]*slot.Slot, error) {
	return f.memoryRepository.List()
}

func (f *FatRepository) writeVideo(slotId int, video *video.Video) error {
	if video == nil {
		return fmt.Errorf("video is nil")
	}

	err := f.memoryRepository.SetWritingFlagById(slotId, true)
	if err != nil {
		return err
	}

	// mkmyfatを利用して、ビデオの書き込み作業を行う
	fmt.Println("Execute:", "makemyfat", "insert", f.imagePath, video.Location(), fmt.Sprintf("%d", slotId))
	if err := exec.Command("makemyfat", "insert",
		f.imagePath, video.Location(), fmt.Sprintf("%d", slotId)).Run(); err != nil {
		// イメージファイルの追加に失敗した場合はエラーを出力
		fmt.Printf("Failed to insert video %s to fileNo %d.\n", video.Location(), slotId)
		return err
	}

	err = f.memoryRepository.SetWritingFlagById(slotId, false)
	if err != nil {
		return err
	}

	return nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
