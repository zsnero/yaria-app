//go:build windows

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	procFindWindowW      = user32.NewProc("FindWindowW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
	procMoveWindow       = user32.NewProc("MoveWindow")
	procClientToScreen   = user32.NewProc("ClientToScreen")
	procSetWindowPos     = user32.NewProc("SetWindowPos")
)

const (
	// WebView2 covers WS_CHILD windows — use an owned popup layered over the hole.
	wsPopup        = 0x80000000
	wsVisible      = 0x10000000
	wsClipSiblings = 0x04000000
	wsClipChildren = 0x02000000
	wsExNoActivate = 0x08000000
	wsExToolwindow = 0x00000080
	wsExTopmost    = 0x00000008
	swHide         = 0
	swShowNoActivate = 4
	swpNoZOrder    = 0x0004
	swpNoActivate  = 0x0010
	swpShowWindow  = 0x0040
	swpHideWindow  = 0x0080
	hwndTop        = 0
	hwndTopmost    = ^uintptr(0) // -1
)

type point struct {
	X, Y int32
}

type winMpv struct {
	ctx        context.Context
	parent     windows.HWND
	child      windows.HWND
	cmd        *exec.Cmd
	conn       net.Conn
	pipeName   string
	mu         sync.Mutex
	writeMu    sync.Mutex
	reqID      atomic.Int64
	pending    map[int64]chan ipcReply
	pendingMu  sync.Mutex
	timePos    float64
	duration   float64
	paused     bool
	eventRun   bool
	lastB      struct{ x, y, w, h int }
}

