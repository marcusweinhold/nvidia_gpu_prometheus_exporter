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
    "fmt"

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

type Device struct {
	Index                 string
	MinorNumber           string
	Name                  string
	UUID                  string
	Temperature           float64
	PowerUsage            float64
	PowerUsageAverage     float64
	FanSpeed              float64
	MemoryTotal           float64
	MemoryUsed            float64
	UtilizationMemory     float64
	UtilizationGPU        float64
	UtilizationGPUAverage float64
    DutyCycle             float64
    AvgDuty               float64
    EncUsage              float64
    DecUsage              float64
}


func collectMetrics() (*Metrics, error) {
	// if err := gonvml.Initialize(); err != nil {
	// 	return nil, err
	// }
	// defer gonvml.Shutdown()

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
            log.Printf("Failed to get deviceByIndex", index, err)
			return nil, err
		}

		uuid, err := device.UUID()
		if err != nil {
			return nil, err
		}

		name, err := device.Name()
		if err != nil {
			return nil, err
		}

		minorNumber, err := device.MinorNumber()
		if err != nil {
			return nil, err
		}

		temperature, err := device.Temperature()
		if err != nil {
			return nil, err
		}

		powerUsage, err := device.PowerUsage()
		if err != nil {
			return nil, err
		}

		powerUsageAverage, err := device.AveragePowerUsage(averageDuration)
		if err != nil {
            powerUsageAverage = 0
		}

        var fanSpeed uint = 0

        if !*disableFanSpeed {
            //var err
		    fanSpeed, err = device.FanSpeed()
		    if err != nil {
                log.Printf("Failed fan")
                return nil, fmt.Errorf("FanSpeed: %w", err)
		    }
        }

		memoryTotal, memoryUsed, err := device.MemoryInfo()
		if err != nil {
			return nil, err
		}

		utilizationGPU, utilizationMemory, err := device.UtilizationRates()
		if err != nil {
			return nil, err
		}

		utilizationGPUAverage, err := device.AverageGPUUtilization(averageDuration)
		if err != nil {
			return nil, err
		}

		encUsage, _, err := device.EncoderUtilization()
		if err != nil {
			return nil, err
		}

		decUsage, _, err := device.DecoderUtilization()
		if err != nil {
			return nil, err
		}

		dutyCycle, _, err := device.UtilizationRates()
		if err != nil {
			return nil, err
		}

        avgDuty, err := device.AverageGPUUtilization(averageDuration)
		if err != nil {
			return nil, err
		}

		metrics.Devices = append(metrics.Devices,
			&Device{
				Index:                  strconv.Itoa(index),
				MinorNumber:            strconv.Itoa(int(minorNumber)),
				Name:                   name,
				UUID:                   uuid,
				Temperature:            float64(temperature),
				PowerUsage:             float64(powerUsage),
				PowerUsageAverage:      float64(powerUsageAverage),
				FanSpeed:               float64(fanSpeed),
				MemoryTotal:            float64(memoryTotal),
				MemoryUsed:             float64(memoryUsed),
				UtilizationMemory:      float64(utilizationMemory),
				UtilizationGPU:         float64(utilizationGPU),
				UtilizationGPUAverage:  float64(utilizationGPUAverage),
                EncUsage:               float64(encUsage),
                DecUsage:               float64(decUsage),
                DutyCycle:              float64(dutyCycle),
                AvgDuty:                float64(avgDuty),
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
	// ch <- c.numDevices.Desc()
	// c.usedMemory.Describe(ch)
	// c.totalMemory.Describe(ch)
	// c.dutyCycle.Describe(ch)
	// c.powerUsage.Describe(ch)
	// c.temperature.Describe(ch)
	// c.fanSpeed.Describe(ch)
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
		e.fanSpeed.WithLabelValues(d.MinorNumber).Set(d.FanSpeed)
		e.memoryTotal.WithLabelValues(d.MinorNumber).Set(d.MemoryTotal)
		e.memoryUsed.WithLabelValues(d.MinorNumber).Set(d.MemoryUsed)
		e.powerUsage.WithLabelValues(d.MinorNumber).Set(d.PowerUsage)
		e.powerUsageAverage.WithLabelValues(d.MinorNumber).Set(d.PowerUsageAverage)
		e.temperatures.WithLabelValues(d.MinorNumber).Set(d.Temperature)
		e.utilizationGPU.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPU)
		e.utilizationGPUAverage.WithLabelValues(d.MinorNumber).Set(d.UtilizationGPUAverage)
		e.utilizationMemory.WithLabelValues(d.MinorNumber).Set(d.UtilizationMemory)
        e.encUsage.WithLabelValues(d.MinorNumber).Set(d.EncUsage)
        e.decUsage.WithLabelValues(d.MinorNumber).Set(d.DecUsage)
        e.dutyCycle.WithLabelValues(d.MinorNumber).Set(d.DutyCycle)
        e.avgDuty.WithLabelValues(d.MinorNumber).Set(d.AvgDuty)
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
