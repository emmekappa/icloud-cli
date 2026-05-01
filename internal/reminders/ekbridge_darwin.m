#import <Foundation/Foundation.h>
#import <EventKit/EventKit.h>
#import <CoreGraphics/CoreGraphics.h>
#import "ekbridge_darwin.h"

static char* dupStr(NSString* s) {
    if (!s) return NULL;
    const char* utf8 = [s UTF8String];
    size_t len = strlen(utf8);
    char* buf = (char*)malloc(len + 1);
    memcpy(buf, utf8, len + 1);
    return buf;
}

static char* jsonCString(id obj, char** errOut) {
    NSError* jsonErr = nil;
    NSData* data = [NSJSONSerialization dataWithJSONObject:obj options:0 error:&jsonErr];
    if (!data) {
        if (errOut) *errOut = dupStr([jsonErr localizedDescription] ?: @"JSON serialization failed");
        return NULL;
    }
    NSString* s = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    return dupStr(s);
}

static NSString* hexColor(CGColorRef color) {
    if (!color) return nil;
    const CGFloat* comps = CGColorGetComponents(color);
    size_t n = CGColorGetNumberOfComponents(color);
    if (n < 3) return nil;
    int r = (int)(comps[0] * 255.0);
    int g = (int)(comps[1] * 255.0);
    int b = (int)(comps[2] * 255.0);
    return [NSString stringWithFormat:@"#%02X%02X%02X", r, g, b];
}

static NSDateFormatter* isoFormatter(void) {
    static NSDateFormatter* f = nil;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        f = [[NSDateFormatter alloc] init];
        f.locale = [NSLocale localeWithLocaleIdentifier:@"en_US_POSIX"];
        f.dateFormat = @"yyyy-MM-dd'T'HH:mm:ssZZZZZ";
        f.timeZone = [NSTimeZone localTimeZone];
    });
    return f;
}

static NSDateFormatter* dateOnlyFormatter(void) {
    static NSDateFormatter* f = nil;
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        f = [[NSDateFormatter alloc] init];
        f.locale = [NSLocale localeWithLocaleIdentifier:@"en_US_POSIX"];
        f.dateFormat = @"yyyy-MM-dd";
    });
    return f;
}

// Requests Reminders access synchronously. Returns a retained store on success
// or nil on denial (setting *errOut). Fast-paths the already-authorized case
// so repeated invocations never block on a semaphore.
static EKEventStore* requestAccessSync(char** errOut) {
    EKEventStore* store = [[EKEventStore alloc] init];

    EKAuthorizationStatus status = [EKEventStore authorizationStatusForEntityType:EKEntityTypeReminder];
    switch (status) {
        case EKAuthorizationStatusAuthorized:
            return store;
        case EKAuthorizationStatusRestricted:
            if (errOut) *errOut = dupStr(@"Reminders access is restricted by this device (MDM / parental controls). Cannot proceed.");
            return nil;
        case EKAuthorizationStatusDenied:
            if (errOut) *errOut = dupStr(@"Reminders access was previously denied. Open System Settings > Privacy & Security > Reminders and enable access for your terminal application.");
            return nil;
        case EKAuthorizationStatusNotDetermined:
        default:
            break;
    }

    dispatch_semaphore_t sema = dispatch_semaphore_create(0);
    __block BOOL granted = NO;
    __block NSError* authErr = nil;

    void (^handler)(BOOL, NSError*) = ^(BOOL g, NSError* e) {
        granted = g;
        authErr = e;
        dispatch_semaphore_signal(sema);
    };

    if (@available(macOS 14.0, *)) {
        [store requestFullAccessToRemindersWithCompletion:handler];
    } else {
        [store requestAccessToEntityType:EKEntityTypeReminder completion:handler];
    }
    dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);

    if (!granted) {
        NSString* m = authErr
            ? [authErr localizedDescription]
            : @"Reminders access not granted. Open System Settings > Privacy & Security > Reminders and enable access for your terminal application.";
        if (errOut) *errOut = dupStr(m);
        return nil;
    }
    return store;
}

// waitWithTimeout blocks on the semaphore up to `seconds`. Returns YES on
// signal, NO on timeout.
static BOOL waitWithTimeout(dispatch_semaphore_t sema, int64_t seconds) {
    dispatch_time_t deadline = dispatch_time(DISPATCH_TIME_NOW, seconds * NSEC_PER_SEC);
    return dispatch_semaphore_wait(sema, deadline) == 0;
}