type ipcReply struct {
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

var winMPV *winMpv

func mpvPlatformAvailable() (bool, string) {
	if p := findMpvExe(); p != "" {
		return true, "using " + p
	}
	return false, "mpv.exe not found — enable Native player to auto-download, or install mpv"
}

func findMpvExe() string {
	deps := NewDepsService()
	if deps != nil {
		// Prefer portable exe in dependencies
		for _, name := range []string{"mpv.exe", "mpv.com"} {
			p := filepath.Join(deps.depsDir, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
		// Walk one level (some zips nest a folder)
		entries, _ := os.ReadDir(deps.depsDir)
		for _, e := range entries {
			if e.IsDir() {
				p := filepath.Join(deps.depsDir, e.Name(), "mpv.exe")
				if st, err := os.Stat(p); err == nil && !st.IsDir() {
					return p
				}
			}
		}
	}
	if p, err := exec.LookPath("mpv.exe"); err == nil {
		return p
	}
	if p, err := exec.LookPath("mpv"); err == nil {
		return p
	}
	return ""
}

func mpvPlatformStart(ctx context.Context) error {
	if winMPV != nil && winMPV.cmd != nil && winMPV.cmd.Process != nil {
		winMPV.ctx = ctx
		return nil
	}
	mpvPlatformStop()

	exe := findMpvExe()
	if exe == "" {
		return fmt.Errorf("mpv.exe not found")
	}

	parent := findWindowByTitle("Yaria")
	if parent == 0 {
		return fmt.Errorf("could not find Yaria window")
	}

	// Owned popup (not WS_CHILD) — WebView2 paints over child HWNDs
	child := createOverlayHWND(parent, 0, 0, 16, 16)
	if child == 0 {
		return fmt.Errorf("failed to create overlay window")
	}

	pipe := fmt.Sprintf(`\\.\pipe\yaria-mpv-%d`, os.Getpid())
	// mpv wants the pipe name without \\.\pipe\ prefix on some versions; try full path first
	ipcArg := pipe

	// Note: do not use --force-window with --wid (can open a second empty window).
	cmd := exec.Command(exe,
		fmt.Sprintf("--wid=%d", uint64(child)),
		"--idle=yes",
		"--keep-open=yes",
		"--no-terminal",
		"--osc=no",
		"--input-default-bindings=no",
		"--input-vo-keyboard=no",
		"--cursor-autohide=always",
		"--hwdec=auto-safe",
		"--vo=gpu",
		fmt.Sprintf("--input-ipc-server=%s", ipcArg),
	)
	hideConsole(cmd)
	// Run from deps dir so DLLs resolve
	if deps := NewDepsService(); deps != nil {
		cmd.Dir = deps.depsDir
	}
	if err := cmd.Start(); err != nil {
		destroyHWND(child)
		return fmt.Errorf("start mpv: %w", err)
	}

	conn, err := dialWindowsPipe(pipe, 8*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		destroyHWND(child)
		return fmt.Errorf("mpv IPC: %w", err)
	}

	w := &winMpv{
		ctx:      ctx,
		parent:   parent,
		child:    child,
		cmd:      cmd,
		conn:     conn,
		pipeName: pipe,
		pending:  make(map[int64]chan ipcReply),
		paused:   true,
		eventRun: true,
	}
	winMPV = w
	go w.readLoop()
	go w.pollLoop()

	// Observe useful properties
	_, _ = w.command("observe_property", 1, "time-pos")
	_, _ = w.command("observe_property", 2, "duration")
	_, _ = w.command("observe_property", 3, "pause")

	return nil
}

func mpvPlatformLoad(pathOrURL string) error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return fmt.Errorf("empty path")
	}
	_, err := w.command("loadfile", pathOrURL, "replace")
	if err != nil {
		return fmt.Errorf("loadfile %q: %w", pathOrURL, err)
	}
	_, _ = w.command("set_property", "pause", false)
	w.mu.Lock()
	w.paused = false
	w.mu.Unlock()
	return nil
}

func mpvPlatformSetBounds(x, y, w, h float64) error {
	st := winMPV
	if st == nil || st.child == 0 || st.parent == 0 {
		return fmt.Errorf("mpv surface not ready")
	}
	ix, iy, iw, ih := int(x+0.5), int(y+0.5), int(w+0.5), int(h+0.5)
	if iw < 2 {
		iw = 2
	}
	if ih < 2 {
		ih = 2
	}
	if st.lastB.x == ix && st.lastB.y == iy && st.lastB.w == iw && st.lastB.h == ih {
		return nil
	}
	st.lastB.x, st.lastB.y, st.lastB.w, st.lastB.h = ix, iy, iw, ih
	// Convert client (webview) coords → screen for owned popup
	sx, sy := clientToScreen(st.parent, ix, iy)
	moveHWND(st.child, sx, sy, iw, ih)
	return nil
}

func mpvPlatformSetVisible(visible bool) {
	st := winMPV
	if st == nil || st.child == 0 {
		return
	}
	const swpNoMove = 0x0002
	const swpNoSize = 0x0001
	if visible {
		showHWND(st.child, true)
		// Stay above WebView2 without activating
		procSetWindowPos.Call(
			uintptr(st.child),
			hwndTopmost,
			0, 0, 0, 0,
			uintptr(swpNoMove|swpNoSize|swpNoActivate|swpShowWindow),
		)
	} else {
		showHWND(st.child, false)
	}
}

func mpvPlatformSetPause(pause bool) error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	_, err := w.command("set_property", "pause", pause)
	if err == nil {
		w.mu.Lock()
		w.paused = pause
		w.mu.Unlock()
	}
	return err
}

func mpvPlatformTogglePause() error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	_, err := w.command("cycle", "pause")
	return err
}

func mpvPlatformSeek(seconds float64) error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	_, err := w.command("seek", seconds, "absolute")
	return err
}

func mpvPlatformSetVolume(vol float64) error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	_, err := w.command("set_property", "volume", vol)
	return err
}

func mpvPlatformSetSubtitle(path string) error {
	w := winMPV
	if w == nil {
		return fmt.Errorf("mpv not started")
	}
	_, err := w.command("sub_add", path, "select")
	return err
}

