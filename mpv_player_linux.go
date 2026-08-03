//go:build linux

package main

/*
#cgo pkg-config: mpv x11
#include <mpv/client.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

static inline int mpv_set_wid(mpv_handle *h, int64_t wid) {
	return mpv_set_property(h, "wid", MPV_FORMAT_INT64, &wid);
}

static Window find_named_window(Display *dpy, Window root, const char *needle) {
	Atom netList = XInternAtom(dpy, "_NET_CLIENT_LIST", False);
	Atom netName = XInternAtom(dpy, "_NET_WM_NAME", False);
	Atom utf8 = XInternAtom(dpy, "UTF8_STRING", False);
	Atom type;
	int format;
	unsigned long nitems = 0, bytes = 0;
	unsigned char *data = NULL;

	if (XGetWindowProperty(dpy, root, netList, 0, 1024, False, XA_WINDOW,
			&type, &format, &nitems, &bytes, &data) == Success && data) {
		Window *wins = (Window *)data;
		for (unsigned long i = 0; i < nitems; i++) {
			Atom t2;
			int f2;
			unsigned long n2 = 0, b2 = 0;
			unsigned char *name = NULL;
			if (XGetWindowProperty(dpy, wins[i], netName, 0, 1024, False, utf8,
					&t2, &f2, &n2, &b2, &name) == Success && name) {
				if (strstr((char *)name, needle) != NULL) {
					Window found = wins[i];
					XFree(name);
					XFree(data);
					return found;
				}
				XFree(name);
			}
			// Fallback WM_NAME
			char *wm = NULL;
			if (XFetchName(dpy, wins[i], &wm) && wm) {
				if (strstr(wm, needle) != NULL) {
					Window found = wins[i];
					XFree(wm);
					XFree(data);
					return found;
				}
				XFree(wm);
			}
		}
		XFree(data);
	}
	return 0;
}

static Window create_child(Display *dpy, Window parent, int x, int y, int w, int h) {
	if (w < 2) w = 2;
	if (h < 2) h = 2;
	XSetWindowAttributes swa;
	memset(&swa, 0, sizeof(swa));
	swa.background_pixel = BlackPixel(dpy, DefaultScreen(dpy));
	swa.event_mask = ExposureMask | StructureNotifyMask;
	swa.override_redirect = False;
	Window child = XCreateWindow(
		dpy, parent,
		x, y, (unsigned)w, (unsigned)h,
		0,
		CopyFromParent, InputOutput, CopyFromParent,
		CWBackPixel | CWEventMask,
		&swa
	);
	XMapWindow(dpy, child);
	XFlush(dpy);
	return child;
}

static void move_resize(Display *dpy, Window win, int x, int y, int w, int h) {
	if (w < 2) w = 2;
	if (h < 2) h = 2;
	XMoveResizeWindow(dpy, win, x, y, (unsigned)w, (unsigned)h);
	XFlush(dpy);
}

static void set_mapped(Display *dpy, Window win, int show) {
	if (show) XMapWindow(dpy, win);
	else XUnmapWindow(dpy, win);
	XFlush(dpy);
}
*/
import "C"

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unsafe"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	mpvOnce     sync.Once
	mpvInitErr  error
	mpvHandle   *C.mpv_handle
	mpvDisplay  *C.Display
	mpvParent   C.Window
	mpvChild    C.Window
	mpvCtx      context.Context
	mpvEventRun bool
	mpvLastB    struct{ x, y, w, h int }
)

func mpvPlatformAvailable() (bool, string) {
	// Need libmpv (linked) and a display; also accept system mpv binary as health signal
	if _, err := exec.LookPath("mpv"); err != nil {
		// libmpv may still work without the CLI
	}
	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return false, "No X11 display (libmpv embed needs X11)"
	}
	C.XCloseDisplay(dpy)
	return true, ""
}

