// Copied from https://github.com/mindprince/nvidia_gpu_prometheus_exporter
// https://github.com/mindprince/nvidia_gpu_prometheus_exporter/blob/master/LICENSE

package main

import (
    "flag"
    "log"
    "net/http"
    "strconv"
    "sync"
    "time"

    "github.com/cfsmp3/gonvml"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
    namespace = "nvidia_gpu"
)

var (
    addr = flag.String("web.listen-address", ":9445", "Address to listen on for web interface and telemetry.")
    disableFanSpeed = flag.Bool("disable-fanspeed", false, "Disable fanspeed metric")

    labels = []string{"minor_number", "uuid", "name"}

    averageDuration = time.Duration(15) * time.Second
)

/* 
                                                                                
*/

type Collector struct {
    sync.Mutex
    numDevices              prometheus.Gauge
    usedMemory              *prometheus.GaugeVec
    totalMemory             *prometheus.GaugeVec
    usedBar1Memory          *prometheus.GaugeVec
    totalBar1Memory         *prometheus.GaugeVec
    powerUsage              *prometheus.GaugeVec
    avgPowerUsage           *prometheus.GaugeVec
    temperature             *prometheus.GaugeVec
    fanSpeed                *prometheus.GaugeVec
    encUsage                *prometheus.GaugeVec
    decUsage                *prometheus.GaugeVec
    GPUUtilizationRate      *prometheus.GaugeVec
    avgGPUUtilization       *prometheus.GaugeVec
    memoryUtilizationRate   *prometheus.GaugeVec
    computeMode             *prometheus.GaugeVec
    performanceState        *prometheus.GaugeVec
    grClockCurrent          *prometheus.GaugeVec
    grClockMax              *prometheus.GaugeVec
    SMClockCurrent          *prometheus.GaugeVec
    SMClockMax              *prometheus.GaugeVec
    memClockCurrent         *prometheus.GaugeVec
    memClockMax             *prometheus.GaugeVec
    videoClockCurrent       *prometheus.GaugeVec
    videoClockMax           *prometheus.GaugeVec
    powerLimitConstraintsMin *prometheus.GaugeVec
    powerLimitConstraintsMax *prometheus.GaugeVec
    powerState              *prometheus.GaugeVec
}

