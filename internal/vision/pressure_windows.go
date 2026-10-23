//go:build windows

package vision

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var getProcessMemoryInfo = windows.NewLazySystemDLL("kernel32.dll").NewProc("K32GetProcessMemoryInfo")

type processMemoryCounters struct {
	Size                       uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// treeRSSKB reads native Windows working sets for a snapshot of the process tree.
// Exited or inaccessible descendants contribute no sample, as with procfs.
func treeRSSKB(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	children := map[uint32][]uint32{}
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err == nil {
		var entry windows.ProcessEntry32
		entry.Size = uint32(unsafe.Sizeof(entry))
		for err = windows.Process32First(snapshot, &entry); err == nil; err = windows.Process32Next(snapshot, &entry) {
			children[entry.ParentProcessID] = append(children[entry.ParentProcessID], entry.ProcessID)
		}
		windows.CloseHandle(snapshot)
	}
	seen := map[uint32]bool{}
	queue := []uint32{uint32(pid)}
	var total int64
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id] {
			continue
		}
		seen[id] = true
		queue = append(queue, children[id]...)
		handle, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, id)
		if err != nil {
			continue
		}
		counters := processMemoryCounters{}
		counters.Size = uint32(unsafe.Sizeof(counters))
		ok, _, _ := getProcessMemoryInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&counters)), uintptr(counters.Size))
		windows.CloseHandle(handle)
		if ok != 0 {
			total += int64(counters.WorkingSetSize / 1024)
		}
	}
	return total
}
