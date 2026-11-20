//go:build darwin

package webview

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#include <stdlib.h>
#include <string.h>
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <objc/runtime.h>

// the name a bundle gives itself, else the process name: what macOS shows
// as the application menu's title, so Hide and Quit say the same.
static NSString *dfw_app_name(void) {
	NSString *name = [[NSBundle mainBundle] objectForInfoDictionaryKey:@"CFBundleName"];
	if ([name length] == 0) {
		name = [[NSProcessInfo processInfo] processName];
	}
	return name;
}

static NSMenuItem *dfw_menu_item(NSMenu *menu, NSString *title, SEL action, NSString *key, NSEventModifierFlags mods) {
	NSMenuItem *item = [menu addItemWithTitle:title action:action keyEquivalent:key];
	if (mods != 0) {
		[item setKeyEquivalentModifierMask:mods];
	}
	return item;
}

// the standard menus a window needs on macOS. without an Edit menu, WebKit
// receives no Cut, Copy, Paste, Select All or Undo key equivalents. Quit
// closes window, the one this process owns, rather than terminating the
// application, so the run loop ends the way the close button ends it.
static NSMenu *dfw_main_menu(NSWindow *window, NSMenu **windowsMenu) {
	NSString *name = dfw_app_name();
	NSEventModifierFlags cmd = NSEventModifierFlagCommand;
	NSMenu *bar = [[NSMenu alloc] init];

	NSMenuItem *appItem = [bar addItemWithTitle:@"" action:nil keyEquivalent:@""];
	NSMenu *app = [[NSMenu alloc] initWithTitle:name];
	dfw_menu_item(app, [@"About " stringByAppendingString:name], @selector(orderFrontStandardAboutPanel:), @"", 0);
	[app addItem:[NSMenuItem separatorItem]];
	dfw_menu_item(app, [@"Hide " stringByAppendingString:name], @selector(hide:), @"h", 0);
	dfw_menu_item(app, @"Hide Others", @selector(hideOtherApplications:), @"h", cmd | NSEventModifierFlagOption);
	dfw_menu_item(app, @"Show All", @selector(unhideAllApplications:), @"", 0);
	[app addItem:[NSMenuItem separatorItem]];
	NSMenuItem *quit = dfw_menu_item(app, [@"Quit " stringByAppendingString:name], @selector(performClose:), @"q", 0);
	[quit setTarget:window];
	[appItem setSubmenu:app];

	NSMenuItem *editItem = [bar addItemWithTitle:@"" action:nil keyEquivalent:@""];
	NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Edit"];
	dfw_menu_item(edit, @"Undo", @selector(undo:), @"z", 0);
	dfw_menu_item(edit, @"Redo", @selector(redo:), @"z", cmd | NSEventModifierFlagShift);
	[edit addItem:[NSMenuItem separatorItem]];
	dfw_menu_item(edit, @"Cut", @selector(cut:), @"x", 0);
	dfw_menu_item(edit, @"Copy", @selector(copy:), @"c", 0);
	dfw_menu_item(edit, @"Paste", @selector(paste:), @"v", 0);
	dfw_menu_item(edit, @"Paste and Match Style", @selector(pasteAsPlainText:), @"v", cmd | NSEventModifierFlagShift | NSEventModifierFlagOption);
	dfw_menu_item(edit, @"Select All", @selector(selectAll:), @"a", 0);
	[editItem setSubmenu:edit];

	NSMenuItem *windowItem = [bar addItemWithTitle:@"" action:nil keyEquivalent:@""];
	NSMenu *windows = [[NSMenu alloc] initWithTitle:@"Window"];
	dfw_menu_item(windows, @"Minimize", @selector(performMiniaturize:), @"m", 0);
	dfw_menu_item(windows, @"Zoom", @selector(performZoom:), @"", 0);
	[windows addItem:[NSMenuItem separatorItem]];
	dfw_menu_item(windows, @"Close", @selector(performClose:), @"w", 0);
	[windowItem setSubmenu:windows];

	*windowsMenu = windows;
	return bar;
}

static WKWebView *dfw_find_web_view(NSView *view) {
	if ([view isKindOfClass:[WKWebView class]]) {
		return (WKWebView *)view;
	}
	for (NSView *sub in [view subviews]) {
		WKWebView *found = dfw_find_web_view(sub);
		if (found != nil) {
			return found;
		}
	}
	return nil;
}