func mpvPlatformGetTime() float64 {
	if winMPV == nil {
		return 0
	}
	winMPV.mu.Lock()
	defer winMPV.mu.Unlock()
	return winMPV.timePos
}

func mpvPlatformGetDuration() float64 {
	if winMPV == nil {
		return 0
	}
	winMPV.mu.Lock()
	defer winMPV.mu.Unlock()
	return winMPV.duration
}

func mpvPlatformIsPaused() bool {
	if winMPV == nil {
		return true
	}
	winMPV.mu.Lock()
	defer winMPV.mu.Unlock()
	return winMPV.paused
}

func mpvPlatformStop() {
	w := winMPV
	winMPV = nil
	if w == nil {
		return
	}
	w.eventRun = false
	_, _ = w.command("quit")
	if w.conn != nil {
		_ = w.conn.Close()
	}
	if w.cmd != nil && w.cmd.Process != nil {
		done := make(chan struct{})
		go func() {
			_, _ = w.cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = w.cmd.Process.Kill()
			_, _ = w.cmd.Process.Wait()
		}
	}
	if w.child != 0 {
		destroyHWND(w.child)
	}
}

// --- IPC ---

func (w *winMpv) command(args ...interface{}) (json.RawMessage, error) {
	if w == nil || w.conn == nil {
		return nil, fmt.Errorf("no ipc")
	}
	id := w.reqID.Add(1)
	ch := make(chan ipcReply, 1)
	w.pendingMu.Lock()
	w.pending[id] = ch
	w.pendingMu.Unlock()

	msg := map[string]interface{}{
		"command":    args,
		"request_id": id,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')

	w.writeMu.Lock()
	_, err = w.conn.Write(b)
	w.writeMu.Unlock()
	if err != nil {
		w.pendingMu.Lock()
		delete(w.pending, id)
		w.pendingMu.Unlock()
		return nil, err
	}

	select {
	case rep := <-ch:
		// mpv sets "error":"success" on OK
		if rep.Error != "" && rep.Error != "success" {
			return rep.Data, fmt.Errorf("%s", rep.Error)
		}
		return rep.Data, nil
	case <-time.After(5 * time.Second):
		w.pendingMu.Lock()
		delete(w.pending, id)
		w.pendingMu.Unlock()
		return nil, fmt.Errorf("ipc timeout")
	}
}

func (w *winMpv) readLoop() {
	sc := bufio.NewScanner(w.conn)
	// large lines for events
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for w.eventRun && sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}
		// Response to command
		if ridRaw, ok := raw["request_id"]; ok {
			var rid int64
			_ = json.Unmarshal(ridRaw, &rid)
			var errStr string
			if e, ok := raw["error"]; ok {
				_ = json.Unmarshal(e, &errStr)
			}
			var data json.RawMessage
			if d, ok := raw["data"]; ok {
				data = d
			}
			w.pendingMu.Lock()
			ch := w.pending[rid]
			delete(w.pending, rid)
			w.pendingMu.Unlock()
			if ch != nil {
				ch <- ipcReply{Data: data, Error: errStr}
			}
			continue
		}
		// Property change / events
		var event string
		if e, ok := raw["event"]; ok {
			_ = json.Unmarshal(e, &event)
		}
		switch event {
		case "property-change":
			var name string
			if n, ok := raw["name"]; ok {
				_ = json.Unmarshal(n, &name)
			}
			data := raw["data"]
			w.handleProp(name, data)
		case "end-file":
			if w.ctx != nil {
				wailsRuntime.EventsEmit(w.ctx, "mpv-eof", map[string]interface{}{})
			}
		}
	}
}

func (w *winMpv) handleProp(name string, data json.RawMessage) {
	w.mu.Lock()
	defer w.mu.Unlock()
	switch name {
	case "time-pos":
		var v float64
		if json.Unmarshal(data, &v) == nil {
			w.timePos = v
		}
	case "duration":
		var v float64
		if json.Unmarshal(data, &v) == nil && v > 0 {
			w.duration = v
		}
	case "pause":
		var v bool
		if json.Unmarshal(data, &v) == nil {
			w.paused = v
		}
	}
}

