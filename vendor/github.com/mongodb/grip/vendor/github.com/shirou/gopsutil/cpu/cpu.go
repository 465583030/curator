package cpu

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/internal/common"
)

type TimesStat struct {
	CPU       string  `json:"cpu" bson:"cpu,omitempty"`
	User      float64 `json:"user" bson:"user,omitempty"`
	System    float64 `json:"system" bson:"system,omitempty"`
	Idle      float64 `json:"idle" bson:"idle,omitempty"`
	Nice      float64 `json:"nice" bson:"nice,omitempty"`
	Iowait    float64 `json:"iowait" bson:"iowait,omitempty"`
	Irq       float64 `json:"irq" bson:"irq,omitempty"`
	Softirq   float64 `json:"softirq" bson:"softirq,omitempty"`
	Steal     float64 `json:"steal" bson:"steal,omitempty"`
	Guest     float64 `json:"guest" bson:"guest,omitempty"`
	GuestNice float64 `json:"guestNice" bson:"guestNice,omitempty"`
	Stolen    float64 `json:"stolen" bson:"stolen,omitempty"`
}

type InfoStat struct {
	CPU        int32    `json:"cpu" bson:"cpu,omitempty"`
	VendorID   string   `json:"vendorId" bson:"vendorId,omitempty"`
	Family     string   `json:"family" bson:"family,omitempty"`
	Model      string   `json:"model" bson:"model,omitempty"`
	Stepping   int32    `json:"stepping" bson:"stepping,omitempty"`
	PhysicalID string   `json:"physicalId" bson:"physicalId,omitempty"`
	CoreID     string   `json:"coreId" bson:"coreId,omitempty"`
	Cores      int32    `json:"cores" bson:"cores,omitempty"`
	ModelName  string   `json:"modelName" bson:"modelName,omitempty"`
	Mhz        float64  `json:"mhz" bson:"mhz,omitempty"`
	CacheSize  int32    `json:"cacheSize" bson:"cacheSize,omitempty"`
	Flags      []string `json:"flags" bson:"flags,omitempty"`
}

type lastPercent struct {
	sync.Mutex
	lastCPUTimes    []TimesStat
	lastPerCPUTimes []TimesStat
}

var lastCPUPercent lastPercent
var invoke common.Invoker

func init() {
	invoke = common.Invoke{}
	lastCPUPercent.Lock()
	lastCPUPercent.lastCPUTimes, _ = Times(false)
	lastCPUPercent.lastPerCPUTimes, _ = Times(true)
	lastCPUPercent.Unlock()
}

func Counts(logical bool) (int, error) {
	return runtime.NumCPU(), nil
}

func (c TimesStat) String() string {
	v := []string{
		`"cpu":"` + c.CPU + `"`,
		`"user":` + strconv.FormatFloat(c.User, 'f', 1, 64),
		`"system":` + strconv.FormatFloat(c.System, 'f', 1, 64),
		`"idle":` + strconv.FormatFloat(c.Idle, 'f', 1, 64),
		`"nice":` + strconv.FormatFloat(c.Nice, 'f', 1, 64),
		`"iowait":` + strconv.FormatFloat(c.Iowait, 'f', 1, 64),
		`"irq":` + strconv.FormatFloat(c.Irq, 'f', 1, 64),
		`"softirq":` + strconv.FormatFloat(c.Softirq, 'f', 1, 64),
		`"steal":` + strconv.FormatFloat(c.Steal, 'f', 1, 64),
		`"guest":` + strconv.FormatFloat(c.Guest, 'f', 1, 64),
		`"guestNice":` + strconv.FormatFloat(c.GuestNice, 'f', 1, 64),
		`"stolen":` + strconv.FormatFloat(c.Stolen, 'f', 1, 64),
	}

	return `{` + strings.Join(v, ",") + `}`
}

// Total returns the total number of seconds in a CPUTimesStat
func (c TimesStat) Total() float64 {
	total := c.User + c.System + c.Nice + c.Iowait + c.Irq + c.Softirq + c.Steal +
		c.Guest + c.GuestNice + c.Idle + c.Stolen
	return total
}

func (c InfoStat) String() string {
	s, _ := json.Marshal(c)
	return string(s)
}

func getAllBusy(t TimesStat) (float64, float64) {
	busy := t.User + t.System + t.Nice + t.Iowait + t.Irq +
		t.Softirq + t.Steal + t.Guest + t.GuestNice + t.Stolen
	return busy + t.Idle, busy
}

func calculateBusy(t1, t2 TimesStat) float64 {
	t1All, t1Busy := getAllBusy(t1)
	t2All, t2Busy := getAllBusy(t2)

	if t2Busy <= t1Busy {
		return 0
	}
	if t2All <= t1All {
		return 1
	}
	return (t2Busy - t1Busy) / (t2All - t1All) * 100
}

func calculateAllBusy(t1, t2 []TimesStat) ([]float64, error) {
	// Make sure the CPU measurements have the same length.
	if len(t1) != len(t2) {
		return nil, fmt.Errorf(
			"received two CPU counts: %d != %d",
			len(t1), len(t2),
		)
	}

	ret := make([]float64, len(t1))
	for i, t := range t2 {
		ret[i] = calculateBusy(t1[i], t)
	}
	return ret, nil
}

//Percent calculates the percentage of cpu used either per CPU or combined.
//If an interval of 0 is given it will compare the current cpu times against the last call.
func Percent(interval time.Duration, percpu bool) ([]float64, error) {
	if interval <= 0 {
		return percentUsedFromLastCall(percpu)
	}

	// Get CPU usage at the start of the interval.
	cpuTimes1, err := Times(percpu)
	if err != nil {
		return nil, err
	}

	time.Sleep(interval)

	// And at the end of the interval.
	cpuTimes2, err := Times(percpu)
	if err != nil {
		return nil, err
	}

	return calculateAllBusy(cpuTimes1, cpuTimes2)
}

func percentUsedFromLastCall(percpu bool) ([]float64, error) {
	cpuTimes, err := Times(percpu)
	if err != nil {
		return nil, err
	}
	lastCPUPercent.Lock()
	defer lastCPUPercent.Unlock()
	var lastTimes []TimesStat
	if percpu {
		lastTimes = lastCPUPercent.lastPerCPUTimes
		lastCPUPercent.lastPerCPUTimes = cpuTimes
	} else {
		lastTimes = lastCPUPercent.lastCPUTimes
		lastCPUPercent.lastCPUTimes = cpuTimes
	}

	if lastTimes == nil {
		return nil, fmt.Errorf("Error getting times for cpu percent. LastTimes was nil")
	}
	return calculateAllBusy(lastTimes, cpuTimes)
}
