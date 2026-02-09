package lock

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
	procOpenProcess  = modkernel32.NewProc("OpenProcess")
	procCloseHandle  = modkernel32.NewProc("CloseHandle")
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
	synchronize             = 0x00100000
)

func lockFile(f *os.File) error {
	handle := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	r1, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockFile(f *os.File) error {
	handle := syscall.Handle(f.Fd())
	ol := new(syscall.Overlapped)
	r1, _, err := procUnlockFileEx.Call(
		uintptr(handle),
		0,
		1,
		0,
		uintptr(unsafe.Pointer(ol)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func checkProcessAlive(proc *os.Process) bool {
	h, _, _ := procOpenProcess.Call(uintptr(synchronize), 0, uintptr(proc.Pid))
	if h == 0 {
		return false
	}
	procCloseHandle.Call(h)
	return true
}