// DFWUIDelegate is the web view's UI delegate on macOS, in place of
// webview_go's. that one is autoreleased as it is set, and WKWebView holds
// its delegate weakly, so it is gone before the run loop turns: the file
// chooser never opens and new-window links are dropped. this one is held by
// the web view itself (dfw_own_ui_delegate) and answers both.
@interface DFWUIDelegate : NSObject <WKUIDelegate>
@end

@implementation DFWUIDelegate

// <input type="file">: a sheet on the window. a webkitdirectory input
// chooses folders, any other chooses files.
- (void)webView:(WKWebView *)webView
	runOpenPanelWithParameters:(WKOpenPanelParameters *)parameters
	initiatedByFrame:(WKFrameInfo *)frame
	completionHandler:(void (^)(NSArray<NSURL *> *URLs))completionHandler {
	NSOpenPanel *panel = [NSOpenPanel openPanel];
	BOOL directories = [parameters allowsDirectories];
	[panel setCanChooseDirectories:directories];
	[panel setCanChooseFiles:!directories];
	[panel setAllowsMultipleSelection:[parameters allowsMultipleSelection]];
	void (^done)(NSModalResponse) = ^(NSModalResponse response) {
		completionHandler(response == NSModalResponseOK ? [panel URLs] : nil);
	};
	NSWindow *window = [webView window];
	if (window != nil) {
		[panel beginSheetModalForWindow:window completionHandler:done];
		return;
	}
	done([panel runModal]);
}

// a link with target="_blank", or window.open: an http, https or mailto
// address opens in the user's default handler. no second web view is made.
- (WKWebView *)webView:(WKWebView *)webView
	createWebViewWithConfiguration:(WKWebViewConfiguration *)configuration
	forNavigationAction:(WKNavigationAction *)action
	windowFeatures:(WKWindowFeatures *)features {
	NSURL *url = [[action request] URL];
	NSString *scheme = [[url scheme] lowercaseString];
	if ([scheme isEqualToString:@"http"] || [scheme isEqualToString:@"https"] || [scheme isEqualToString:@"mailto"]) {
		[[NSWorkspace sharedWorkspace] openURL:url];
	}
	return nil;
}

@end

static char dfw_ui_delegate_key;

