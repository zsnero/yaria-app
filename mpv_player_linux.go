//go:build linux

package main

/*
#cgo pkg-config: x11
#cgo LDFLAGS: -ldl -lX11
#include <mpv/client.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/Xutil.h>
#include <dlfcn.h>
#include <locale.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// --- X11 helpers (unchanged) ---

// Forward decl — defined after function pointers below
static int (*p_mpv_set_property)(mpv_handle*, const char*, mpv_format, void*);

static inline int yaria_mpv_set_wid(mpv_handle *h, int64_t wid) {
	return p_mpv_set_property(h, "wid", MPV_FORMAT_INT64, &wid);
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

// --- Runtime libmpv via dlopen (no DT_NEEDED — app starts without system mpv) ---

static void *yaria_libmpv;

static mpv_handle *(*p_mpv_create)(void);
static int (*p_mpv_initialize)(mpv_handle*);
static void (*p_mpv_destroy)(mpv_handle*);
static void (*p_mpv_terminate_destroy)(mpv_handle*);
static int (*p_mpv_set_option_string)(mpv_handle*, const char*, const char*);
// p_mpv_set_property declared above (used by yaria_mpv_set_wid)
static int (*p_mpv_get_property)(mpv_handle*, const char*, mpv_format, void*);
static int (*p_mpv_command)(mpv_handle*, const char**);
static const char *(*p_mpv_error_string)(int);
static mpv_event *(*p_mpv_wait_event)(mpv_handle*, double);
static void (*p_mpv_free_node_contents)(mpv_node*);

static int yaria_bind_sym(void **dst, const char *name) {
	*dst = dlsym(yaria_libmpv, name);
	return *dst ? 0 : -1;
}

// Load libmpv from path (or default soname if path is NULL/empty).
// Returns 0 on success; on failure sets *err_out via dlerror (caller frees with free()).
int yaria_load_libmpv(const char *path, char **err_out) {
	if (yaria_libmpv) return 0;
	const char *try_path = (path && path[0]) ? path : "libmpv.so.2";
	yaria_libmpv = dlopen(try_path, RTLD_NOW | RTLD_GLOBAL);
	if (!yaria_libmpv && path && path[0]) {
		yaria_libmpv = dlopen("libmpv.so.2", RTLD_NOW | RTLD_GLOBAL);
	}
	if (!yaria_libmpv) {
		yaria_libmpv = dlopen("libmpv.so", RTLD_NOW | RTLD_GLOBAL);
	}
	if (!yaria_libmpv) {
		if (err_out) {
			const char *e = dlerror();
			*err_out = e ? strdup(e) : strdup("dlopen libmpv failed");
		}
		return -1;
	}
	if (yaria_bind_sym((void**)&p_mpv_create, "mpv_create") ||
		yaria_bind_sym((void**)&p_mpv_initialize, "mpv_initialize") ||
		yaria_bind_sym((void**)&p_mpv_destroy, "mpv_destroy") ||
		yaria_bind_sym((void**)&p_mpv_terminate_destroy, "mpv_terminate_destroy") ||
		yaria_bind_sym((void**)&p_mpv_set_option_string, "mpv_set_option_string") ||
		yaria_bind_sym((void**)&p_mpv_set_property, "mpv_set_property") ||
		yaria_bind_sym((void**)&p_mpv_get_property, "mpv_get_property") ||
		yaria_bind_sym((void**)&p_mpv_command, "mpv_command") ||
		yaria_bind_sym((void**)&p_mpv_error_string, "mpv_error_string") ||
		yaria_bind_sym((void**)&p_mpv_wait_event, "mpv_wait_event") ||
		yaria_bind_sym((void**)&p_mpv_free_node_contents, "mpv_free_node_contents")) {
		if (err_out) *err_out = strdup("libmpv missing required symbols");
		dlclose(yaria_libmpv);
		yaria_libmpv = NULL;
		return -1;
	}
	return 0;
}

int yaria_libmpv_loaded(void) { return yaria_libmpv != NULL; }

// Thin wrappers so Go can call through function pointers
mpv_handle *y_mpv_create(void) { return p_mpv_create(); }
int y_mpv_initialize(mpv_handle *h) { return p_mpv_initialize(h); }
void y_mpv_destroy(mpv_handle *h) { p_mpv_destroy(h); }
void y_mpv_terminate_destroy(mpv_handle *h) { p_mpv_terminate_destroy(h); }
int y_mpv_set_option_string(mpv_handle *h, const char *k, const char *v) { return p_mpv_set_option_string(h, k, v); }
int y_mpv_set_property(mpv_handle *h, const char *name, mpv_format fmt, void *data) { return p_mpv_set_property(h, name, fmt, data); }
int y_mpv_set_property_string(mpv_handle *h, const char *name, const char *value) { return p_mpv_set_property(h, name, MPV_FORMAT_STRING, (void*)&value); }
int y_mpv_get_property(mpv_handle *h, const char *name, mpv_format fmt, void *data) { return p_mpv_get_property(h, name, fmt, data); }
int y_mpv_command(mpv_handle *h, const char **args) { return p_mpv_command(h, args); }
const char *y_mpv_error_string(int code) { return p_mpv_error_string(code); }
mpv_event *y_mpv_wait_event(mpv_handle *h, double timeout) { return p_mpv_wait_event(h, timeout); }

// --- track-list → JSON (audio/video/subtitle track enumeration) ---

static void yaria_jappend(char **out, size_t *cap, size_t *len, const char *s) {
	size_t sl = strlen(s);
	if (*out == NULL) {
		*cap = 256;
		*out = malloc(*cap);
		*len = 0;
		(*out)[0] = 0;
	}
	if (*len + sl + 1 > *cap) {
		while (*len + sl + 1 > *cap) *cap *= 2;
		*out = realloc(*out, *cap);
	}
	memcpy(*out + *len, s, sl);
	*len += sl;
	(*out)[*len] = 0;
}

static int yaria_is_track_key(const char *k) {
	return strcmp(k, "id") == 0 || strcmp(k, "type") == 0 || strcmp(k, "lang") == 0 ||
		strcmp(k, "title") == 0 || strcmp(k, "selected") == 0 || strcmp(k, "default") == 0 ||
		strcmp(k, "external") == 0 || strcmp(k, "codec") == 0;
}

static void yaria_track_value(char **out, size_t *cap, size_t *len, const mpv_node *v) {
	switch (v->format) {
	case MPV_FORMAT_STRING:
		yaria_jappend(out, cap, len, "\"");
		if (v->u.string) {
			const char *p = v->u.string;
			while (*p) {
				if (*p == '"' || *p == '\\') {
					char esc[3] = { '\\', *p, 0 };
					yaria_jappend(out, cap, len, esc);
				} else {
					char c[2] = { *p, 0 };
					yaria_jappend(out, cap, len, c);
				}
				p++;
			}
		}
		yaria_jappend(out, cap, len, "\"");
		break;
	case MPV_FORMAT_FLAG:
		yaria_jappend(out, cap, len, v->u.flag ? "true" : "false");
		break;
	case MPV_FORMAT_INT64:
		{
			char tmp[32];
			snprintf(tmp, sizeof tmp, "%lld", (long long)v->u.int64);
			yaria_jappend(out, cap, len, tmp);
		}
		break;
	case MPV_FORMAT_DOUBLE:
		{
			char tmp[40];
			snprintf(tmp, sizeof tmp, "%g", v->u.double_);
			yaria_jappend(out, cap, len, tmp);
		}
		break;
	default:
		yaria_jappend(out, cap, len, "null");
	}
}

// Returns a malloc'd JSON array of the current file's tracks (caller frees).
static char *yaria_mpv_get_tracks_json(mpv_handle *h) {
	mpv_node root;
	if (p_mpv_get_property(h, "track-list", MPV_FORMAT_NODE, &root) < 0) {
		return NULL;
	}
	char *out = NULL;
	size_t cap = 0, len = 0;
	yaria_jappend(&out, &cap, &len, "[");
	if (root.format == MPV_FORMAT_NODE_ARRAY && root.u.list) {
		for (int i = 0; i < root.u.list->num; i++) {
			if (i) yaria_jappend(&out, &cap, &len, ",");
			mpv_node *item = &root.u.list->values[i];
			yaria_jappend(&out, &cap, &len, "{");
			if (item && item->format == MPV_FORMAT_NODE_MAP && item->u.list) {
				int first = 1;
				for (int k = 0; k < item->u.list->num; k++) {
					const char *key = item->u.list->keys ? item->u.list->keys[k] : NULL;
					if (!key || !yaria_is_track_key(key)) continue;
					if (!first) yaria_jappend(&out, &cap, &len, ",");
					first = 0;
					yaria_jappend(&out, &cap, &len, "\"");
					yaria_jappend(&out, &cap, &len, key);
					yaria_jappend(&out, &cap, &len, "\":");
					yaria_track_value(&out, &cap, &len, &item->u.list->values[k]);
				}
			}
			yaria_jappend(&out, &cap, &len, "}");
		}
	}
	yaria_jappend(&out, &cap, &len, "]");
	p_mpv_free_node_contents(&root);
	return out;
}
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	mpvHandle   *C.mpv_handle
	mpvDisplay  *C.Display
	mpvParent   C.Window
	mpvChild    C.Window
	mpvCtx      context.Context
	mpvEventRun bool
	mpvLastB    struct{ x, y, w, h int }
	mpvLoadMu   sync.Mutex
	mpvLoadErr  error
	mpvLoaded   bool
)

// ensureLibmpvLoaded dlopens system or bundled libmpv. Never links at process start.
func ensureLibmpvLoaded() error {
	mpvLoadMu.Lock()
	defer mpvLoadMu.Unlock()
	if mpvLoaded {
		return mpvLoadErr
	}
	mpvLoaded = true

	deps := NewDepsService()
	// Prefer bundled libs for dlopen resolution of NEEDED entries
	if deps != nil && deps.depsDir != "" {
		prev := os.Getenv("LD_LIBRARY_PATH")
		joined := deps.depsDir
		if prev != "" {
			joined = deps.depsDir + string(os.PathListSeparator) + prev
		}
		_ = os.Setenv("LD_LIBRARY_PATH", joined)
	}

	candidates := []string{}
	if deps != nil {
		if p, ok := deps.MpvLibPath(); ok {
			candidates = append(candidates, p)
		}
		for _, name := range []string{"libmpv.so.2", "libmpv.so", "libmpv.so.1"} {
			candidates = append(candidates, filepath.Join(deps.depsDir, name))
		}
	}
	candidates = append(candidates,
		"libmpv.so.2",
		"/usr/lib/libmpv.so.2",
		"/usr/lib64/libmpv.so.2",
		"/usr/lib/x86_64-linux-gnu/libmpv.so.2",
		"/usr/lib/aarch64-linux-gnu/libmpv.so.2",
		"libmpv.so",
	)

	var lastErr string
	for _, c := range candidates {
		if c == "" {
			continue
		}
		// Skip missing files for absolute paths
		if strings.Contains(c, "/") {
			if st, err := os.Stat(c); err != nil || st.IsDir() {
				continue
			}
		}
		cs := C.CString(c)
		var cerr *C.char
		ret := C.yaria_load_libmpv(cs, &cerr)
		C.free(unsafe.Pointer(cs))
		if ret == 0 {
			mpvLoadErr = nil
			return nil
		}
		if cerr != nil {
			lastErr = C.GoString(cerr)
			C.free(unsafe.Pointer(cerr))
		}
	}

	// Last resort: trigger auto-download then retry once
	if deps != nil {
		deps.downloadMpv()
		if p, ok := deps.MpvLibPath(); ok {
			cs := C.CString(p)
			var cerr *C.char
			ret := C.yaria_load_libmpv(cs, &cerr)
			C.free(unsafe.Pointer(cs))
			if ret == 0 {
				mpvLoadErr = nil
				return nil
			}
			if cerr != nil {
				lastErr = C.GoString(cerr)
				C.free(unsafe.Pointer(cerr))
			}
		}
	}

	if lastErr == "" {
		lastErr = "libmpv not found"
	}
	mpvLoadErr = fmt.Errorf("%s (auto-setup failed — install mpv or retry online)", lastErr)
	return mpvLoadErr
}

func mpvPlatformAvailable() (bool, string) {
	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return false, "No X11 display (libmpv embed needs X11 / XWayland)"
	}
	C.XCloseDisplay(dpy)

	if C.yaria_libmpv_loaded() != 0 {
		return true, "libmpv loaded"
	}
	if deps := NewDepsService(); deps != nil {
		if p, ok := deps.MpvLibPath(); ok {
			return true, "using " + p
		}
		if p, ok := deps.MpvLibOrBinary(); ok {
			// binary alone isn't enough for embed, but download may have lib
			if strings.Contains(p, "libmpv") {
				return true, "using " + p
			}
		}
	}
	// Probe without caching failure (download may still be in progress)
	for _, p := range []string{
		"/usr/lib/libmpv.so.2", "/usr/lib64/libmpv.so.2",
		"/usr/lib/x86_64-linux-gnu/libmpv.so.2",
		"/usr/lib/aarch64-linux-gnu/libmpv.so.2",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return true, "system " + p
		}
	}
	return false, "libmpv not installed yet — Yaria will auto-download it on first setup"
}

func mpvPlatformStart(ctx context.Context) error {
	mpvCtx = ctx
	if mpvHandle != nil {
		return nil
	}
	if err := ensureLibmpvLoaded(); err != nil {
		return err
	}

	dpy := C.XOpenDisplay(nil)
	if dpy == nil {
		return fmt.Errorf("cannot open X11 display (DISPLAY=%q)", os.Getenv("DISPLAY"))
	}
	mpvDisplay = dpy
	root := C.XDefaultRootWindow(dpy)

	name := C.CString("Yaria")
	defer C.free(unsafe.Pointer(name))
	parent := C.find_named_window(dpy, root, name)
	if parent == 0 {
		C.XCloseDisplay(dpy)
		mpvDisplay = nil
		return fmt.Errorf("could not find Yaria window (is the title still \"Yaria\"?)")
	}
	mpvParent = parent

	mpvChild = C.create_child(dpy, parent, 0, 0, 16, 16)
	if mpvChild == 0 {
		C.XCloseDisplay(dpy)
		mpvDisplay = nil
		return fmt.Errorf("failed to create child window")
	}

	// libmpv aborts in mpv_create() unless LC_NUMERIC is "C" (locale-independent
	// number parsing). Set it before creating the handle — safe for Go/GTK which
	// do their own number formatting.
	loc := C.CString("C")
	C.setlocale(C.LC_NUMERIC, loc)
	C.free(unsafe.Pointer(loc))

	h := C.y_mpv_create()
	if h == nil {
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("mpv_create failed")
	}

	if C.yaria_mpv_set_wid(h, C.int64_t(mpvChild)) != 0 {
		C.y_mpv_destroy(h)
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("failed to set mpv wid")
	}

	setOpt := func(k, v string) {
		ck, cv := C.CString(k), C.CString(v)
		C.y_mpv_set_option_string(h, ck, cv)
		C.free(unsafe.Pointer(ck))
		C.free(unsafe.Pointer(cv))
	}
	setOpt("vo", "gpu")
	// mpv 0.35+ defaults gpu-api to Vulkan; with an X11 child window under
	// XWayland that silently falls back to a detached top-level window.
	// Force the EGL/OpenGL context so the video actually embeds in-window.
	setOpt("gpu-context", "x11egl")
	setOpt("hwdec", "auto-safe")
	setOpt("keep-open", "yes")
	setOpt("idle", "yes")
	setOpt("osc", "no")
	setOpt("input-default-bindings", "no")
	setOpt("input-vo-keyboard", "no")
	setOpt("cursor-autohide", "always")
	// Allow the volume property above 100% so the frontend's audio-boost slider
	// (100-500%) maps directly onto mpv's volume.
	setOpt("volume-max", "500")

	if C.y_mpv_initialize(h) < 0 {
		C.y_mpv_destroy(h)
		C.XDestroyWindow(dpy, mpvChild)
		C.XCloseDisplay(dpy)
		mpvChild = 0
		mpvDisplay = nil
		return fmt.Errorf("mpv_initialize failed")
	}

	mpvHandle = h
	mpvEventRun = true
	go mpvEventLoop()
	// Restore the user's last aspect-ratio mode (idle set is valid; it applies
	// once a file loads).
	_ = mpvPlatformSetAspectMode(mpvAspectMode)
	return nil
}

func mpvEventLoop() {
	for mpvEventRun && mpvHandle != nil {
		ev := C.y_mpv_wait_event(mpvHandle, 0.25)
		if ev != nil {
			switch ev.event_id {
			case C.MPV_EVENT_SHUTDOWN:
				return
			case C.MPV_EVENT_END_FILE:
				if mpvCtx != nil {
					wailsRuntime.EventsEmit(mpvCtx, "mpv-eof", map[string]interface{}{})
				}
			case C.MPV_EVENT_PROPERTY_CHANGE, C.MPV_EVENT_PLAYBACK_RESTART:
			}
		}
		// Steady-state playback produces almost no mpv events, so also push
		// time/duration on every wake. Without this the frontend's currentTime
		// stalls near 0 and position saves / Continue Watching never update.
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
	args := []*C.char{
		C.CString("loadfile"),
		cpath,
		C.CString("replace"),
		nil,
	}
	defer C.free(unsafe.Pointer(args[0]))
	defer C.free(unsafe.Pointer(args[2]))
	ret := C.y_mpv_command(mpvHandle, &args[0])
	if ret < 0 {
		return fmt.Errorf("loadfile failed: %s", C.GoString(C.y_mpv_error_string(ret)))
	}
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
	return mpvRet(C.y_mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_FLAG, unsafe.Pointer(&flag)))
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
	return mpvRet(C.y_mpv_command(mpvHandle, &args[0]))
}

func mpvPlatformSeek(seconds float64) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	ck := C.CString("seek")
	cv := C.CString(fmt.Sprintf("%f", seconds))
	abs := C.CString("absolute")
	defer C.free(unsafe.Pointer(ck))
	defer C.free(unsafe.Pointer(cv))
	defer C.free(unsafe.Pointer(abs))
	args := []*C.char{ck, cv, abs, nil}
	return mpvRet(C.y_mpv_command(mpvHandle, &args[0]))
}

func mpvPlatformSetVolume(vol float64) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 500 {
		vol = 500
	}
	v := C.double(vol)
	ck := C.CString("volume")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.y_mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)))
}

// mpvPlatformSetSubtitle adds an external subtitle file (sub-add ... select).
func mpvPlatformSetSubtitle(path string) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty subtitle path")
	}
	subCmd := C.CString("sub-add")
	cpath := C.CString(path)
	sel := C.CString("select")
	defer C.free(unsafe.Pointer(subCmd))
	defer C.free(unsafe.Pointer(cpath))
	defer C.free(unsafe.Pointer(sel))
	args := []*C.char{subCmd, cpath, sel, nil}
	return mpvRet(C.y_mpv_command(mpvHandle, &args[0]))
}

func mpvPlatformGetTime() float64 {
	if mpvHandle == nil {
		return 0
	}
	var v C.double
	ck := C.CString("time-pos")
	defer C.free(unsafe.Pointer(ck))
	if C.y_mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)) < 0 {
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
	if C.y_mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_DOUBLE, unsafe.Pointer(&v)) < 0 {
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
	if C.y_mpv_get_property(mpvHandle, ck, C.MPV_FORMAT_FLAG, unsafe.Pointer(&flag)) < 0 {
		return true
	}
	return flag != 0
}

func mpvPlatformStop() {
	mpvEventRun = false
	if mpvHandle != nil {
		C.y_mpv_terminate_destroy(mpvHandle)
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
	return fmt.Errorf("%s", C.GoString(C.y_mpv_error_string(code)))
}

// mpvPlatformGetTracks returns the current file's video/audio/subtitle tracks.
func mpvPlatformGetTracks() []MpvTrack {
	if mpvHandle == nil {
		return nil
	}
	cj := C.yaria_mpv_get_tracks_json(mpvHandle)
	if cj == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cj))
	var tracks []MpvTrack
	if err := json.Unmarshal([]byte(C.GoString(cj)), &tracks); err != nil {
		return nil
	}
	return tracks
}

// mpvPlatformSetAudioTrack selects an audio track by id (0 = mpv auto).
func mpvPlatformSetAudioTrack(id int) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	if id < 0 {
		id = 0
	}
	v := C.int64_t(id)
	ck := C.CString("aid")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.y_mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_INT64, unsafe.Pointer(&v)))
}

// mpvPlatformSetSubtitleTrack selects a subtitle track by id (0 = mpv auto, -1 = no).
func mpvPlatformSetSubtitleTrack(id int) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	v := C.int64_t(id)
	ck := C.CString("sid")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.y_mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_INT64, unsafe.Pointer(&v)))
}

// mpvPlatformSetSubtitleEnabled toggles sub-visibility without changing the track.
func mpvPlatformSetSubtitleEnabled(on bool) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	var flag C.int
	if on {
		flag = 1
	}
	ck := C.CString("sub-visibility")
	defer C.free(unsafe.Pointer(ck))
	return mpvRet(C.y_mpv_set_property(mpvHandle, ck, C.MPV_FORMAT_FLAG, unsafe.Pointer(&flag)))
}

// mpvAspectMode tracks the current aspect-ratio mode so the frontend can cycle
// through them. Default "original" (letterbox, keep aspect) matches mpv's
// default framing.
var mpvAspectMode = "original"

// mpvPlatformSetAspectMode applies one of: stretch | fill | original | 16:9 |
// 4:3 | 2.35. Modes react to window resizes automatically (keepaspect/panscan
// are continuous properties; video-aspect-override is only used for fixed
// ratios).
func mpvPlatformSetAspectMode(mode string) error {
	if mpvHandle == nil {
		return fmt.Errorf("mpv not started")
	}
	mpvAspectMode = mode
	setP := func(name, val string) error {
		cn := C.CString(name)
		cv := C.CString(val)
		defer C.free(unsafe.Pointer(cn))
		defer C.free(unsafe.Pointer(cv))
		return mpvRet(C.y_mpv_set_property_string(mpvHandle, cn, cv))
	}
	// Reset to default framing first.
	if err := setP("keepaspect", "yes"); err != nil {
		return err
	}
	if err := setP("panscan", "0"); err != nil {
		return err
	}
	if err := setP("video-aspect-override", "-2"); err != nil {
		return err
	}
	switch mode {
	case "stretch":
		return setP("keepaspect", "no")
	case "fill":
		return setP("panscan", "1")
	case "original":
		return nil
	case "16:9":
		return setP("video-aspect-override", "1.7777778")
	case "4:3":
		return setP("video-aspect-override", "1.3333333")
	case "2.35":
		return setP("video-aspect-override", "2.35")
	}
	return nil
}

func mpvPlatformGetAspectMode() string {
	return mpvAspectMode
}
