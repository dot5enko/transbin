package utils

import (
	"fmt"
	"runtime"
)

var res, firstRun runtime.MemStats
var reports int

const allocsPerRun int = 3
const statFormat string = "%4d %8d %25s\n"

var memoryUsed = uint64(0)

//var prevMallocs, curMallocs,mallocs uint64

var Reporting = false

func ReportAllocs(label string) {
	if !Reporting {
		return
	}

	runtime.ReadMemStats(&res)

	if reports == 0 {
		firstRun = res
	}

	fmt.Printf(statFormat, res.Mallocs-uint64(allocsPerRun*reports)-firstRun.Mallocs, res.Alloc-firstRun.Alloc-memoryUsed, label)

	reports++
	memoryUsed += uint64(27)

}
