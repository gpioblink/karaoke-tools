//go:build !windows

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ---- Address ranges (as provided) ----
const (
	BASE0 = 0x000000000049c800
	BASE1 = 0x000000002049c800
	BASE2 = 0x000000004049c800
	LIMIT = 0x000000006049c800 // exclusive
)

func addrToFileIdx(addr uint64) int {
	switch {
	case addr >= BASE0 && addr < BASE1:
		return 0
	case addr >= BASE1 && addr < BASE2:
		return 1
	case addr >= BASE2 && addr < LIMIT:
		return 2
	default:
		return -1
	}
}

var fileSpan = uint64(BASE1 - BASE0) // bytes

func fileBase(i int) uint64 {
	if i == 0 {
		return BASE0
	}
	if i == 1 {
		return BASE1
	}
	return BASE2
}

func sizeToLevel(sz uint64) int {
	switch {
	case sz < 4*1024:
		return 0 // small (green)
	case sz < 64*1024:
		return 1 // medium (yellow)
	default:
		return 2 // large (red)
	}
}

func fmtMiB(b uint64) string { return fmt.Sprintf("%.1f", float64(b)/1024.0/1024.0) }

// lunX: file read SIZE @ ADDR  (decimal)
var lunReadRe = regexp.MustCompile(`lun\d+:\s*file read\s*(\d+)\s*@\s*(\d+)`)

type event struct {
	ts    time.Time
	idx   int
	addr  uint64
	size  uint64
	level int
}

type ring struct {
	buf []event
	n   int
	i   int
}

func newRing(n int) *ring { return &ring{buf: make([]event, n)} }
func (r *ring) push(e event) {
	if len(r.buf) == 0 {
		return
	}
	r.buf[r.i%len(r.buf)] = e
	r.i++
	if r.n < len(r.buf) {
		r.n++
	}
}
func (r *ring) latestFirst(k int) []event {
	if r.n == 0 {
		return nil
	}
	if k > r.n {
		k = r.n
	}
	out := make([]event, 0, k)
	for c := 0; c < k; c++ {
		idx := (r.i - 1 - c + len(r.buf)) % len(r.buf)
		out = append(out, r.buf[idx])
	}
	return out
}

type model struct {
	flashMs int

	lastFlashMs [3]int64 // unix ms
	lastLevel   [3]int32
	readCount   [3]uint64
	byteSum     [3]uint64
	lastOff     [3]*uint64 // file-internal offset (bytes)
	lastAddr    [3]*uint64

	events *ring
}

func newModel(flashMs, history int) *model {
	return &model{
		flashMs: flashMs,
		events:  newRing(history),
	}
}

func (m *model) note(idx int, addr, size uint64) {
	if idx < 0 || idx >= 3 {
		return
	}
	atomic.AddUint64(&m.readCount[idx], 1)
	atomic.AddUint64(&m.byteSum[idx], size)
	atomic.StoreInt32(&m.lastLevel[idx], int32(sizeToLevel(size)))
	atomic.StoreInt64(&m.lastFlashMs[idx], time.Now().UnixMilli())

	off := addr - fileBase(idx)
	m.lastOff[idx] = &off
	m.lastAddr[idx] = &addr

	m.events.push(event{
		ts:    time.Now(),
		idx:   idx,
		addr:  addr,
		size:  size,
		level: sizeToLevel(size),
	})
}

type ui struct {
	screen      tcell.Screen
	styleText   tcell.Style
	styleLabel  tcell.Style
	styleBorder tcell.Style
	styleLvl    [3]tcell.Style
	useASCII    bool
	showAddr    bool
	recentLines int
}

func newUI(useASCII bool, showAddr bool, recent int) (*ui, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	u := &ui{
		screen:      s,
		styleText:   tcell.StyleDefault.Foreground(tcell.ColorWhite),
		styleLabel:  tcell.StyleDefault.Foreground(tcell.ColorLightCyan).Bold(true),
		styleBorder: tcell.StyleDefault.Foreground(tcell.ColorFuchsia),
		useASCII:    useASCII,
		showAddr:    showAddr,
		recentLines: recent,
	}
	u.styleLvl[0] = tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true)
	u.styleLvl[1] = tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)
	u.styleLvl[2] = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	return u, nil
}

func (u *ui) fini() {
	if u.screen != nil {
		u.screen.Fini()
	}
}