func NewCollector() *Collector {
    return &Collector{
        numDevices: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "num_devices",
                Help:      "Number of GPU devices",
            },
        ),
        usedMemory: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "memory_used_bytes",
                Help:      "Memory used by the GPU device in bytes",
            },
            labels,
        ),
        totalMemory: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "memory_total_bytes",
                Help:      "Total memory of the GPU device in bytes",
            },
            labels,
        ),
        usedBar1Memory: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "bar1_memory_used_bytes",
                Help:      "BAR1 Memory used by the GPU device in bytes",
            },
            labels,
        ),
        totalBar1Memory: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "bar1_memory_total_bytes",
                Help:      "Total BAR1 memory of the GPU device in bytes",
            },
            labels,
        ),
        powerUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_usage_milliwatts",
                Help:      "Power usage of the GPU device in milliwatts",
            },
            labels,
        ),
        avgPowerUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "avg_power_usage_milliwatts",
                Help:      "power usage for this GPU and its associated circuitry in milliwatts averaged over the samples collected in the last `since` duration.",
            },
            labels,
        ),
        temperature: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "temperature_celsius",
                Help:      "Temperature of the GPU device in celsius",
            },
            labels,
        ),
        fanSpeed: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "fanspeed_percent",
                Help:      "Fanspeed of the GPU device as a percent of its maximum",
            },
            labels,
        ),
        encUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "encoder_utilization_percent",
                Help:      "EncoderUtilization returns the percent of time over the last sample period during which the GPU video encoder was being used.",
            },
            labels,
        ),
        decUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "decoder_utilization_percent",
                Help:      "DecoderUtilization returns the percent of time over the last sample period during which the GPU video decoder was being used.",
            },
            labels,
        ),
        GPUUtilizationRate: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "gpu_utilization_rate",
                Help:      "GPUUtilizationRate returns the percent of time one of more kernels were executing on the GPU) averaged over the samples collected in the last `since` duration.",
            },
            labels,
        ),
        avgGPUUtilization: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "avg_gpu_utilization",
                Help:      "avgGPUUtilization returns the percent of time over the past sample period during which one or more kernels were executing on the GPU.",
            },
            labels,
        ),
        memoryUtilizationRate: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "memory_utilization_rate",
                Help:      "memoryUtilizationRate returns the percent of time over the past sample period during which global (device) memory was being read or written.",
            },
            labels,
        ),
        computeMode: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "compute_mode",
                Help:      "computeMode returns the compute mode of the device (prohibited, exclusive to thread, exclusive to process...).",
            },
            labels,
        ),
        performanceState: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "performance_state",
                Help:      "PerformanceState returns the current performance state for the device (P0 maximum, P15 minimum, P32 unknown)",
            },
            labels,
        ),
        grClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_gr_current",
                Help:      "grClockCurrent returns the current speed of the graphics clock ",
            },
            labels,
        ),
        grClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_gr_max",
                Help:      "grClockMax returns the maximum speed of the graphics clock ",
            },
            labels,
        ),
        SMClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_sm_current",
                Help:      "smClockCurrent returns the current speed of the streaming multiprocessors (SM) clock ",
            },
            labels,
        ),
        SMClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_sm_max",
                Help:      "smClockMax returns the maximum speed of the streaming multiprocessors (SM) clock ",
            },
            labels,
        ),
        memClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_mem_current",
                Help:      "memClockCurrent returns the current speed of the memory clock ",
            },
            labels,
        ),
        memClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_mem_max",
                Help:      "memClockMax returns the maximum speed of the memory clock ",
            },
            labels,
        ),
        videoClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_video_current",
                Help:      "videolockCurrent returns the current speed of the video encoder/decoder clock ",
            },
            labels,
        ),
        videoClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_video_max",
                Help:      "memClockMax returns the maximum speed of the video encoder/decoder clock ",
            },
            labels,
        ),
        powerLimitConstraintsMin: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_min",
                Help:      "PowerLimitConstraints retrieves information about possible values of power management limits on this device (min)",
            },
            labels,
        ),
        powerLimitConstraintsMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_max",
                Help:      "PowerLimitConstraints retrieves information about possible values of power management limits on this device (max)",
            },
            labels,
        ),
        powerState: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_state",
                Help:      "PowerState returns the current PState of the GPU Device",
            },
            labels,
        ),
    }
}