static void dfw_own_ui_delegate(NSWindow *window) {
	WKWebView *web = dfw_find_web_view([window contentView]);
	if (web == nil) {
		return;
	}
	DFWUIDelegate *delegate = [[DFWUIDelegate alloc] init];
	objc_setAssociatedObject(web, &dfw_ui_delegate_key, delegate, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
	[web setUIDelegate:delegate];
}

// the web view's delegate class, for the tests.
static char *dfw_describe_ui_delegate(void *window) {
	WKWebView *web = dfw_find_web_view([(__bridge NSWindow *)window contentView]);
	id delegate = [web UIDelegate];
	if (delegate == nil) {
		return strdup("nil");
	}
	NSString *s = [NSString stringWithFormat:@"%@ openPanel=%d newWindow=%d", NSStringFromClass([delegate class]),
		[delegate respondsToSelector:@selector(webView:runOpenPanelWithParameters:initiatedByFrame:completionHandler:)],
		[delegate respondsToSelector:@selector(webView:createWebViewWithConfiguration:forNavigationAction:windowFeatures:)]];
	return strdup([s UTF8String]);
}

// a bundle with LSUIElement, as a menu bar daemon has, makes every process
// it starts an accessory, and webview_go leaves a bundled application's
// activation policy alone. the window process is the visible application,
// so it takes the regular policy: a Dock icon and a menu bar.
static void dfw_prepare_app(void) {
	[[NSApplication sharedApplication] setActivationPolicy:NSApplicationActivationPolicyRegular];
}

nm// DFWQuitHandler answers the quit Apple Event, which the Dock's Quit, a
// logout or restart, and `quit app` from a script send. NSApplication's own
// handler calls terminate:, which exits inside the run loop before Window or
// Run returns, so the window state, the server's shutdown and every deferred
// cleanup are skipped. this one closes the window, as Quit in the menu does.
// it is a handler and not applicationShouldTerminate:, because cancelling
// there cancels a logout too, and a later reply would be due after the run
// loop that has to carry it has ended.
@interface DFWQuitHandler : NSObject
@property (weak) NSWindow *window;
- (void)handleQuit:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply;
@end

@implementation DFWQuitHandler
- (void)handleQuit:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply {
	[[self window] performClose:nil];
}
@end

// the Apple Event manager does not hold its handlers.
static DFWQuitHandler *dfw_quit_handler;

// NSApplication registers its own handler in finishLaunching, which would
// replace this one. it has run by now: webview_go's constructor runs the
// application until applicationDidFinishLaunching:, and the run loop that
// Run enters later does not register it again.
static void dfw_install_quit_handler(NSWindow *window) {
	if (dfw_quit_handler == nil) {
		dfw_quit_handler = [[DFWQuitHandler alloc] init];
	}
	[dfw_quit_handler setWindow:window];
	[[NSAppleEventManager sharedAppleEventManager] setEventHandler:dfw_quit_handler
		andSelector:@selector(handleQuit:withReplyEvent:)
		forEventClass:kCoreEventClass
		andEventID:kAEQuitApplication];
}

// webview_go's window is titled only, and gets its close and minimize
// buttons from SetSize, which dfw calls only for a size it knows. a window
// without them ignores performClose: and performMiniaturize:, so Quit, Close,
// Minimize and a quit from the Dock or a logout would all do nothing.
static void dfw_make_closable(NSWindow *window) {
	[window setStyleMask:[window styleMask] | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable];
}

// its own pool, drained before it returns: whatever must outlive the call is
// held by something other than autorelease.
static void dfw_apply_app(void *window) {
	@autoreleasepool {
		NSWindow *w = (__bridge NSWindow *)window;
		dfw_make_closable(w);
		NSMenu *windows = nil;
		[NSApp setMainMenu:dfw_main_menu(w, &windows)];
		[NSApp setWindowsMenu:windows];
		dfw_own_ui_delegate(w);
		dfw_install_quit_handler(w);
	}
}

static void dfw_front(void) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	// cooperative activation ([NSApp activate]) is refused here: nothing
	// yields to a process the daemon started. this one is honoured.
	[NSApp activateIgnoringOtherApps:YES];
#pragma clang diagnostic pop
	for (NSWindow *w in [NSApp windows]) {
		if (![w canBecomeMainWindow]) {
			continue;
		}
		[w makeKeyAndOrderFront:nil];
		WKWebView *web = dfw_find_web_view([w contentView]);
		if (web != nil) {
			[w makeFirstResponder:web];
		}
	}
}

// brings the window to the front with the keyboard in the page, once the run
// loop turns. a window opened from a tray menu loses the activation to the
// application that was active before the menu, as the menu closes, so it is
// taken once more if it was lost within the first moment.
static void dfw_activate_app(void) {
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 200 * NSEC_PER_MSEC), dispatch_get_main_queue(), ^{
		dfw_front();
	});
	dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 800 * NSEC_PER_MSEC), dispatch_get_main_queue(), ^{
		if (![NSApp isActive]) {
			dfw_front();
		}
	});
}

// a window that is never shown, titled only and holding a web view as
// webview_go's is, for the tests to apply the menu and the delegate to.
static void *dfw_new_hidden_window(void) {
	[NSApplication sharedApplication];
	NSWindow *w = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 100, 100)
		styleMask:NSWindowStyleMaskTitled
		backing:NSBackingStoreBuffered
		defer:YES];
	[w setReleasedWhenClosed:NO];
	[w setContentView:[[WKWebView alloc] initWithFrame:NSMakeRect(0, 0, 100, 100)
		configuration:[[WKWebViewConfiguration alloc] init]]];
	return (__bridge_retained void *)w;
}

// the window's buttons, for the tests.
static char *dfw_describe_window_style(void *window) {
	NSWindowStyleMask mask = [(__bridge NSWindow *)window styleMask];
	NSString *s = [NSString stringWithFormat:@"closable=%d miniaturizable=%d",
		(mask & NSWindowStyleMaskClosable) != 0, (mask & NSWindowStyleMaskMiniaturizable) != 0];
	return strdup([s UTF8String]);
}

// records the window being asked to close, and keeps it open.
@interface DFWCloseRecorder : NSObject <NSWindowDelegate>
@property int asked;
@end

