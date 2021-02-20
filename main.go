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

    "github.com/mindprince/gonvml"
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
    powerUsage              *prometheus.GaugeVec
    avgPowerUsage           *prometheus.GaugeVec
    temperature             *prometheus.GaugeVec
    fanSpeed                *prometheus.GaugeVec
    encUsage                *prometheus.GaugeVec
    decUsage                *prometheus.GaugeVec
    GPUUtilizationRate      *prometheus.GaugeVec
    avgGPUUtilization       *prometheus.GaugeVec
    memoryUtilizationRate   *prometheus.GaugeVec
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
    }
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.numDevices.Desc()
    c.usedMemory.Describe(ch)
    c.totalMemory.Describe(ch)
    c.powerUsage.Describe(ch)
    c.avgPowerUsage.Describe(ch)
    c.temperature.Describe(ch)
    c.fanSpeed.Describe(ch)
    c.encUsage.Describe(ch)
    c.decUsage.Describe(ch)
    c.GPUUtilizationRate.Describe(ch)
    c.avgGPUUtilization.Describe(ch)
    c.memoryUtilizationRate.Describe(ch)
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
    // Only one Collect call in progress at a time.
    c.Lock()
    defer c.Unlock()

    c.usedMemory.Reset()
    c.totalMemory.Reset()
    c.powerUsage.Reset()
    c.avgPowerUsage.Reset()
    c.temperature.Reset()
    c.fanSpeed.Reset()
    c.encUsage.Reset()
    c.decUsage.Reset()
    c.GPUUtilizationRate.Reset()
    c.avgGPUUtilization.Reset()
    c.memoryUtilizationRate.Reset()

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
    }
    c.usedMemory.Collect(ch)
    c.totalMemory.Collect(ch)
    c.powerUsage.Collect(ch)
    c.avgPowerUsage.Collect(ch)
    c.temperature.Collect(ch)
    c.fanSpeed.Collect(ch)
    c.encUsage.Collect(ch)
    c.decUsage.Collect(ch)
    c.GPUUtilizationRate.Collect(ch)
    c.avgGPUUtilization.Collect(ch)
    c.memoryUtilizationRate.Collect(ch)
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

    prometheus.MustRegister(NewCollector())

    // Serve on all paths under addr
    log.Fatalf("ListenAndServe error: %v", http.ListenAndServe(*addr, promhttp.Handler()))
}
