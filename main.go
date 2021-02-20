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
)

var (
	averageDuration = 10 * time.Second
)

type Metrics struct {
	Version string
	Devices []*Device
}

type MaybeMetric struct {
	value float64;
    isPresent bool;
}

func setMetric (value float64) MaybeMetric {
    return MaybeMetric {value: value, isPresent: true}
}

func unsetMetric (value float64) MaybeMetric {
    return MaybeMetric {value: 0, isPresent: false}
}

type Device struct {
	Index                 string
	MinorNumber           string
	Name                  string
	UUID                  string
	Temperature           MaybeMetric
	PowerUsage            MaybeMetric
	PowerUsageAverage     MaybeMetric
	FanSpeed              MaybeMetric
	MemoryTotal           MaybeMetric
	MemoryUsed            MaybeMetric
	UtilizationMemory     MaybeMetric
	UtilizationGPU        MaybeMetric
	UtilizationGPUAverage MaybeMetric
    DutyCycle             MaybeMetric
    AvgDuty               MaybeMetric
    EncUsage              MaybeMetric
    DecUsage              MaybeMetric
}


func collectMetrics() (*Metrics, error) {
	version, err := gonvml.SystemDriverVersion()
	if err != nil {
		return nil, err
	}

	metrics := &Metrics{
		Version: version,
	}

	numDevices, err := gonvml.DeviceCount()
	if err != nil {
		return nil, err
	}

	for index := 0; index < int(numDevices); index++ {
		device, err := gonvml.DeviceHandleByIndex(uint(index))
		if err != nil {
			continue
		}

		uuid, err := device.UUID()
		if err != nil {
			continue
		}

		name, err := device.Name()
		if err != nil {
			continue
		}

		minorNumber, err := device.MinorNumber()
		if err != nil {
			continue
		}

	    var MaybeTemperature            MaybeMetric
	    var MaybePowerUsage             MaybeMetric
	    var MaybePowerUsageAverage      MaybeMetric
	    var MaybeFanSpeed               MaybeMetric
	    var MaybeMemoryTotal            MaybeMetric
	    var MaybeMemoryUsed             MaybeMetric
	    var MaybeUtilizationMemory      MaybeMetric
	    var MaybeUtilizationGPU         MaybeMetric
	    var MaybeUtilizationGPUAverage  MaybeMetric
        var MaybeDutyCycle              MaybeMetric
        var MaybeAvgDuty                MaybeMetric
        var MaybeEncUsage               MaybeMetric
        var MaybeDecUsage               MaybeMetric

		temperature, err := device.Temperature()
		if err == nil {
            MaybeTemperature = setMetric(float64 (temperature))
		}

		powerUsage, err := device.PowerUsage()
		if err == nil {
            MaybePowerUsage = setMetric (float64 (powerUsage))
		}

		powerUsageAverage, err := device.AveragePowerUsage(averageDuration)
		if err == nil {
            MaybePowerUsageAverage = setMetric (float64 (powerUsageAverage))
		}

        fanSpeed, err := device.FanSpeed()
	    if err == nil {
                MaybeFanSpeed = setMetric (float64 (fanSpeed))
        }

		memoryTotal, memoryUsed, err := device.MemoryInfo()
		if err == nil {
            MaybeMemoryTotal = setMetric (float64 (memoryTotal))
            MaybeMemoryUsed = setMetric (float64 (memoryUsed))
		}

		utilizationGPU, utilizationMemory, err := device.UtilizationRates()
		if err == nil {
            MaybeUtilizationGPU = setMetric (float64 (utilizationGPU))
            MaybeUtilizationMemory = setMetric (float64 (utilizationMemory))
		}

		utilizationGPUAverage, err := device.AverageGPUUtilization(averageDuration)
		if err == nil {
            MaybeUtilizationGPUAverage = setMetric (float64 (utilizationGPUAverage))
		}

		encUsage, _, err := device.EncoderUtilization()
		if err == nil {
            MaybeEncUsage = setMetric (float64 (encUsage))
		}

		decUsage, _, err := device.DecoderUtilization()
		if err == nil {
            MaybeDecUsage = setMetric (float64 (decUsage))
		}

		dutyCycle, _, err := device.UtilizationRates()
		if err == nil {
            MaybeDutyCycle = setMetric (float64 (dutyCycle))
		}

        avgDuty, err := device.AverageGPUUtilization(averageDuration)
		if err == nil {
            MaybeAvgDuty = setMetric (float64 (avgDuty))
		}

		metrics.Devices = append(metrics.Devices,
			&Device{
				Index:                  strconv.Itoa(index),
				MinorNumber:            strconv.Itoa(int(minorNumber)),
				Name:                   name,
				UUID:                   uuid,
				Temperature:            MaybeTemperature,
				PowerUsage:             MaybePowerUsage,
				PowerUsageAverage:      MaybePowerUsageAverage,
				FanSpeed:               MaybeFanSpeed,
				MemoryTotal:            MaybeMemoryTotal,
				MemoryUsed:             MaybeMemoryUsed,
				UtilizationMemory:      MaybeUtilizationMemory,
				UtilizationGPU:         MaybeUtilizationGPU,
				UtilizationGPUAverage:  MaybeUtilizationGPUAverage,
                EncUsage:               MaybeEncUsage,
                DecUsage:               MaybeDecUsage,
                DutyCycle:              MaybeDutyCycle,
                AvgDuty:                MaybeAvgDuty,
			})
	}

	return metrics, nil
}

