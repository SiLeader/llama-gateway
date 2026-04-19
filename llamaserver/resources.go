package llamaserver

import (
	"bufio"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func getCpuThreads() (threads int) {
	threads = runtime.NumCPU()

	file, err := os.Open("/sys/fs/cgroup/cpu.max")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if sl := strings.Split(line, " "); len(sl) == 2 {
			allowdUs, err := strconv.ParseInt(sl[0], 10, 64)
			if err != nil {
				slog.Warn("failed to parse CPU allowed micro secs", "error", err)
				return
			}
			unitUs, err := strconv.ParseInt(sl[1], 10, 64)
			if err != nil {
				slog.Warn("failed to parse CPU unit micro secs", "error", err)
				return
			}

			threads = int(max(allowdUs/unitUs, 1))

			return
		}
	}
	return
}
