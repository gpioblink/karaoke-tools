#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <linux/lirc.h>
#include <stdint.h>
#include <string.h>
#include <errno.h>
#include <sys/stat.h>

#define DEVICE_PATH "/dev/lirc1"
#define FIFO_PATH "/tmp/karaoke-fifo"

void receiveIRSignals() {
    int lirc_fd, fifo_fd, res;
    struct lirc_scancode scancode;
    uint32_t mode = LIRC_MODE_SCANCODE;
    uint32_t features = LIRC_CAN_REC_SCANCODE;

    printf("LIRC opened...\n");

    // LIRCデバイスを開く
    lirc_fd = open(DEVICE_PATH, O_RDWR);
    if (lirc_fd == -1) {
        perror("Failed to open LIRC device");
        exit(EXIT_FAILURE);
    }

    // printf("FIFO opened...\n");

    // // FIFOファイルを開く
    // fifo_fd = open(FIFO_PATH, O_WRONLY);
    // if (fifo_fd == -1) {
    //     perror("Failed to open FIFO");
    //     close(lirc_fd);
    //     exit(EXIT_FAILURE);
    // }

    // LIRC_MODE_SCANCODEに対応するか表示
    if (ioctl(lirc_fd, LIRC_GET_FEATURES, &features) == -1) {
        perror("LIRC_CAN_REC_SCANCODE is not supported");
        exit(EXIT_FAILURE);
    }

    // 受信モードをLIRC_MODE_SCANCODEに設定
    res = ioctl(lirc_fd, LIRC_SET_REC_MODE, &mode);

    switch (res) {
        case ENODEV:
            fprintf(stderr, "Error: Device not available.\n");
            perror("Failed to set LIRC mode to SCANCODE");
            close(lirc_fd);
            close(fifo_fd);
            exit(EXIT_FAILURE);
            break;
        case ENOTTY:
            fprintf(stderr, "Error: Device does not support receiving.\n");
            perror("Failed to set LIRC mode to SCANCODE");
            close(lirc_fd);
            close(fifo_fd);
            exit(EXIT_FAILURE);
            break;
        case EINVAL:
            fprintf(stderr, "Error: Invalid mode or invalid mode for this device.\n");
            perror("Failed to set LIRC mode to SCANCODE");
            close(lirc_fd);
            close(fifo_fd);
            exit(EXIT_FAILURE);
            break;
    }


    printf("LIRC mode set to LIRC_MODE_SCANCODE. Waiting for NEC codes...\n");

    // 受信ループ
    while (1) {
        printf("reading...\n");

        // LIRCからスキャンコードを受信する
        if (read(lirc_fd, &scancode, sizeof(scancode)) != sizeof(scancode)) {
            perror("Failed to read from LIRC device");
            close(lirc_fd);
            close(fifo_fd);
            exit(EXIT_FAILURE);
        }

        // 受信したスキャンコードをFIFOに書き込む
        char output[128];
        int len = snprintf(output, sizeof(output), 
                           "Received NEC code - Scancode: 0x%llx, Flags: 0x%x\n", 
                           scancode.scancode, scancode.flags);

        // if (write(fifo_fd, output, len) == -1) {
        //     perror("Failed to write to FIFO");
        //     close(lirc_fd);
        //     close(fifo_fd);
        //     exit(EXIT_FAILURE);
        // }

        printf("%s", output);
    }

    close(lirc_fd);
    close(fifo_fd);
}

int main() {
    // FIFOファイルを作成
    if (mkfifo(FIFO_PATH, 0666) == -1 && errno != EEXIST) {
        perror("Failed to create FIFO");
        exit(EXIT_FAILURE);
    }

    receiveIRSignals();

    return 0;
}