@implementation DFWCloseRecorder
- (BOOL)windowShouldClose:(NSWindow *)window {
	[self setAsked:[self asked] + 1];
	return NO;
}
@end

// sends this process the quit Apple Event, as the Dock's Quit does, and
// counts the times the window is asked to close, for the tests.
static int dfw_quit_event_close_requests(void *window) {
	NSWindow *w = (__bridge NSWindow *)window;
	DFWCloseRecorder *recorder = [[DFWCloseRecorder alloc] init];
	id<NSWindowDelegate> previous = [w delegate];
	[w setDelegate:recorder];
	NSAppleEventDescriptor *event = [NSAppleEventDescriptor appleEventWithEventClass:kCoreEventClass
		eventID:kAEQuitApplication
		targetDescriptor:[NSAppleEventDescriptor currentProcessDescriptor]
		returnID:kAutoGenerateReturnID
		transactionID:kAnyTransactionID];
	AppleEvent reply = {typeNull, NULL};
	// the reference constant goes unused, but may not be null.
	[[NSAppleEventManager sharedAppleEventManager] dispatchRawAppleEvent:[event aeDesc] withRawReply:&reply handlerRefCon:(__bridge void *)w];
	AEDisposeDesc(&reply);
	[w setDelegate:previous];
	return [recorder asked];
}

// describes the main menu as "Menu: Title=mods+key/action>target; ...|...",
// for the tests.
static char *dfw_describe_main_menu(void) {
	NSMutableArray *menus = [NSMutableArray array];
	for (NSMenuItem *top in [[NSApp mainMenu] itemArray]) {
		NSMenu *menu = [top submenu];
		NSMutableArray *items = [NSMutableArray array];
		for (NSMenuItem *item in [menu itemArray]) {
			if ([item isSeparatorItem]) {
				continue;
			}
			NSEventModifierFlags m = [item keyEquivalentModifierMask];
			NSString *mods = [NSString stringWithFormat:@"%@%@%@%@",
				(m & NSEventModifierFlagControl) ? @"ctrl+" : @"",
				(m & NSEventModifierFlagOption) ? @"opt+" : @"",
				(m & NSEventModifierFlagShift) ? @"shift+" : @"",
				(m & NSEventModifierFlagCommand) ? @"cmd+" : @""];
			NSString *target = [item target] != nil ? NSStringFromClass([[item target] class]) : @"responder";
			[items addObject:[NSString stringWithFormat:@"%@=%@%@/%@>%@", [item title], mods, [item keyEquivalent],
				NSStringFromSelector([item action]), target]];
		}
		[menus addObject:[NSString stringWithFormat:@"%@: %@", [menu title], [items componentsJoinedByString:@"; "]]];
	}
	return strdup([[menus componentsJoinedByString:@"|"] UTF8String]);
}
*/
import "C"

import "unsafe"

// prepareNativeApp runs before the webview exists, on the UI thread.
func prepareNativeApp() {
	C.dfw_prepare_app()
}

// applyNativeApp runs once the webview and its window exist, on the UI
// thread.
func applyNativeApp(window unsafe.Pointer) {
	if window == nil {
		return
	}
	C.dfw_apply_app(window)
}

// activateNativeApp runs on the UI thread as the run loop starts.
func activateNativeApp() {
	C.dfw_activate_app()
}

// newNativeHiddenWindow, the describeNative helpers and
// nativeQuitEventCloseRequests serve the tests, which cannot use cgo
// themselves.
func newNativeHiddenWindow() unsafe.Pointer {
	return C.dfw_new_hidden_window()
}

func describeNativeUIDelegate(window unsafe.Pointer) string {
	description := C.dfw_describe_ui_delegate(window)
	defer C.free(unsafe.Pointer(description))
	return C.GoString(description)
}

func describeNativeMainMenu() string {
	description := C.dfw_describe_main_menu()
	defer C.free(unsafe.Pointer(description))
	return C.GoString(description)
}

func describeNativeWindowStyle(window unsafe.Pointer) string {
	description := C.dfw_describe_window_style(window)
	defer C.free(unsafe.Pointer(description))
	return C.GoString(description)
}

func nativeQuitEventCloseRequests(window unsafe.Pointer) int {
	return int(C.dfw_quit_event_close_requests(window))
}
