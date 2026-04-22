//go:build darwin && cgo

package reminders

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework EventKit -framework CoreGraphics
#include <stdlib.h>
#include "ekbridge_darwin.h"
*/
import "C"

import (
	"encoding/json"
	"errors"
	"unsafe"
)

type ekBackend struct{}

func newBackend() Backend {
	return &ekBackend{}
}

// consume releases the C strings returned by the bridge and turns them into a
// Go error (on failure) or a Go byte slice (on success). It MUST be the only
// owner of both pointers after being called.
func consume(cs *C.char, cerr *C.char) ([]byte, error) {
	if cs == nil {
		if cerr == nil {
			return nil, errors.New("reminders: unknown bridge error")
		}
		defer C.free(unsafe.Pointer(cerr))
		return nil, errors.New(C.GoString(cerr))
	}
	defer C.free(unsafe.Pointer(cs))
	if cerr != nil {
		C.free(unsafe.Pointer(cerr))
	}
	return []byte(C.GoString(cs)), nil
}

func unmarshalInto[T any](data []byte, out *T) error {
	return json.Unmarshal(data, out)
}

func decode[T any](data []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	var out T
	if err := unmarshalInto(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func decodeList[T any](data []byte, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	var out []T
	if err := unmarshalInto(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (ekBackend) RequestAccess() error {
	var cerr *C.char
	_, err := consume(C.EKRequestAccess(&cerr), cerr)
	return err
}

func (ekBackend) ListLists() ([]List, error) {
	var cerr *C.char
	return decodeList[List](consume(C.EKListLists(&cerr), cerr))
}

func (ekBackend) ListReminders(listID string, includeCompleted bool) ([]Reminder, error) {
	cid := C.CString(listID)
	defer C.free(unsafe.Pointer(cid))
	incl := C.int(0)
	if includeCompleted {
		incl = 1
	}
	var cerr *C.char
	return decodeList[Reminder](consume(C.EKListReminders(cid, incl, &cerr), cerr))
}

func (ekBackend) GetReminder(uid string) (*Reminder, error) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	var cerr *C.char
	data, err := consume(C.EKGetReminder(cuid, &cerr), cerr)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return nil, ErrNotFound
	}
	return decode[Reminder](data, nil)
}

func (ekBackend) Create(in CreateInput) (*Reminder, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cp := C.CString(string(payload))
	defer C.free(unsafe.Pointer(cp))
	var cerr *C.char
	return decode[Reminder](consume(C.EKCreateReminder(cp, &cerr), cerr))
}

func (ekBackend) Update(uid string, in UpdateInput) (*Reminder, error) {
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	cp := C.CString(string(payload))
	defer C.free(unsafe.Pointer(cp))
	var cerr *C.char
	return decode[Reminder](consume(C.EKUpdateReminder(cuid, cp, &cerr), cerr))
}

func (ekBackend) SetCompleted(uid string, completed bool) (*Reminder, error) {
	cuid := C.CString(uid)
	defer C.free(unsafe.Pointer(cuid))
	c := C.int(0)
	if completed {
		c = 1
	}
	var cerr *C.char
	return decode[Reminder](consume(C.EKSetCompleted(cuid, c, &cerr), cerr))
}
