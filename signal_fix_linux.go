//go:build linux

package main

/*
#include <signal.h>
#include <string.h>
#include <pthread.h>
#include <unistd.h>


// Re-apply SA_ONSTACK on critical signals. WebKitGTK/JSC lazily installs its
// own handlers WITHOUT SA_ONSTACK when JS first runs, which clobbers Go's
// runtime handlers and causes:
//   fatal error: non-Go code set up signal handler without SA_ONSTACK flag
// Wails >= 2.13 re-applies for ~5s; we keep going longer as a safety net on
// WebKitGTK 2.52+ (see wailsapp/wails#5506).
static void yaria_fix_one(int signo) {
	struct sigaction sa;
	memset(&sa, 0, sizeof(sa));
	if (sigaction(signo, NULL, &sa) != 0) {
		return;
	}
	if (sa.sa_flags & SA_ONSTACK) {
		return;
	}
	sa.sa_flags |= SA_ONSTACK;
	sigaction(signo, &sa, NULL);
}

static void yaria_fix_signals(void) {
	// Do NOT touch SIGUSR1 — WebKit GC uses it; forcing SA_ONSTACK there can freeze (wails#5530).
	const int sigs[] = { SIGSEGV, SIGBUS, SIGFPE, SIGILL, SIGABRT, SIGTRAP };
	for (unsigned i = 0; i < sizeof(sigs)/sizeof(sigs[0]); i++) {
		yaria_fix_one(sigs[i]);
	}
}

static void *yaria_signal_fix_thread(void *arg) {
	(void)arg;
	// Aggressive for first ~10s (covers JSC lazy init), then slow for another ~50s.
	for (int i = 0; i < 200; i++) {
		yaria_fix_signals();
		usleep(50 * 1000); // 50ms
	}
	for (int i = 0; i < 50; i++) {
		yaria_fix_signals();
		usleep(1000 * 1000); // 1s
	}
	return NULL;
}

static void yaria_start_signal_fix(void) __attribute__((constructor));
static void yaria_start_signal_fix(void) {
	yaria_fix_signals();
	pthread_t t;
	pthread_attr_t attr;
	pthread_attr_init(&attr);
	pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
	pthread_create(&t, &attr, yaria_signal_fix_thread, NULL);
	pthread_attr_destroy(&attr);
}
*/
import "C"