func mpvPlatformStart(ctx context.Context) error {
	mpvCtx = ctx
	if mpvHandle != nil {
		return nil
	}

	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return fmt.Errorf("cannot open X11 display")
	}
	mpvDisplay = dpy
	root := C.XDefaultRootWindow(dpy)

	// Find Yaria main window by title
	name := C.CString("Yaria")
	defer C.free(unsafe.Pointer(name))
	parent := C.find_named_window(dpy, root, name)
	if parent == 0 {
		C.XCloseDisplay(dpy)
		mpvDisplay = nil
		return fmt.Errorf("could not find Yaria window (is the title still \"Yaria\"?)")
	}
	mpvParent = parent

	// Placeholder child; real size comes from SetBounds
	mpvChild = C.create_child(dpy, parent, 0, 0, 16, 16)
	if mpvChild == 0 {
		C.XCloseDisplay(dpy)
		mpvDisplay = nil
		return fmt.Errorf("failed to create child window")
	}

	h := C.mpv_create()
	if h == nil {
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("mpv_create failed")
	}

	// Embed into child
	if C.mpv_set_wid(h, C.int64_t(mpvChild)) != 0 {
		C.mpv_destroy(h)
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("failed to set mpv wid")
	}

	// Reasonable defaults
	setOpt := func(k, v string) {
		ck, cv := C.CString(k), C.CString(v)
		C.mpv_set_option_string(h, ck, cv)
		C.free(unsafe.Pointer(ck))
		C.free(unsafe.Pointer(cv))
	}
	setOpt("vo", "gpu")
	setOpt("hwdec", "auto-safe")
	setOpt("keep-open", "yes")
	setOpt("idle", "yes")
	setOpt("osc", "no")
	setOpt("input-default-bindings", "no")
	setOpt("input-vo-keyboard", "no")
	setOpt("cursor-autohide", "always")

	if C.mpv_initialize(h) < 0 {
		C.mpv_destroy(h)
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("mpv_initialize failed")
	}

	mpvHandle = h
	mpvEventRun = true
	go mpvEventLoop()
	return nil
}

func mpvEventLoop() {
	for mpvEventRun && mpvHandle != nil {
		ev := C.mpv_wait_event(mpvHandle, 0.25)
		if ev == nil {
			continue
		}
		switch ev.event_id {
		case C.MPV_EVENT_SHUTDOWN:
			return
		case C.MPV_EVENT_END_FILE:
			if mpvCtx != nil {
				wailsRuntime.EventsEmit(mpvCtx, "mpv-eof", map[string]interface{}{})
			}
		case C.MPV_EVENT_PROPERTY_CHANGE, C.MPV_EVENT_PLAYBACK_RESTART:
			// periodic time updates via poll below
		}
		// Also emit time periodically
		if mpvCtx != nil && mpvHandle != nil {
			t := mpvPlatformGetTime()
			d := mpvPlatformGetDuration()
			wailsRuntime.EventsEmit(mpvCtx, "mpv-time", map[string]interface{}{
				"time": t, "duration": d, "paused": mpvPlatformIsPaused(),
			})
		}
	}
}

func mpvPlatformLoad(pathOrURL string) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	pathOrURL = strings.TrimSpace(pathOrURL)
	if pathOrURL == "" {
		return fmt.Errorf("empty path")
	}
	cpath := C.CString(pathOrURL)
	defer C.free(unsafe.Pointer(cpath))
	// mpv_command: loadfile <url> replace
	cmd := "**\x00"
	_ = cmd
	args := []*C.char{
		C.CString("loadfile"),
		cpath,
		C.CString("replace"),
		nil,
	}
	defer C.free(unsafe.Pointer(args[0]))
	defer C.free(unsafe.Pointer(args[2]))
	// loadfile keeps path alive until command returns
	ret := C.mpv_command(mpvHandle, &args[0])
	if ret < 0 {
		return fmt.Errorf("loadfile failed: %s", C.GoString(C.mpv_error_string(ret)))
	}
	// Observe nothing extra; event loop polls time
	_ = time.Now()
	return nil
}

