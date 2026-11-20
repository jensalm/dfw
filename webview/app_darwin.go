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

// its own pool, drained before it returns: whatever must outlive the call is
// held by something other than autorelease.
static void dfw_apply_app(void *window) {
	@autoreleasepool {
		NSWindow *w = (__bridge NSWindow *)window;
		NSMenu *windows = nil;
		[NSApp setMainMenu:dfw_main_menu(w, &windows)];
		[NSApp setWindowsMenu:windows];
		dfw_own_ui_delegate(w);
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

// a window that is never shown, holding a web view as webview_go's does,
// for the tests to apply the menu and the delegate to.
static void *dfw_new_hidden_window(void) {
	[NSApplication sharedApplication];
	NSWindow *w = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 100, 100)
		styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		backing:NSBackingStoreBuffered
		defer:YES];
	[w setReleasedWhenClosed:NO];
	[w setContentView:[[WKWebView alloc] initWithFrame:NSMakeRect(0, 0, 100, 100)
		configuration:[[WKWebViewConfiguration alloc] init]]];
	return (__bridge_retained void *)w;
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

// newNativeHiddenWindow, describeNativeUIDelegate and describeNativeMainMenu
// serve the tests, which cannot use cgo themselves.
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