func putStr(s tcell.Screen, x, y int, st tcell.Style, str string, maxW int) {
	w, _ := s.Size()
	end := x + len([]rune(str))
	if maxW > 0 && end-x > maxW {
		// truncate
		rs := []rune(str)
		rs = rs[:maxW]
		str = string(rs)
	}
	for i, r := range str {
		if x+i >= w {
			break
		}
		s.SetContent(x+i, y, r, nil, st)
	}
}

func (u *ui) drawBox(x, y, w, h int, title string) {
	s := u.screen
	var H, V, UL, UR, LL, LR rune
	if u.useASCII {
		H, V, UL, UR, LL, LR = '-', '|', '+', '+', '+', '+'
	} else {
		H, V, UL, UR, LL, LR = '─', '│', '┌', '┐', '└', '┘'
	}

	// horizontal
	for i := 0; i < w; i++ {
		s.SetContent(x+i, y, H, nil, u.styleBorder)
		s.SetContent(x+i, y+h-1, H, nil, u.styleBorder)
	}
	// vertical
	for i := 0; i < h; i++ {
		s.SetContent(x, y+i, V, nil, u.styleBorder)
		s.SetContent(x+w-1, y+i, V, nil, u.styleBorder)
	}
	// corners
	s.SetContent(x, y, UL, nil, u.styleBorder)
	s.SetContent(x+w-1, y, UR, nil, u.styleBorder)
	s.SetContent(x, y+h-1, LL, nil, u.styleBorder)
	s.SetContent(x+w-1, y+h-1, LR, nil, u.styleBorder)

	// title
	t := " " + title + " "
	putStr(s, x+2, y, u.styleLabel, t, w-4)
}

func (u *ui) renderVertical(m *model, flashMs int) {
	s := u.screen
	s.Clear()

	W, H := s.Size()
	// narrow layout defaults
	left := 1
	top := 0
	boxW := int(math.Max(24, float64(W-2)))
	boxH := 6 // compact height per section

	// If height is tight, reduce per-box height to 5
	if top+3*boxH+6 > H {
		boxH = 5
	}

	nowMs := time.Now().UnixMilli()
	titles := [3]string{
		fmt.Sprintf("Section0[%#x-%#x)", BASE0, BASE1),
		fmt.Sprintf("Section1[%#x-%#x)", BASE1, BASE2),
		fmt.Sprintf("Section2[%#x-%#x)", BASE2, LIMIT),
	}

	y := top + 0
	for i := 0; i < 3; i++ {
		if y+boxH+1 >= H {
			break
		}
		u.drawBox(left, y, boxW, boxH, titles[i])

		active := (nowMs - atomic.LoadInt64(&m.lastFlashMs[i])) < int64(flashMs)
		level := int(atomic.LoadInt32(&m.lastLevel[i]))

		// flash fill (inner area)
		if active {
			sty := u.styleLvl[level]
			for yy := y + 1; yy < y+boxH-1; yy++ {
				for xx := left + 1; xx < left+boxW-1; xx++ {
					s.SetContent(xx, yy, ' ', nil, sty)
				}
			}
		}

		// compact text lines (short labels)
		col := left + 2
		row := y + 1
		putStr(s, col, row, u.styleText, fmt.Sprintf("R:%d  B:%d  L:%d", atomic.LoadUint64(&m.readCount[i]), atomic.LoadUint64(&m.byteSum[i]), level), boxW-4)
		row++
		if m.lastOff[i] != nil {
			putStr(s, col, row, u.styleText, fmt.Sprintf("Pos:%s/%s MiB", fmtMiB(*m.lastOff[i]), fmtMiB(fileSpan)), boxW-4)
		} else {
			putStr(s, col, row, u.styleText, "Pos:-", boxW-4)
		}
		row++
		if u.showAddr && m.lastAddr[i] != nil && row < y+boxH-1 {
			putStr(s, col, row, u.styleText, fmt.Sprintf("Addr:%#x", *m.lastAddr[i]), boxW-4)
		}
		y += boxH // stack vertically
	}

	// Legend (keep tiny)
	legend := "Legend: green<4K yellow<64K red≥64K | Pos=MiB in file"
	putStr(s, left, y+1, u.styleText, legend, W-2)

	// Recent events (compact)
	evY := y + 3
	if evY < H-1 && u.recentLines > 0 {
		putStr(s, left, evY, u.styleLabel, "Recent:", W-2)
		evY++
		for _, ev := range m.events.latestFirst(u.recentLines) {
			line := fmt.Sprintf("[%s] s=%d sz=%d", ev.ts.Format("15:04:05"), ev.idx, ev.size)
			putStr(s, left, evY, u.styleText, line, W-2)
			evY++
			if evY >= H-1 {
				break
			}
		}
	}

	s.Show()
}

