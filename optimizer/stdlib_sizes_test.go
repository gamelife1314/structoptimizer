package optimizer

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"
)

// TestStdlibTypeSizesReal verifies that the hardcoded type sizes in stdlibTypeSizes
// match the actual unsafe.Sizeof values for the current Go version and platform.
func TestStdlibTypeSizesReal(t *testing.T) {
	testCases := []struct {
		pkg      string
		typeName string
		size     int64
	}{
		{"time", "Time", int64(unsafe.Sizeof(time.Time{}))},
		{"time", "Duration", int64(unsafe.Sizeof(time.Duration(0)))},
		{"time", "Location", int64(unsafe.Sizeof((*time.Location)(nil)))},
		{"sync", "Mutex", int64(unsafe.Sizeof(sync.Mutex{}))},
		{"sync", "RWMutex", int64(unsafe.Sizeof(sync.RWMutex{}))},
		{"sync", "WaitGroup", int64(unsafe.Sizeof(sync.WaitGroup{}))},
		{"sync", "Cond", int64(unsafe.Sizeof((*sync.Cond)(nil)))},
		{"sync", "Once", int64(unsafe.Sizeof(sync.Once{}))},
		{"context", "Context", int64(reflectSizeOfInterface[context.Context]())},
		{"context", "CancelFunc", int64(unsafe.Sizeof(context.CancelFunc(nil)))},
		{"bytes", "Buffer", int64(unsafe.Sizeof(bytes.Buffer{}))},
		{"strings", "Builder", int64(unsafe.Sizeof(strings.Builder{}))},
		{"net", "IP", int64(unsafe.Sizeof(net.IP(nil)))},
		{"net", "IPMask", int64(unsafe.Sizeof(net.IPMask(nil)))},
		{"url", "URL", int64(unsafe.Sizeof(neturl.URL{}))},
		{"http", "Request", int64(unsafe.Sizeof(http.Request{}))},
		{"http", "Response", int64(unsafe.Sizeof(http.Response{}))},
		{"http", "Header", int64(unsafe.Sizeof(http.Header(nil)))},
		{"json", "RawMessage", int64(unsafe.Sizeof(json.RawMessage(nil)))},
	}

	for _, tc := range testCases {
		if pkgTypes, ok := stdlibTypeSizes[tc.pkg]; ok {
			if ts, ok := pkgTypes[tc.typeName]; ok {
				if ts.Size != tc.size {
					t.Errorf("stdlibTypeSizes[%q][%q].Size = %d, want %d (actual unsafe.Sizeof)",
						tc.pkg, tc.typeName, ts.Size, tc.size)
				}
			} else {
				t.Errorf("stdlibTypeSizes missing entry for [%q][%q]", tc.pkg, tc.typeName)
			}
		} else {
			t.Errorf("stdlibTypeSizes missing package %q", tc.pkg)
		}
	}
}

// reflectSizeOfInterface returns the size of an interface value (2 words on 64-bit).
func reflectSizeOfInterface[T any]() int64 {
	var v T
	return int64(unsafe.Sizeof(v))
}