func (w *winMpv) pollLoop() {
	t := time.NewTicker(250 * time.Millisecond)
	defer t.Stop()
	for w.eventRun {
		<-t.C
		if w.ctx == nil {
			continue
		}
		w.mu.Lock()
		tp, dur, paused := w.timePos, w.duration, w.paused
		w.mu.Unlock()
		wailsRuntime.EventsEmit(w.ctx, "mpv-time", map[string]interface{}{
			"time": tp, "duration": dur, "paused": paused,
		})
	}
}

// --- Win32 helpers ---

func findWindowByTitle(title string) windows.HWND {
	t, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	r, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	return windows.HWND(r)
}

func createOverlayHWND(owner windows.HWND, x, y, w, h int) windows.HWND {
	className, _ := windows.UTF16PtrFromString("STATIC")
	windowName, _ := windows.UTF16PtrFromString("")
	// Owned popup sits above WebView2; owner keeps it tied to Yaria
	exStyle := uintptr(wsExNoActivate | wsExToolwindow | wsExTopmost)
	style := uintptr(wsPopup | wsVisible | wsClipSiblings | wsClipChildren)
	sx, sy := clientToScreen(owner, x, y)
	r, _, _ := procCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		style,
		uintptr(sx), uintptr(sy), uintptr(w), uintptr(h),
		uintptr(owner),
		0, 0, 0,
	)
	return windows.HWND(r)
}

func clientToScreen(hwnd windows.HWND, x, y int) (int, int) {
	pt := point{X: int32(x), Y: int32(y)}
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pt)))
	return int(pt.X), int(pt.Y)
}

func destroyHWND(h windows.HWND) {
	if h != 0 {
		procDestroyWindow.Call(uintptr(h))
	}
}

func moveHWND(h windows.HWND, x, y, w, hgt int) {
	procMoveWindow.Call(uintptr(h), uintptr(x), uintptr(y), uintptr(w), uintptr(hgt), 1)
}

func showHWND(h windows.HWND, show bool) {
	cmd := uintptr(swHide)
	if show {
		cmd = swShowNoActivate
	}
	procShowWindow.Call(uintptr(h), cmd)
}

func dialWindowsPipe(pipe string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		// net.Dial works for named pipes on recent Go with winio-less path via "npipe" — use CreateFile
		conn, err := dialPipeCreateFile(pipe)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timeout")
	}
	return nil, lastErr
}

func dialPipeCreateFile(pipe string) (net.Conn, error) {
	path, err := windows.UTF16PtrFromString(pipe)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(
		path,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}
	// Dup handle into two os.Files for bufio, or use net.FileConn
	f := os.NewFile(uintptr(h), pipe)
	// Use a simple wrapper: one file for read/write
	return &fileConn{f: f}, nil
}

// fileConn adapts *os.File to net.Conn for IPC.
type fileConn struct {
	f *os.File
}

func (c *fileConn) Read(b []byte) (int, error)         { return c.f.Read(b) }
func (c *fileConn) Write(b []byte) (int, error)        { return c.f.Write(b) }
func (c *fileConn) Close() error                      { return c.f.Close() }
func (c *fileConn) LocalAddr() net.Addr               { return pipeAddr("local") }
func (c *fileConn) RemoteAddr() net.Addr              { return pipeAddr("remote") }
func (c *fileConn) SetDeadline(t time.Time) error     { return c.f.SetDeadline(t) }
func (c *fileConn) SetReadDeadline(t time.Time) error { return c.f.SetReadDeadline(t) }
func (c *fileConn) SetWriteDeadline(t time.Time) error {
	return c.f.SetWriteDeadline(t)
}

type pipeAddr string

func (p pipeAddr) Network() string { return "pipe" }
func (p pipeAddr) String() string  { return string(p) }
