package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ANSI color codes (variables so Init() can clear them)
var (
	useColors = true

	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	BrightRed = "\033[91m"
	BrightGreen = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue = "\033[94m"
	BrightCyan = "\033[96m"
)

func Init() {
	if !vtEnabled() {
		useColors = false
		Reset = ""
		Bold = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Magenta = ""
		Cyan = ""
		White = ""
		BrightRed = ""
		BrightGreen = ""
		BrightYellow = ""
		BrightBlue = ""
		BrightCyan = ""
	}
}

type Bar struct {
	mu           sync.Mutex
	prefix       string
	total        int64
	current      int64
	width        int
	done         bool
	failed       bool
	start        time.Time
	lastDraw     string
	lastDrawTime time.Time
	hideSpeed    bool
	minInterval  time.Duration
}

func (b *Bar) HideSpeed() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hideSpeed = true
}

func New(prefix string, total int64) *Bar {
	return &Bar{
		prefix:      prefix,
		total:       total,
		width:       30,
		start:       time.Now(),
		minInterval: 50 * time.Millisecond,
	}
}

func (b *Bar) SetCurrent(n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if n > b.total {
		n = b.total
	}
	b.current = n
	b.draw()
}

func (b *Bar) Add(n int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current += n
	if b.current > b.total {
		b.current = b.total
	}
	b.draw()
}

func (b *Bar) Done() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.current = b.total
	b.render()
	b.done = true
	fmt.Println()
}

func (b *Bar) Fail() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.done = true
	b.failed = true
	fmt.Printf("\r%s %s[FAILED]%s %s\n", b.prefix, Red, Reset, b.prefix)
}

func (b *Bar) draw() {
	if b.done {
		return
	}
	if time.Since(b.lastDrawTime) < b.minInterval {
		return
	}
	b.lastDrawTime = time.Now()
	b.render()
}

func barColor(pct float64) string {
	switch {
	case pct < 25:
		return BrightRed
	case pct < 50:
		return BrightYellow
	case pct < 75:
		return Yellow
	default:
		return BrightGreen
	}
}

func (b *Bar) render() {
	pct := float64(0)
	if b.total > 0 {
		pct = float64(b.current) / float64(b.total) * 100
	}
	filled := int(float64(b.width) * pct / 100)
	if filled > b.width {
		filled = b.width
	}

	color := barColor(pct)
	bar := color + strings.Repeat("=", filled) + White + strings.Repeat(" ", b.width-filled) + Reset

	elapsed := time.Since(b.start).Round(time.Second)
	speed := ""
	if !b.hideSpeed && elapsed > 0 && b.current > 0 {
		bytesPerSec := float64(b.current) / elapsed.Seconds()
		if bytesPerSec > 1024*1024 {
			speed = fmt.Sprintf(" %s%.1f MB/s%s", BrightYellow, bytesPerSec/(1024*1024), Reset)
		} else if bytesPerSec > 1024 {
			speed = fmt.Sprintf(" %s%.0f KB/s%s", BrightYellow, bytesPerSec/1024, Reset)
		} else {
			speed = fmt.Sprintf(" %s%.0f B/s%s", BrightYellow, bytesPerSec, Reset)
		}
	}

	pctStr := fmt.Sprintf("%.1f%%", pct)
	// Truncate prefix to fit within 80-column terminal
	prefix := b.prefix
	extraLen := b.width + len(pctStr) + 4 + len(speed)
	if extraLen > 78 {
		extraLen = 78
	}
	maxPrefix := 80 - extraLen
	if maxPrefix < 10 {
		maxPrefix = 10
	}
	if len(prefix) > maxPrefix {
		prefix = prefix[:maxPrefix-3] + "..."
	}

	var line string
	if useColors {
		line = fmt.Sprintf("\033[1G\033[K%s %s[%s]%s %s%s%s%s",
			Bold+prefix+Reset, Cyan, bar, Reset,
			Bold, pctStr, Reset, speed)
	} else {
		plainBar := strings.Repeat("=", filled) + strings.Repeat(" ", b.width-filled)
		line = fmt.Sprintf("\r%s [%s] %s%s", prefix, plainBar, pctStr, speed)
	}

	if line != b.lastDraw {
		fmt.Print(line)
		b.lastDraw = line
	}
}

type Spinner struct {
	mu      sync.Mutex
	prefix  string
	done    bool
	failed  bool
	quit    chan struct{}
	frames  []string
	colors  []string
}

func NewSpinner(prefix string) *Spinner {
	return &Spinner{
		prefix: prefix,
		frames: []string{"|", "/", "-", "\\"},
		colors: []string{Cyan, Magenta, Blue, BrightCyan},
		quit:   make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	go func() {
		i := 0
		for {
			select {
			case <-s.quit:
				return
			default:
				s.mu.Lock()
				if !s.done {
					fmt.Printf("\r%s%s %s%s%s ", s.colors[i%len(s.colors)], s.frames[i%len(s.frames)], Bold+s.prefix+Reset, s.colors[i%len(s.colors)], Reset)
				}
				s.mu.Unlock()
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Done(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	close(s.quit)
	if msg != "" {
		fmt.Printf("\r%s%s[OK]%s %s%s%s %s\n", Green, Bold, Reset, Bold, s.prefix, Reset, msg)
	} else {
		fmt.Printf("\r%s%s[OK]%s %s%s%s\n", Green, Bold, Reset, Bold, s.prefix, Reset)
	}
}

func (s *Spinner) Fail(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
	s.failed = true
	close(s.quit)
	fmt.Printf("\r%s%s[FAIL]%s %s%s%s %s\n", Red, Bold, Reset, Bold, s.prefix, Reset, msg)
}
