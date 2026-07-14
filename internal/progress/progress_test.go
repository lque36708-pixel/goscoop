package progress

import (
	"os"
	"testing"
	"time"
)

var (
	saveUseColors     bool
	saveReset, saveBold, saveRed, saveGreen, saveYellow  string
	saveBlue, saveMagenta, saveCyan, saveWhite           string
	saveBrightRed, saveBrightGreen, saveBrightYellow      string
	saveBrightBlue, saveBrightCyan                       string
)

func saveState() {
	saveUseColors = useColors
	saveReset, saveBold = Reset, Bold
	saveRed, saveGreen, saveYellow = Red, Green, Yellow
	saveBlue, saveMagenta, saveCyan, saveWhite = Blue, Magenta, Cyan, White
	saveBrightRed, saveBrightGreen, saveBrightYellow = BrightRed, BrightGreen, BrightYellow
	saveBrightBlue, saveBrightCyan = BrightBlue, BrightCyan
}

func restoreState() {
	useColors = saveUseColors
	Reset, Bold = saveReset, saveBold
	Red, Green, Yellow = saveRed, saveGreen, saveYellow
	Blue, Magenta, Cyan, White = saveBlue, saveMagenta, saveCyan, saveWhite
	BrightRed, BrightGreen, BrightYellow = saveBrightRed, saveBrightGreen, saveBrightYellow
	BrightBlue, BrightCyan = saveBrightBlue, saveBrightCyan
}

func TestInitRespectsNoColor(t *testing.T) {
	saveState()
	defer restoreState()
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")
	useColors = true
	Init()
	if useColors {
		t.Error("Init should disable colors when NO_COLOR is set")
	}
}

func TestInitNoNoColor(t *testing.T) {
	saveState()
	defer restoreState()
	os.Unsetenv("NO_COLOR")
	useColors = false
	Init()
}

func TestBarLifecycle(t *testing.T) {
	saveState()
	defer restoreState()
	useColors = true
	b := New("test", 100)
	if b == nil {
		t.Fatal("New returned nil")
	}
	b.SetCurrent(50)
	b.Add(50)
	b.Done()
}

func TestBarFail(t *testing.T) {
	saveState()
	defer restoreState()
	b := New("fail", 10)
	b.Fail()
}

func TestBarHideSpeed(t *testing.T) {
	saveState()
	defer restoreState()
	b := New("test", 100)
	b.HideSpeed()
	b.SetCurrent(25)
	b.Done()
}

func TestSpinnerLifecycle(t *testing.T) {
	saveState()
	defer restoreState()
	s := NewSpinner("testing")
	if s == nil {
		t.Fatal("NewSpinner returned nil")
	}
	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Done("complete")
}

func TestSpinnerFail(t *testing.T) {
	saveState()
	defer restoreState()
	s := NewSpinner("failtask")
	s.Start()
	time.Sleep(20 * time.Millisecond)
	s.Fail("error msg")
}

func TestSpinnerPrefix(t *testing.T) {
	saveState()
	defer restoreState()
	s := NewSpinner("myprefix")
	if s.prefix != "myprefix" {
		t.Errorf("spinner prefix = %q, want %q", s.prefix, "myprefix")
	}
}

func TestColorVariablesHaveDefaults(t *testing.T) {
	saveState()
	defer restoreState()
	// Manually reset to initial values (don't call Init)
	useColors = true
	Reset = "\033[0m"
	Bold = "\033[1m"
	Red = "\033[31m"
	Green = "\033[32m"
	Yellow = "\033[33m"
	Blue = "\033[34m"
	Magenta = "\033[35m"
	Cyan = "\033[36m"
	White = "\033[37m"
	BrightRed = "\033[91m"
	BrightGreen = "\033[92m"
	BrightYellow = "\033[93m"
	BrightBlue = "\033[94m"
	BrightCyan = "\033[96m"
	if Reset == "" || Bold == "" || Green == "" {
		t.Error("color variables should not be empty")
	}
}

func TestBarFormatting(t *testing.T) {
	saveState()
	defer restoreState()
	useColors = true
	b := New("short", 100)
	b.SetCurrent(50)
	b.render()
}

func TestBarNoColors(t *testing.T) {
	saveState()
	defer restoreState()
	useColors = false
	b := New("nocolor", 50)
	b.SetCurrent(25)
	b.render()
}

func TestSpinnerNoColors(t *testing.T) {
	saveState()
	defer restoreState()
	useColors = false
	s := NewSpinner("nocolor")
	s.Start()
	time.Sleep(20 * time.Millisecond)
	s.Done("")
}