static NSDictionary* reminderToDict(EKReminder* r) {
    NSMutableDictionary* d = [NSMutableDictionary dictionary];
    d[@"uid"] = r.calendarItemIdentifier ?: @"";
    d[@"title"] = r.title ?: @"";
    d[@"notes"] = r.notes ?: @"";
    d[@"url"] = r.URL ? r.URL.absoluteString : @"";
    d[@"list_id"] = r.calendar.calendarIdentifier ?: @"";
    d[@"list"] = r.calendar.title ?: @"";
    d[@"completed"] = @(r.completed);
    d[@"priority"] = @(r.priority);

    if (r.completionDate) {
        d[@"completed_at"] = [isoFormatter() stringFromDate:r.completionDate];
    }
    if (r.creationDate) {
        d[@"created_at"] = [isoFormatter() stringFromDate:r.creationDate];
    }
    if (r.lastModifiedDate) {
        d[@"modified_at"] = [isoFormatter() stringFromDate:r.lastModifiedDate];
    }

    NSDateComponents* due = r.dueDateComponents;
    if (due) {
        NSCalendar* cal = [NSCalendar currentCalendar];
        NSDate* dueDate = [cal dateFromComponents:due];
        BOOL hasTime = (due.hour != NSDateComponentUndefined);
        if (dueDate) {
            if (hasTime) {
                d[@"due"] = [isoFormatter() stringFromDate:dueDate];
                d[@"due_has_time"] = @YES;
            } else {
                d[@"due"] = [dateOnlyFormatter() stringFromDate:dueDate];
                d[@"due_has_time"] = @NO;
            }
        }
    }

    NSMutableArray* alarms = [NSMutableArray array];
    for (EKAlarm* a in (r.alarms ?: @[])) {
        NSMutableDictionary* ad = [NSMutableDictionary dictionary];
        if (a.absoluteDate) {
            ad[@"type"] = @"absolute";
            ad[@"at"] = [isoFormatter() stringFromDate:a.absoluteDate];
        } else {
            ad[@"type"] = @"relative";
            ad[@"offset_seconds"] = @(a.relativeOffset);
        }
        [alarms addObject:ad];
    }
    d[@"alarms"] = alarms;

    return d;
}

char* EKRequestAccess(char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;
        return dupStr(@"{\"granted\":true}");
    }
}

char* EKListLists(char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        NSArray<EKCalendar*>* lists = [store calendarsForEntityType:EKEntityTypeReminder];
        EKCalendar* defaultCal = [store defaultCalendarForNewReminders];
        NSString* defaultID = defaultCal ? defaultCal.calendarIdentifier : nil;

        NSMutableArray* out = [NSMutableArray array];
        for (EKCalendar* c in lists) {
            NSString* color = hexColor(c.CGColor);
            NSMutableDictionary* d = [NSMutableDictionary dictionary];
            d[@"id"] = c.calendarIdentifier ?: @"";
            d[@"title"] = c.title ?: @"";
            if (color) d[@"color"] = color;
            d[@"allows_modifications"] = @(c.allowsContentModifications);
            d[@"is_default"] = @([defaultID isEqualToString:c.calendarIdentifier]);
            d[@"source"] = c.source.title ?: @"";
            [out addObject:d];
        }
        return jsonCString(out, errOut);
    }
}

char* EKListReminders(const char* listID, int includeCompleted, char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        NSArray<EKCalendar*>* calendars = nil;
        if (listID && strlen(listID) > 0) {
            NSString* idStr = [NSString stringWithUTF8String:listID];
            EKCalendar* c = [store calendarWithIdentifier:idStr];
            if (!c) {
                if (errOut) *errOut = dupStr([NSString stringWithFormat:@"reminder list not found: %@", idStr]);
                return NULL;
            }
            calendars = @[c];
        }

        NSPredicate* pred = [store predicateForRemindersInCalendars:calendars];
        dispatch_semaphore_t sema = dispatch_semaphore_create(0);
        __block NSArray<EKReminder*>* result = nil;
        [store fetchRemindersMatchingPredicate:pred completion:^(NSArray<EKReminder*>* arr) {
            result = arr;
            dispatch_semaphore_signal(sema);
        }];
        if (!waitWithTimeout(sema, 60)) {
            if (errOut) *errOut = dupStr(@"timed out waiting for Reminders data — is the remindd daemon responsive?");
            return NULL;
        }

        NSMutableArray* out = [NSMutableArray array];
        for (EKReminder* r in (result ?: @[])) {
            if (!includeCompleted && r.completed) continue;
            [out addObject:reminderToDict(r)];
        }
        return jsonCString(out, errOut);
    }
}