// --------- /proc/kmsg reader (nonblocking) ----------
type kmsgReader struct {
	fd   int
	buf  []byte
	line bytes.Buffer
}

func newKmsgReader(path string) (*kmsgReader, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	return &kmsgReader{fd: fd, buf: make([]byte, 16*1024)}, nil
}
func (r *kmsgReader) Close() error {
	if r.fd >= 0 {
		err := syscall.Close(r.fd)
		r.fd = -1
		return err
	}
	return nil
}
func (r *kmsgReader) tryReadLines() ([]string, error) {
	n, err := syscall.Read(r.fd, r.buf)
	if n == 0 {
		if err == nil {
			return nil, nil
		}
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, nil
		}
		return nil, err
	}
	if n < 0 {
		if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, nil
		}
		return nil, err
	}

	var lines []string
	for _, b := range r.buf[:n] {
		if b == '\n' {
			lines = append(lines, r.line.String())
			r.line.Reset()
		} else {
			r.line.WriteByte(b)
		}
	}
	return lines, nil
}

// --------- stdin reader (blocking) ----------
func readStdin(ch chan<- string) {
	defer close(ch)
	sc := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024)
	for sc.Scan() {
		ch <- sc.Text()
	}
}

// --------- main ----------
func main() {
	var (
		useStdin    = flag.Bool("stdin", false, "Read from stdin (for testing)")
		flashMS     = flag.Int("flash-ms", 180, "Flash duration in ms")
		history     = flag.Int("history", 64, "Recent events to keep")
		recentLines = flag.Int("recent", 3, "Recent events lines to show")
		showAddr    = flag.Bool("show-addr", false, "Show last addr line in each section")
		asciiBorder = flag.Bool("ascii-border", false, "Use +-| ASCII border (UART safe)")
	)
	flag.Parse()

	m := newModel(*flashMS, *history)
	ui, err := newUI(*asciiBorder, *showAddr, *recentLines)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tcell init error: %v\n", err)
		os.Exit(1)
	}
	defer ui.fini()

	// input
	var (
		kmsg *kmsgReader
		chIn chan string
	)
	if *useStdin {
		chIn = make(chan string, 128)
		go readStdin(chIn)
	} else {
		kmsg, err = newKmsgReader("/proc/kmsg")
		if err != nil {
			ui.fini()
			fmt.Fprintf(os.Stderr, "open /proc/kmsg: %v (root required?)\n", err)
			os.Exit(1)
		}
		defer kmsg.Close()
	}

	// timers & key handling
	tRender := time.NewTicker(80 * time.Millisecond) // 少し抑えめ
	defer tRender.Stop()
	tPoll := time.NewTicker(50 * time.Millisecond)
	defer tPoll.Stop()

	quit := make(chan struct{})
	go func() {
		for {
			ev := ui.screen.PollEvent()
			switch tev := ev.(type) {
			case *tcell.EventKey:
				if tev.Key() == tcell.KeyCtrlC || tev.Rune() == 'q' || tev.Key() == tcell.KeyEscape {
					close(quit)
					return
				}
			case *tcell.EventResize:
				ui.screen.Sync()
			}
		}
	}()

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-tPoll.C:
			if *useStdin {
				select {
				case line, ok := <-chIn:
					if !ok {
						chIn = nil
						continue
					}
					processLine(m, line)
				default:
				}
			} else if kmsg != nil {
				lines, err := kmsg.tryReadLines()
				if err != nil && !errors.Is(err, io.EOF) {
					// ignore transient read errors
				}
				for _, ln := range lines {
					processLine(m, ln)
				}
			}
		case <-tRender.C:
			ui.renderVertical(m, *flashMS)
		}
	}
}

func processLine(m *model, line string) {
	ms := lunReadRe.FindStringSubmatch(line)
	if len(ms) != 3 {
		return
	}
	var size, addr uint64
	_, err1 := fmt.Sscan(ms[1], &size)
	_, err2 := fmt.Sscan(ms[2], &addr)
	if err1 != nil || err2 != nil {
		return
	}
	idx := addrToFileIdx(addr)
	if idx < 0 {
		return
	}
	m.note(idx, addr, size)
}