func mpvPlatformSetBounds(x, y, w, h float64) error {
	if mpvDisplay == nil || mpvChild == 0 {
		return fmt.Errorf("mpv surface not ready")
	}
	ix, iy, iw, ih := int(x+0.5), int(y+0.5), int(w+0.5), int(h+0.5)
	if iw < 2 {
		iw = 2
	}
	if ih < 2 {
		ih = 2
	}
	if mpvLastB.x == ix && mpvLastB.y == iy && mpvLastB.w == iw && mpvLastB.h == ih {
		return nil
	}
	mpvLastB.x, mpvLastB.y, mpvLastB.w, mpvLastB.h = ix, iy, iw, ih
	C.move_resize(mpvDisplay, mpvChild, C.int(ix), C.int(iy), C.int(iw), C.int(ih))
	return nil
}

func mpvPlatformSetVisible(visible bool) {
	if mpvDisplay == nil || mpvChild == 0 {
		return
	}
	v := C.int(0)
	if visible {
		v = 1
	}
	C.set_mapped(mpvDisplay, mpvChild, v)
}

func mpvPlatformSetPause(pause bool) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	var flag C.int
	if pause {
		flag = 1
	}
	ck := C.CString("pause")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_FLAG, unsafe.Pointer(&flag)))
}

func mpvPlatformTogglePause() error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	ck := C.CString("cycle")
	cv := C.CString("pause")
	defer C.free(unsafe.Pointer(ck))
	defer C.free(unsafe.Pointer(cv))
	args := []*C.char{ck, cv, nil}
	return mpvRet(C.mpv_command(mpvHandle, &args[0]))
}

func mpvPlatformSeek(seconds float64) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	// seek absolute
	ck := C.CString("seek")
	cv := C.CString(fmt.Sprintf("%f", seconds))
	abs := C.CString("absolute")
	defer C.free(unsafe.Pointer(ck))
	defer C.free(unsafe.Pointer(cv))
	defer C.free(unsafe.Pointer(abs))
	args := []*C.char{ck, cv, abs, nil}
	return mpvRet(C.mpv_command(mpvHandle, &args[0]))
}

func mpvPlatformSetVolume(vol float64) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	v := C.double(vol)
	ck := C.CString("volume")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)))
}

func mpvPlatformGetTime() float64 {
	if mpvHandle == nil {
		return 0
	}
	var v C.double
	ck := C.CString("time-pos")
	defer C.free(unsafe.Pointer(ck))
	if C.mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)) < 0 {
		return 0
	}
	return float64(v)
}

func mpvPlatformGetDuration() float64 {
	if mpvHandle == nil {
		return 0
	}
	var v C.double
	ck := C.CString("duration")
	defer C.free(unsafe.Pointer(ck))
	if C.mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)) < 0 {
		return 0
	}
	return float64(v)
}

func mpvPlatformIsPaused() bool {
	if mpvHandle == nil {
		return true
	}
	var flag C.int
	ck := C.CString("pause")
	defer C.free(unsafe.Pointer(ck))
	if C.mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_FLAG, unsafe.Pointer(&flag)) < 0 {
		return true
	}
	return flag != 0
}

func mpvPlatformStop() {
	mpvEventRun = false
	if mpvHandle != nil {
		C.mpv_terminate_destroy(mpvHandle)
		mpvHandle = nil
	}
	if mpvDisplay != nil {
		if mpvChild != 0 {
			C.XDestroyWindow(mpvDisplay, mpvChild)
			mpvChild = 0
		}
		C.XCloseDisplay(mpvDisplay)
		mpvDisplay = nil
	}
	mpvParent = 0
	mpvLastB = struct{ x, y, w, h int }{}
}

func mpvRet(code C.int) error {
	if code >= 0 {
		return nil
	}
	return fmt.Errorf("%s", C.GoString(C.mpv_error_string(code)))
}