func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.numDevices.Desc()
    c.usedMemory.Describe(ch)
    c.totalMemory.Describe(ch)
    c.usedBar1Memory.Describe(ch)
    c.totalBar1Memory.Describe(ch)
    c.powerUsage.Describe(ch)
    c.avgPowerUsage.Describe(ch)
    c.temperature.Describe(ch)
    c.fanSpeed.Describe(ch)
    c.encUsage.Describe(ch)
    c.decUsage.Describe(ch)
    c.GPUUtilizationRate.Describe(ch)
    c.avgGPUUtilization.Describe(ch)
    c.memoryUtilizationRate.Describe(ch)
    c.computeMode.Describe(ch)
    c.performanceState.Describe(ch)
    c.grClockCurrent.Describe(ch)
    c.grClockMax.Describe(ch)
    c.SMClockCurrent.Describe(ch)
    c.SMClockMax.Describe(ch)
    c.memClockCurrent.Describe(ch)
    c.memClockMax.Describe(ch)
    c.videoClockCurrent.Describe(ch)
    c.videoClockMax.Describe(ch)
    c.powerLimitConstraintsMin.Describe(ch)
    c.powerLimitConstraintsMax.Describe(ch)
    c.powerState.Describe(ch)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
    // Only one Collect call in progress at a time.
    c.Lock()
    defer c.Unlock()

    c.usedMemory.Reset()
    c.totalMemory.Reset()
    c.usedBar1Memory.Reset()
    c.totalBar1Memory.Reset()
    c.powerUsage.Reset()
    c.avgPowerUsage.Reset()
    c.temperature.Reset()
    c.fanSpeed.Reset()
    c.encUsage.Reset()
    c.decUsage.Reset()
    c.GPUUtilizationRate.Reset()
    c.avgGPUUtilization.Reset()
    c.memoryUtilizationRate.Reset()
    c.computeMode.Reset()
    c.performanceState.Reset()
    c.grClockCurrent.Reset()
    c.grClockMax.Reset()
    c.SMClockCurrent.Reset()
    c.SMClockMax.Reset()
    c.memClockCurrent.Reset()
    c.memClockMax.Reset()
    c.videoClockCurrent.Reset()
    c.videoClockMax.Reset()
    c.powerLimitConstraintsMin.Reset()
    c.powerLimitConstraintsMax.Reset()
    c.powerState.Reset()

    numDevices, err := gonvml.DeviceCount()
    if err != nil {
        log.Printf("DeviceCount() error: %v", err)
        return
    } else {
        c.numDevices.Set(float64(numDevices))
        ch <- c.numDevices
    }

    for i := 0; i < int(numDevices); i++ {
        dev, err := gonvml.DeviceHandleByIndex(uint(i))
        if err != nil {
            log.Printf("DeviceHandleByIndex(%d) error: %v", i, err)
            continue
        }

        minorNumber, err := dev.MinorNumber()
        if err != nil {
            log.Printf("MinorNumber() error: %v", err)
            continue
        }
        minor := strconv.Itoa(int(minorNumber))

        uuid, err := dev.UUID()
        if err != nil {
            log.Printf("UUID() error: %v", err)
            continue
        }

        name, err := dev.Name()
        if err != nil {
            log.Printf("Name() error: %v", err)
            continue
        }

        totalMemory, usedMemory, err := dev.MemoryInfo()
        if err != nil {
            log.Printf("MemoryInfo() error: %v", err)
        } else {
            c.usedMemory.WithLabelValues(minor, uuid, name).Set(float64(usedMemory))
            c.totalMemory.WithLabelValues(minor, uuid, name).Set(float64(totalMemory))
        }

        totalBar1Memory, usedBar1Memory, err := dev.Bar1MemoryInfo()
        if err != nil {
            log.Printf("Bar1MemoryInfo() error: %v", err)
        } else {
            c.usedBar1Memory.WithLabelValues(minor, uuid, name).Set(float64(usedBar1Memory))
            c.totalBar1Memory.WithLabelValues(minor, uuid, name).Set(float64(totalBar1Memory))
        }

        utilizationGPU, utilizationMemory, err := dev.UtilizationRates()
        if err == nil {
            c.GPUUtilizationRate.WithLabelValues(minor, uuid, name).Set(float64(utilizationGPU))
            c.memoryUtilizationRate.WithLabelValues(minor, uuid, name).Set(float64(utilizationMemory))
        }

        powerUsage, err := dev.PowerUsage()
        if err != nil {
            log.Printf("PowerUsage() error: %v", err)
        } else {
            c.powerUsage.WithLabelValues(minor, uuid, name).Set(float64(powerUsage))
        }

        avgPowerUsage, err := dev.AveragePowerUsage(averageDuration)
        if err != nil {
            log.Printf("AveragePowerUsage() error: %v", err)
        } else {
            c.avgPowerUsage.WithLabelValues(minor, uuid, name).Set(float64(avgPowerUsage))
        }

        powerLimitConstraintsMin, powerLimitConstraintsMax, err := dev.PowerLimitConstraints()
        if err != nil {
            log.Printf("PowerLimitConstraints() error: %v", err)
        } else {
            c.powerLimitConstraintsMin.WithLabelValues(minor, uuid, name).Set(float64(powerLimitConstraintsMin))
            c.powerLimitConstraintsMax.WithLabelValues(minor, uuid, name).Set(float64(powerLimitConstraintsMax))
        }

        powerState, err := dev.PowerState()
        if err != nil {
            log.Printf("PowerState() error: %v", err)
        } else {
            c.powerState.WithLabelValues(minor, uuid, name).Set(float64(powerState))
        }
        temperature, err := dev.Temperature()
        if err != nil {
            log.Printf("Temperature() error: %v", err)
        } else {
            c.temperature.WithLabelValues(minor, uuid, name).Set(float64(temperature))
        }

        if !*disableFanSpeed {
            fanSpeed, err := dev.FanSpeed()
            if err != nil {
                log.Printf("FanSpeed() error: %v", err)
            } else {
                c.fanSpeed.WithLabelValues(minor, uuid, name).Set(float64(fanSpeed))
            }
        }
        encUsage, _, err := dev.EncoderUtilization()
        if err != nil {
            log.Printf("EncoderUtilization() error: %v", err)
        } else {
            c.encUsage.WithLabelValues(minor, uuid, name).Set(float64(encUsage))
        }
        decUsage, _, err := dev.DecoderUtilization()
        if err != nil {
            log.Printf("DecoderUtilization() error: %v", err)
        } else {
            c.decUsage.WithLabelValues(minor, uuid, name).Set(float64(decUsage))
        }

        utilizationGPUAverage, err := dev.AverageGPUUtilization(averageDuration)
        if err == nil {
            c.avgGPUUtilization.WithLabelValues(minor, uuid, name).Set(float64(utilizationGPUAverage))
        }

        computeMode, err := dev.ComputeMode()
        if err == nil {
            c.computeMode.WithLabelValues(minor, uuid, name).Set(float64(computeMode))
        }

        performanceState, err := dev.PerformanceState()
        if err == nil {
            c.performanceState.WithLabelValues(minor, uuid, name).Set(float64(performanceState))
        }

        grClockCurrent, err := dev.GrClock()
        if err == nil {
            c.grClockCurrent.WithLabelValues(minor, uuid, name).Set(float64(grClockCurrent))
        }
        grClockMax, err := dev.GrMaxClock()
        if err == nil {
            c.grClockMax.WithLabelValues(minor, uuid, name).Set(float64(grClockMax))
        }
        SMClockCurrent, err := dev.SMClock()
        if err == nil {
            c.SMClockCurrent.WithLabelValues(minor, uuid, name).Set(float64(SMClockCurrent))
        }
        SMClockMax, err := dev.SMMaxClock()
        if err == nil {
            c.SMClockMax.WithLabelValues(minor, uuid, name).Set(float64(SMClockMax))
        }
        MemClockCurrent, err := dev.MemClock()
        if err == nil {
            c.memClockCurrent.WithLabelValues(minor, uuid, name).Set(float64(MemClockCurrent))
        }
        MemClockMax, err := dev.MemMaxClock()
        if err == nil {
            c.memClockMax.WithLabelValues(minor, uuid, name).Set(float64(MemClockMax))
        }
        videoClockCurrent, err := dev.VideoClock()
        if err == nil {
            c.videoClockCurrent.WithLabelValues(minor, uuid, name).Set(float64(videoClockCurrent))
        }
        videoClockMax, err := dev.VideoMaxClock()
        if err == nil {
            c.videoClockMax.WithLabelValues(minor, uuid, name).Set(float64(videoClockMax))
        }
    }
    c.usedMemory.Collect(ch)
    c.totalMemory.Collect(ch)
    c.usedBar1Memory.Collect(ch)
    c.totalBar1Memory.Collect(ch)
    c.powerUsage.Collect(ch)
    c.avgPowerUsage.Collect(ch)
    c.temperature.Collect(ch)
    c.fanSpeed.Collect(ch)
    c.encUsage.Collect(ch)
    c.decUsage.Collect(ch)
    c.GPUUtilizationRate.Collect(ch)
    c.avgGPUUtilization.Collect(ch)
    c.memoryUtilizationRate.Collect(ch)
    c.computeMode.Collect(ch)
    c.performanceState.Collect(ch)
    c.grClockCurrent.Collect(ch)
    c.grClockMax.Collect(ch)
    c.SMClockCurrent.Collect(ch)
    c.SMClockMax.Collect(ch)
    c.memClockCurrent.Collect(ch)
    c.memClockMax.Collect(ch)
    c.videoClockCurrent.Collect(ch)
    c.videoClockMax.Collect(ch)
    c.powerLimitConstraintsMin.Collect(ch)
    c.powerLimitConstraintsMax.Collect(ch)
    c.powerState.Collect(ch)
}

func main() {
    flag.Parse()

    if err := gonvml.Initialize(); err != nil {
        log.Fatalf("Couldn't initialize gonvml: %v. Make sure NVML is in the shared library search path.", err)
    }
    defer gonvml.Shutdown()

    if driverVersion, err := gonvml.SystemDriverVersion(); err != nil {
        log.Printf("SystemDriverVersion() error: %v", err)
    } else {
        log.Printf("SystemDriverVersion(): %v", driverVersion)
    }

    if NVMLVersion, err := gonvml.SystemNVMLVersion(); err != nil {
        log.Printf("SystemNVMLVersion() error: %v", err)
    } else {
        log.Printf("SystemNVMLVersion(): %v", NVMLVersion)
    }

    prometheus.MustRegister(NewCollector())

    // Serve on all paths under addr
    log.Fatalf("ListenAndServe error: %v", http.ListenAndServe(*addr, promhttp.Handler()))
}