char* EKGetReminder(const char* uid, char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        if (!uid) {
            if (errOut) *errOut = dupStr(@"uid required");
            return NULL;
        }
        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        NSString* uidStr = [NSString stringWithUTF8String:uid];
        EKCalendarItem* item = [store calendarItemWithIdentifier:uidStr];
        if (![item isKindOfClass:[EKReminder class]]) {
            return dupStr(@"null");
        }
        return jsonCString(reminderToDict((EKReminder*)item), errOut);
    }
}

// Parses the JSON "due" sub-object into an NSDateComponents. Returns nil if
// not present or malformed. Sets *hasTime based on whether "hour" is present.
static NSDateComponents* parseDueComponents(NSDictionary* input, BOOL* hasTime) {
    id dueRaw = input[@"due"];
    if (![dueRaw isKindOfClass:[NSDictionary class]]) return nil;
    NSDictionary* due = (NSDictionary*)dueRaw;

    NSDateComponents* dc = [[NSDateComponents alloc] init];
    dc.calendar = [NSCalendar currentCalendar];
    dc.timeZone = [NSTimeZone localTimeZone];

    NSNumber* y = due[@"year"];
    NSNumber* mo = due[@"month"];
    NSNumber* d = due[@"day"];
    if (!y || !mo || !d) return nil;
    dc.year = y.integerValue;
    dc.month = mo.integerValue;
    dc.day = d.integerValue;

    NSNumber* h = due[@"hour"];
    NSNumber* mi = due[@"minute"];
    if (h) {
        dc.hour = h.integerValue;
        dc.minute = mi ? mi.integerValue : 0;
        if (hasTime) *hasTime = YES;
    } else {
        if (hasTime) *hasTime = NO;
    }
    return dc;
}

// Applies a "due" field change to a reminder. When hasTime is YES, a single
// absolute alarm is installed at that moment (matching Reminders.app default
// for timed reminders). In the date-only case alarms are left untouched so we
// don't silently wipe any pre-existing user alarms on update.
static void applyDueToReminder(EKReminder* r, NSDateComponents* dc, BOOL hasTime) {
    r.dueDateComponents = dc;
    if (!dc || !hasTime) return;

    NSDate* d = [[NSCalendar currentCalendar] dateFromComponents:dc];
    if (!d) return;
    r.alarms = @[[EKAlarm alarmWithAbsoluteDate:d]];
}

char* EKCreateReminder(const char* jsonInput, char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        if (!jsonInput) {
            if (errOut) *errOut = dupStr(@"jsonInput required");
            return NULL;
        }

        NSData* inData = [[NSString stringWithUTF8String:jsonInput] dataUsingEncoding:NSUTF8StringEncoding];
        NSError* parseErr = nil;
        NSDictionary* input = [NSJSONSerialization JSONObjectWithData:inData options:0 error:&parseErr];
        if (![input isKindOfClass:[NSDictionary class]]) {
            if (errOut) *errOut = dupStr(parseErr ? [parseErr localizedDescription] : @"invalid JSON input");
            return NULL;
        }

        NSString* title = input[@"title"];
        if (!title || title.length == 0) {
            if (errOut) *errOut = dupStr(@"title is required");
            return NULL;
        }

        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        EKCalendar* target = nil;
        NSString* listID = input[@"list_id"];
        if (listID && listID.length > 0) {
            target = [store calendarWithIdentifier:listID];
            if (!target) {
                if (errOut) *errOut = dupStr([NSString stringWithFormat:@"reminder list not found: %@", listID]);
                return NULL;
            }
        } else {
            target = [store defaultCalendarForNewReminders];
            if (!target) {
                if (errOut) *errOut = dupStr(@"no default reminder list configured in the system");
                return NULL;
            }
        }
        if (!target.allowsContentModifications) {
            if (errOut) *errOut = dupStr([NSString stringWithFormat:@"list \"%@\" does not allow modifications", target.title]);
            return NULL;
        }

        EKReminder* r = [EKReminder reminderWithEventStore:store];
        r.calendar = target;
        r.title = title;

        id notes = input[@"notes"];
        if ([notes isKindOfClass:[NSString class]] && [notes length] > 0) r.notes = notes;

        id url = input[@"url"];
        if ([url isKindOfClass:[NSString class]] && [url length] > 0) {
            r.URL = [NSURL URLWithString:url];
        }

        id prio = input[@"priority"];
        if ([prio isKindOfClass:[NSNumber class]]) r.priority = [prio integerValue];

        BOOL hasTime = NO;
        NSDateComponents* dc = parseDueComponents(input, &hasTime);
        if (dc) applyDueToReminder(r, dc, hasTime);

        NSError* saveErr = nil;
        if (![store saveReminder:r commit:YES error:&saveErr]) {
            if (errOut) *errOut = dupStr(saveErr ? [saveErr localizedDescription] : @"save failed");
            return NULL;
        }

        return jsonCString(reminderToDict(r), errOut);
    }
}