type Collector struct {
	sync.Mutex
    up                      prometheus.Gauge
	info                    *prometheus.GaugeVec
	deviceCount             prometheus.Gauge
	temperatures            *prometheus.GaugeVec
	deviceInfo              *prometheus.GaugeVec
	powerUsage              *prometheus.GaugeVec
	powerUsageAverage       *prometheus.GaugeVec
	fanSpeed                *prometheus.GaugeVec
	memoryTotal             *prometheus.GaugeVec
	memoryUsed              *prometheus.GaugeVec
	utilizationMemory       *prometheus.GaugeVec
	utilizationGPU          *prometheus.GaugeVec
	utilizationGPUAverage   *prometheus.GaugeVec
    encUsage                *prometheus.GaugeVec
    decUsage                *prometheus.GaugeVec
    dutyCycle               *prometheus.GaugeVec
    avgDuty                 *prometheus.GaugeVec
}

func NewCollector() *Collector {
	return &Collector{
		up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "up",
				Help:      "NVML Metric Collection Operational",
			},
		),
		info: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "driver_info",
				Help:      "NVML Info",
			},
			[]string{"version"},
		),
		deviceCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "device_count",
				Help:      "Count of found nvidia devices",
			},
		),
		deviceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "info",
				Help:      "Info as reported by the device",
			},
			[]string{"index", "minor", "uuid", "name"},
		),
		temperatures: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "temperatures",
				Help:      "Temperature as reported by the device",
			},
			[]string{"minor"},
		),
		powerUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "power_usage",
				Help:      "Power usage as reported by the device",
			},
			[]string{"minor"},
		),
		powerUsageAverage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "power_usage_average",
				Help:      "Power usage as reported by the device averaged over 10s",
			},
			[]string{"minor"},
		),
		fanSpeed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "fanspeed",
				Help:      "Fan speed as reported by the device",
			},
			[]string{"minor"},
		),
		memoryTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_total",
				Help:      "Total memory as reported by the device",
			},
			[]string{"minor"},
		),
		memoryUsed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "memory_used",
				Help:      "Used memory as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_memory",
				Help:      "Memory Utilization as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationGPU: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_gpu",
				Help:      "GPU utilization as reported by the device",
			},
			[]string{"minor"},
		),
		utilizationGPUAverage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_gpu_average",
				Help:      "Used memory as reported by the device averraged over 10s",
			},
			[]string{"minor"},
		),
		encUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_encoder",
				Help:      "Percent of time over the last sample period during which the GPU video encoder was being used.",
			},
			[]string{"minor"},
		),
		decUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "utilization_decoder",
				Help:      "Percent of time over the last sample period during which the GPU video decoder was being used.",
			},
			[]string{"minor"},
		),
		dutyCycle: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "duty_cycle",
				Help:      "Percent of time over the past sample period during which one or more kernels were executing on the GPU device",
			},
			[]string{"minor"},
		),
		avgDuty: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "avg_duty_cycle",
				Help:      "Average time over the past 15 seconds during which one or more kernels were executing on the GPU device",
			},
			[]string{"minor"},
		),
	}
}

