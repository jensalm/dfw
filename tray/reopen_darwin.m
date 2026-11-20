#import <Cocoa/Cocoa.h>

#include "_cgo_export.h"

@interface DFWReopenHandler : NSObject
- (void)handleReopen:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply;
@end

@implementation DFWReopenHandler
- (void)handleReopen:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply {
	dfwTrayReopened();
}
@end

static DFWReopenHandler *dfwReopenHandler;

// installed on the main thread once the tray is up, which is after
// NSApplication's finishLaunching has registered its own handlers, so this
// one is not overwritten.
void dfwInstallReopenHandler(void) {
	dispatch_async(dispatch_get_main_queue(), ^{
		if (dfwReopenHandler == nil) {
			dfwReopenHandler = [[DFWReopenHandler alloc] init];
		}
		[[NSAppleEventManager sharedAppleEventManager] setEventHandler:dfwReopenHandler
			andSelector:@selector(handleReopen:withReplyEvent:)
			forEventClass:kCoreEventClass
			andEventID:kAEReopenApplication];
	});
}