char* EKUpdateReminder(const char* uid, const char* jsonInput, char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        if (!uid || !jsonInput) {
            if (errOut) *errOut = dupStr(@"uid and jsonInput required");
            return NULL;
        }

        NSData* inData = [[NSString stringWithUTF8String:jsonInput] dataUsingEncoding:NSUTF8StringEncoding];
        NSError* parseErr = nil;
        NSDictionary* input = [NSJSONSerialization JSONObjectWithData:inData options:0 error:&parseErr];
        if (![input isKindOfClass:[NSDictionary class]]) {
            if (errOut) *errOut = dupStr(parseErr ? [parseErr localizedDescription] : @"invalid JSON input");
            return NULL;
        }

        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        NSString* uidStr = [NSString stringWithUTF8String:uid];
        EKCalendarItem* item = [store calendarItemWithIdentifier:uidStr];
        if (![item isKindOfClass:[EKReminder class]]) {
            if (errOut) *errOut = dupStr([NSString stringWithFormat:@"reminder not found: %@", uidStr]);
            return NULL;
        }
        EKReminder* r = (EKReminder*)item;

        id title = input[@"title"];
        if ([title isKindOfClass:[NSString class]] && [title length] > 0) r.title = title;

        id notes = input[@"notes"];
        if ([notes isKindOfClass:[NSString class]]) r.notes = [notes length] > 0 ? notes : nil;

        id url = input[@"url"];
        if ([url isKindOfClass:[NSString class]]) {
            r.URL = [url length] > 0 ? [NSURL URLWithString:url] : nil;
        }

        id prio = input[@"priority"];
        if ([prio isKindOfClass:[NSNumber class]]) r.priority = [prio integerValue];

        id listID = input[@"list_id"];
        if ([listID isKindOfClass:[NSString class]] && [listID length] > 0) {
            EKCalendar* newCal = [store calendarWithIdentifier:listID];
            if (!newCal) {
                if (errOut) *errOut = dupStr([NSString stringWithFormat:@"reminder list not found: %@", listID]);
                return NULL;
            }
            if (!newCal.allowsContentModifications) {
                if (errOut) *errOut = dupStr([NSString stringWithFormat:@"list \"%@\" does not allow modifications", newCal.title]);
                return NULL;
            }
            r.calendar = newCal;
        }

        id clearDue = input[@"clear_due"];
        if ([clearDue isKindOfClass:[NSNumber class]] && [clearDue boolValue]) {
            r.dueDateComponents = nil;
            r.alarms = @[];
        } else if (input[@"due"]) {
            BOOL hasTime = NO;
            NSDateComponents* dc = parseDueComponents(input, &hasTime);
            if (dc) applyDueToReminder(r, dc, hasTime);
        }

        NSError* saveErr = nil;
        if (![store saveReminder:r commit:YES error:&saveErr]) {
            if (errOut) *errOut = dupStr(saveErr ? [saveErr localizedDescription] : @"save failed");
            return NULL;
        }

        return jsonCString(reminderToDict(r), errOut);
    }
}

char* EKSetCompleted(const char* uid, int completed, char** errOut) {
    @autoreleasepool {
        if (errOut) *errOut = NULL;
        if (!uid) {
            if (errOut) *errOut = dupStr(@"uid required");
            return NULL;
        }

        EKEventStore* store = requestAccessSync(errOut);
        if (!store) return NULL;

        NSString* uidStr = [NSString stringWithUTF8String:uid];
        EKCalendarItem* item = [store calendarItemWithIdentifier:uidStr];
        if (![item isKindOfClass:[EKReminder class]]) {
            if (errOut) *errOut = dupStr([NSString stringWithFormat:@"reminder not found: %@", uidStr]);
            return NULL;
        }
        EKReminder* r = (EKReminder*)item;
        r.completed = (completed != 0);

        NSError* saveErr = nil;
        if (![store saveReminder:r commit:YES error:&saveErr]) {
            if (errOut) *errOut = dupStr(saveErr ? [saveErr localizedDescription] : @"save failed");
            return NULL;
        }

        return jsonCString(reminderToDict(r), errOut);
    }
}