func (c *Collector) Describe(descs chan<- *prometheus.Desc) {
	c.deviceCount.Describe(descs)
	c.deviceInfo.Describe(descs)
	c.fanSpeed.Describe(descs)
	c.info.Describe(descs)
	c.memoryTotal.Describe(descs)
	c.memoryUsed.Describe(descs)
	c.powerUsage.Describe(descs)
	c.powerUsageAverage.Describe(descs)
	c.temperatures.Describe(descs)
	c.up.Describe(descs)
	c.utilizationGPU.Describe(descs)
	c.utilizationGPUAverage.Describe(descs)
	c.utilizationMemory.Describe(descs)
    c.encUsage.Describe(descs)
    c.decUsage.Describe(descs)
    c.dutyCycle.Describe(descs)
    c.avgDuty.Describe(descs)
}

func (e *Collector) Collect(metrics chan<- prometheus.Metric) {
	data, err := collectMetrics()
	if err != nil {
		log.Printf("Failed to collect metrics: %s\n", err)
		e.up.Set(0)
		e.up.Collect(metrics)
		return
	}

	e.up.Set(1)
	e.info.WithLabelValues(data.Version).Set(1)
	e.deviceCount.Set(float64(len(data.Devices)))

	for i := 0; i < len(data.Devices); i++ {
		d := data.Devices[i]
		e.deviceInfo.WithLabelValues(d.Index, d.MinorNumber, d.Name, d.UUID).Set(1)
        if d.FanSpeed.isPresent {
		    e.fanSpeed.WithLabelValues(d.MinorNumber).Set(d.FanSpeed.value)
        }

		e.memoryTotal.WithLabelValues(d.MinorNumber).Set(d.MemoryTotal.value)
		e.memoryUsed.WithLabelValues(d.MinorNumber).Set(d.MemoryUsed.value)
		e.powerUsage.WithLabelValues(d.MinorNumber).Set(d.PowerUsage.value)
		e.powerUsageAverage.WithLabelValues(d.MinorNumber).Set(d.PowerUsageAverage.value)
		e.temperatures.WithLabelValues(d.MinorNumber).Set(d.Temperature.value)
		e.utilizationGPU.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPU.value)
		e.utilizationGPUAverage.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPUAverage.value)
		e.utilizationMemory.WithLabelValues(d.MinorNumber).Set(d.UtilizationMemory.value)
        e.encUsage.WithLabelValues(d.MinorNumber).Set(d.EncUsage.value)
        e.decUsage.WithLabelValues(d.MinorNumber).Set(d.DecUsage.value)
        e.dutyCycle.WithLabelValues(d.MinorNumber).Set(d.DutyCycle.value)
        e.avgDuty.WithLabelValues(d.MinorNumber).Set(d.AvgDuty.value)
	}

	e.deviceCount.Collect(metrics)
	e.deviceInfo.Collect(metrics)
	e.fanSpeed.Collect(metrics)
	e.info.Collect(metrics)
	e.memoryTotal.Collect(metrics)
	e.memoryUsed.Collect(metrics)
	e.powerUsage.Collect(metrics)
	e.powerUsageAverage.Collect(metrics)
	e.temperatures.Collect(metrics)
	e.up.Collect(metrics)
	e.utilizationGPU.Collect(metrics)
	e.utilizationGPUAverage.Collect(metrics)
	e.utilizationMemory.Collect(metrics)
    e.encUsage.Collect(metrics)
    e.decUsage.Collect(metrics)
    e.dutyCycle.Collect(metrics)
    e.avgDuty.Collect(metrics)
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
