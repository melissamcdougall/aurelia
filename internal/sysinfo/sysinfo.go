package sysinfo

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// SystemResources holds a point-in-time snapshot of system resource usage.
type SystemResources struct {
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`

	MemTotalBytes int64   `json:"mem_total_bytes"`
	MemUsedBytes  int64   `json:"mem_used_bytes"`
	MemPercent    float64 `json:"mem_percent"`

	DiskTotalBytes int64   `json:"disk_total_bytes"`
	DiskUsedBytes  int64   `json:"disk_used_bytes"`
	DiskAvailBytes int64   `json:"disk_avail_bytes"`
	DiskPercent    float64 `json:"disk_percent"`
}

// Snapshot collects current system resource usage.
func Snapshot() (SystemResources, error) {
	var res SystemResources

	if err := loadAvg(&res); err != nil {
		return res, fmt.Errorf("load avg: %w", err)
	}
	if err := memUsage(&res); err != nil {
		return res, fmt.Errorf("memory: %w", err)
	}
	if err := diskUsage(&res); err != nil {
		return res, fmt.Errorf("disk: %w", err)
	}

	return res, nil
}

// loadAvg parses "sysctl -n vm.loadavg" output: "{ 1.23 4.56 7.89 }"
func loadAvg(res *SystemResources) error {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return err
	}

	s := strings.TrimSpace(string(out))
	s = strings.Trim(s, "{ }")
	fields := strings.Fields(s)
	if len(fields) < 3 {
		return fmt.Errorf("unexpected loadavg format: %q", string(out))
	}

	res.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
	res.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
	res.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
	return nil
}

// memUsage uses sysctl for total memory and vm_stat for active+wired pages.
func memUsage(res *SystemResources) error {
	// Total memory
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return err
	}
	total, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return fmt.Errorf("parsing hw.memsize: %w", err)
	}
	res.MemTotalBytes = total

	// vm_stat for page counts
	out, err = exec.Command("vm_stat").Output()
	if err != nil {
		return err
	}

	var pageSize int64 = 16384 // default Apple Silicon
	var active, wired, inactive, speculative int64

	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics: (page size of ") {
			// Parse page size from header
			s := strings.TrimPrefix(line, "Mach Virtual Memory Statistics: (page size of ")
			s = strings.TrimSuffix(s, " bytes)")
			if ps, err := strconv.ParseInt(s, 10, 64); err == nil {
				pageSize = ps
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "."))

		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}

		switch key {
		case "Pages active":
			active = n
		case "Pages wired down":
			wired = n
		case "Pages inactive":
			inactive = n
		case "Pages speculative":
			speculative = n
		}
	}

	// Used = active + wired (what the system considers "in use")
	// inactive + speculative are reclaimable
	_ = inactive
	_ = speculative
	res.MemUsedBytes = (active + wired) * pageSize
	if res.MemTotalBytes > 0 {
		res.MemPercent = float64(res.MemUsedBytes) / float64(res.MemTotalBytes) * 100
	}
	return nil
}

// diskUsage uses syscall.Statfs on "/" for disk metrics.
func diskUsage(res *SystemResources) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return err
	}

	res.DiskTotalBytes = int64(stat.Blocks) * int64(stat.Bsize)
	res.DiskAvailBytes = int64(stat.Bavail) * int64(stat.Bsize)
	res.DiskUsedBytes = res.DiskTotalBytes - int64(stat.Bfree)*int64(stat.Bsize)
	if res.DiskTotalBytes > 0 {
		res.DiskPercent = float64(res.DiskUsedBytes) / float64(res.DiskTotalBytes) * 100
	}
	return nil
}
