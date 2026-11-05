package pkgid

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	getCurrentPackageFamilyName = kernel32.NewProc("GetCurrentPackageFamilyName")

	once   sync.Once
	family string
)

func FamilyName() string {
	once.Do(func() {
		if getCurrentPackageFamilyName.Find() != nil {
			return
		}
		var n uint32 = 256
		buf := make([]uint16, n)
		r, _, _ := getCurrentPackageFamilyName.Call(uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&buf[0])))
		if r != 0 {
			return
		}
		family = syscall.UTF16ToString(buf[:n])
	})
	return family
}
