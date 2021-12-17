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
    enableFanSpeed = flag.Bool("enable-fanspeed", true, "Enable fanspeed metric")
    enablePowerLimits = flag.Bool("enable-powerlimits", true, "Enable power limit metrics")
    enableAveragePowerUsage = flag.Bool("enable-averagepowerusage", true, "Enable average power usage metric")


    labels = []string{"minor_number", "uuid", "name"}

    averageDuration = time.Duration(15) * time.Second
)

/* 
                                                                                
*/

type Collector struct {
    sync.Mutex
    numDevices                      prometheus.Gauge
    usedMemory                      *prometheus.GaugeVec
    totalMemory                     *prometheus.GaugeVec
    usedBar1Memory                  *prometheus.GaugeVec
    totalBar1Memory                 *prometheus.GaugeVec
    powerUsage                      *prometheus.GaugeVec
    avgPowerUsage                   *prometheus.GaugeVec
    energyConsumption               *prometheus.GaugeVec
    temperature                     *prometheus.GaugeVec
    temperatureThresholdShutDown    *prometheus.GaugeVec
    temperatureThresholdSlowDown    *prometheus.GaugeVec
    throttlingReason                *prometheus.GaugeVec
    fanSpeed                        *prometheus.GaugeVec
    encUsage                        *prometheus.GaugeVec
    decUsage                        *prometheus.GaugeVec
    GPUUtilizationRate              *prometheus.GaugeVec
    avgGPUUtilization               *prometheus.GaugeVec
    memoryUtilizationRate           *prometheus.GaugeVec
    computeMode                     *prometheus.GaugeVec
    performanceState                *prometheus.GaugeVec
    grClockCurrent                  *prometheus.GaugeVec
    grClockMax                      *prometheus.GaugeVec
    SMClockCurrent                  *prometheus.GaugeVec
    SMClockMax                      *prometheus.GaugeVec
    memClockCurrent                 *prometheus.GaugeVec
    memClockMax                     *prometheus.GaugeVec
    videoClockCurrent               *prometheus.GaugeVec
    videoClockMax                   *prometheus.GaugeVec
    powerLimitConstraintsMin        *prometheus.GaugeVec
    powerLimitConstraintsMax        *prometheus.GaugeVec
    powerLimitManagement            *prometheus.GaugeVec
    powerLimitEnforced              *prometheus.GaugeVec
    powerManagementDefaultLimit     *prometheus.GaugeVec
    pciTxThroughput                 *prometheus.GaugeVec
    pciRxThroughput                 *prometheus.GaugeVec
    pciLinkGenerationCurrent        *prometheus.GaugeVec
    pciLinkGenerationMax            *prometheus.GaugeVec
    pciLinkWidthCurrent             *prometheus.GaugeVec
    pciLinkWidthMax                 *prometheus.GaugeVec
    videoEncoderCapacityH264        *prometheus.GaugeVec
    videoEncoderCapacityHEVC        *prometheus.GaugeVec
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
                Name:      "power_usage_watts",
                Help:      "Power usage of the GPU device in watts",
            },
            labels,
        ),
        avgPowerUsage: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "avg_power_usage_watts",
                Help:      "power usage for this GPU and its associated circuitry in watts averaged over the samples collected in the last `since` duration.",
            },
            labels,
        ),
        energyConsumption: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "energy_consumption_joules",
                Help:      "total energy consumption for this GPU in joules (J) since the driver was last reloaded.",
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
        temperatureThresholdShutDown: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "temperature_threshold_shutdown_celcius",
                Help:      "Temperature slowdown threshold celsius",
            },
            labels,
        ),
        temperatureThresholdSlowDown: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "temperature_threshold_slowdown_celcius",
                Help:      "Temperature slowdown threshold in celsius",
            },
            labels,
        ),
        throttlingReason: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "throttling_reason",
                Help:      "Most serious reason for the GPU being throttling",
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
                Name:      "avg_gpu_utilization_percent",
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
                Name:      "clock_gr_current_mhz",
                Help:      "grClockCurrent returns the current speed of the graphics clock",
            },
            labels,
        ),
        grClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_gr_max_mhz",
                Help:      "grClockMax returns the maximum speed of the graphics clock",
            },
            labels,
        ),
        SMClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_sm_current_mhz",
                Help:      "smClockCurrent returns the current speed of the streaming multiprocessors (SM) clock",
            },
            labels,
        ),
        SMClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_sm_max_mhz",
                Help:      "smClockMax returns the maximum speed of the streaming multiprocessors (SM) clock",
            },
            labels,
        ),
        memClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_mem_current_mhz",
                Help:      "memClockCurrent returns the current speed of the memory clock",
            },
            labels,
        ),
        memClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_mem_max_mhz",
                Help:      "memClockMax returns the maximum speed of the memory clock",
            },
            labels,
        ),
        videoClockCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_video_current_mhz",
                Help:      "videolockCurrent returns the current speed of the video encoder/decoder clock",
            },
            labels,
        ),
        videoClockMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "clock_video_max_mhz",
                Help:      "memClockMax returns the maximum speed of the video encoder/decoder clock",
            },
            labels,
        ),
        powerLimitConstraintsMin: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_min_watts",
                Help:      "PowerLimitConstraints retrieves information about possible values of power management limits on this device (min)",
            },
            labels,
        ),
        powerLimitConstraintsMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_max_watts",
                Help:      "PowerLimitConstraints retrieves information about possible values of power management limits on this device (max)",
            },
            labels,
        ),
        powerLimitManagement: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_management_watts",
                Help:      "The power limit defines the upper boundary for the card's power draw. If the card's total power draw reaches this limit the power management algorithm kicks in.",
            },
            labels,
        ),
        powerLimitEnforced: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_limit_enforced_watts",
                Help:      "Effective power limit that the driver enforces after taking into account all limiters.  Note: This can be different from the management limit if other limits are set elsewhere This includes the out of band power limit interface",
            },
            labels,
        ),
        powerManagementDefaultLimit: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "power_management_default_limit_watts",
                Help:      "PowerManagementDefaultLimit returns the power limit for this GPU and its associated circuitry in watts",
            },
            labels,
        ),
        pciTxThroughput: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_throughput_tx_kilobytes_per_second",
                Help:      "tx throughput in KB/s",
            },
            labels,
        ),
        pciRxThroughput: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_throughput_rx_kilobytes_per_second",
                Help:      "rx throughput in KB/s",
            },
            labels,
        ),
        pciLinkGenerationCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_generation_current",
                Help:      "current PCIe link generation",
            },
            labels,
        ),
        pciLinkGenerationMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_generation_max",
                Help:      "Max PCIe link generation",
            },
            labels,
        ),
        pciLinkWidthCurrent: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_width_current",
                Help:      "current PCIe link width",
            },
            labels,
        ),
        pciLinkWidthMax: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "pci_width_max",
                Help:      "Max PCIe link width",
            },
            labels,
        ),
        videoEncoderCapacityH264: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "video_encoder_capacity_h264",
                Help:      "Percentage of maximum encoder capacity (H264)",
            },
            labels,
        ),
        videoEncoderCapacityHEVC: prometheus.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: namespace,
                Name:      "video_encoder_capacity_hevc",
                Help:      "Percentage of maximum encoder capacity (HEVC)",
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
    c.energyConsumption.Describe(ch)
    c.temperature.Describe(ch)
    c.temperatureThresholdShutDown.Describe(ch)
    c.temperatureThresholdSlowDown.Describe(ch)
    c.throttlingReason.Describe(ch)
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
    c.powerLimitManagement.Describe(ch)
    c.powerLimitEnforced.Describe(ch)
    c.powerManagementDefaultLimit.Describe(ch)
    c.pciTxThroughput.Describe(ch)
    c.pciRxThroughput.Describe(ch)
    c.pciLinkGenerationCurrent.Describe(ch)
    c.pciLinkGenerationMax.Describe(ch)
    c.pciLinkWidthCurrent.Describe(ch)
    c.pciLinkWidthMax.Describe(ch)
    c.videoEncoderCapacityH264.Describe(ch)
    c.videoEncoderCapacityHEVC.Describe(ch)
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
    c.energyConsumption.Reset()
    c.temperature.Reset()
    c.temperatureThresholdShutDown.Reset()
    c.temperatureThresholdSlowDown.Reset()
    c.throttlingReason.Reset()
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
    c.powerLimitManagement.Reset()
    c.powerLimitEnforced.Reset()
    c.powerManagementDefaultLimit.Reset()
    c.pciTxThroughput.Reset()
    c.pciRxThroughput.Reset()
    c.pciLinkGenerationCurrent.Reset()
    c.pciLinkGenerationMax.Reset()
    c.pciLinkWidthCurrent.Reset()
    c.pciLinkWidthMax.Reset()
    c.videoEncoderCapacityH264.Reset()
    c.videoEncoderCapacityHEVC.Reset()

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
            c.powerUsage.WithLabelValues(minor, uuid, name).Set(float64(powerUsage/1000))
        }

        if *enableAveragePowerUsage {
            avgPowerUsage, err := dev.AveragePowerUsage(averageDuration)
            if err != nil {
                log.Printf("AveragePowerUsage() error: %v", err)
            } else {
                c.avgPowerUsage.WithLabelValues(minor, uuid, name).Set(float64(avgPowerUsage/1000))
            }
        }

        energyConsumption, err := dev.TotalEnergyConsumption()
        if err != nil {
            log.Printf("TotalEnergyConsumption() error: %v", err)
        } else {
            c.energyConsumption.WithLabelValues(minor, uuid, name).Set(float64(energyConsumption/1000))
        }

        if *enablePowerLimits {
            powerLimitConstraintsMin, powerLimitConstraintsMax, err := dev.PowerLimitConstraints()
            if err != nil {
                log.Printf("PowerLimitConstraints() error: %v", err)
            } else {
                c.powerLimitConstraintsMin.WithLabelValues(minor, uuid, name).Set(float64(powerLimitConstraintsMin/1000))
                c.powerLimitConstraintsMax.WithLabelValues(minor, uuid, name).Set(float64(powerLimitConstraintsMax/1000))
            }

            powerLimitManagement, powerLimitEnforced, err := dev.PowerLimits()
            if err != nil {
                log.Printf("PowerLimits() error: %v", err)
            } else {
                c.powerLimitManagement.WithLabelValues(minor, uuid, name).Set(float64(powerLimitManagement/1000))
                c.powerLimitEnforced.WithLabelValues(minor, uuid, name).Set(float64(powerLimitEnforced/1000))
            }
            powerManagementDefaultLimit, err := dev.PowerManagementDefaultLimit()
            if err != nil {
                log.Printf("PowerManagementDefaultLimit() error: %v", err)
            } else {
                c.powerManagementDefaultLimit.WithLabelValues(minor, uuid, name).Set(float64(powerManagementDefaultLimit/1000))
            }
        }

        temperature, err := dev.Temperature()
        if err != nil {
            log.Printf("Temperature() error: %v", err)
        } else {
            c.temperature.WithLabelValues(minor, uuid, name).Set(float64(temperature))
        }
        temperature_threshold_shutdown, temperature_threshold_slowdown, err := dev.TemperatureThresholds()
        if err != nil {
            log.Printf("TemperatureThresholds() error: %v", err)
        } else {
            c.temperatureThresholdShutDown.WithLabelValues(minor, uuid, name).Set(float64(temperature_threshold_shutdown))
            c.temperatureThresholdSlowDown.WithLabelValues(minor, uuid, name).Set(float64(temperature_threshold_slowdown))
        }

        throttling_reason, err := dev.MostSeriousClocksThrottleReason()
        if err != nil {
            log.Printf("throttlingReason() error: %v", err)
        } else {
            c.throttlingReason.WithLabelValues(minor, uuid, name).Set(float64(throttling_reason))
        }

        if *enableFanSpeed {
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


        pciTxThroughput, err := dev.PcieTxThroughput()
        if err == nil {
            c.pciTxThroughput.WithLabelValues(minor, uuid, name).Set(float64(pciTxThroughput))
        }
        PciRxThroughput, err := dev.PcieRxThroughput()
        if err == nil {
            c.pciRxThroughput.WithLabelValues(minor, uuid, name).Set(float64(PciRxThroughput))
        }
        pciLinkGenerationCurrent, err := dev.PcieGeneration()
        if err == nil {
            c.pciLinkGenerationCurrent.WithLabelValues(minor, uuid, name).Set(float64(pciLinkGenerationCurrent))
        }
        pciLinkGenerationMax, err := dev.PcieMaxGeneration()
        if err == nil {
            c.pciLinkGenerationMax.WithLabelValues(minor, uuid, name).Set(float64(pciLinkGenerationMax))
        }
        pciLinkWidthCurrent, err := dev.PcieWidth()
        if err == nil {
            c.pciLinkWidthCurrent.WithLabelValues(minor, uuid, name).Set(float64(pciLinkWidthCurrent))
        }
        pciLinkWidthMax, err := dev.PcieMaxWidth()
        if err == nil {
            c.pciLinkWidthMax.WithLabelValues(minor, uuid, name).Set(float64(pciLinkWidthMax))
        }
        caph264, caphevc, err := dev.EncoderCapacity()
        if err == nil {
            c.videoEncoderCapacityH264.WithLabelValues(minor, uuid, name).Set(float64(caph264))
            c.videoEncoderCapacityHEVC.WithLabelValues(minor, uuid, name).Set(float64(caphevc))
        }

    }
    c.usedMemory.Collect(ch)
    c.totalMemory.Collect(ch)
    c.usedBar1Memory.Collect(ch)
    c.totalBar1Memory.Collect(ch)
    c.powerUsage.Collect(ch)
    c.avgPowerUsage.Collect(ch)
    c.energyConsumption.Collect(ch)
    c.temperature.Collect(ch)
    c.temperatureThresholdShutDown.Collect(ch)
    c.temperatureThresholdSlowDown.Collect(ch)
    c.throttlingReason.Collect(ch)
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
    c.powerLimitManagement.Collect(ch)
    c.powerLimitEnforced.Collect(ch)
    c.powerManagementDefaultLimit.Collect(ch)
    c.pciTxThroughput.Collect(ch)
    c.pciRxThroughput.Collect(ch)
    c.pciLinkGenerationCurrent.Collect(ch)
    c.pciLinkGenerationMax.Collect(ch)
    c.pciLinkWidthCurrent.Collect(ch)
    c.pciLinkWidthMax.Collect(ch)
    c.videoEncoderCapacityH264.Collect(ch)
    c.videoEncoderCapacityHEVC.Collect(ch)
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
