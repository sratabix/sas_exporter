package collector

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var controllerTempDesc = prometheus.NewDesc(
	prometheus.BuildFQName(namespace, "controller", "temperature_celsius"),
	"SAS controller temperature in Celsius.",
	[]string{"controller", "sensor", "label"},
	nil,
)

type HwmonCollector struct {
	root string
}

func NewHwmonCollector(hwmonRoot string) *HwmonCollector {
	return &HwmonCollector{root: hwmonRoot}
}

func (c *HwmonCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- controllerTempDesc
}

func (c *HwmonCollector) Collect(ch chan<- prometheus.Metric) {
	entries, err := os.ReadDir(c.root)
	if err != nil {

		return
	}

	for _, entry := range entries {
		hwmonPath := filepath.Join(c.root, entry.Name())

		if !isSASDriver(hwmonPath) {
			continue
		}

		controller := hwmonPCIAddress(hwmonPath)
		temps, err := readTempInputs(hwmonPath)
		if err != nil {
			log.Printf("sas_exporter: hwmon %s: %v", entry.Name(), err)
			continue
		}

		for _, t := range temps {
			ch <- prometheus.MustNewConstMetric(
				controllerTempDesc, prometheus.GaugeValue, t.celsius,
				controller, t.sensor, t.label,
			)
		}
	}
}

func isSASDriver(hwmonPath string) bool {

	target, err := os.Readlink(filepath.Join(hwmonPath, "device", "driver"))
	if err != nil {
		return false
	}
	name := filepath.Base(target)
	return name == "mpt3sas" || name == "mpt2sas"
}

func hwmonPCIAddress(hwmonPath string) string {
	devicePath, err := filepath.EvalSymlinks(filepath.Join(hwmonPath, "device"))
	if err != nil {
		return ""
	}
	return filepath.Base(devicePath)
}

type tempReading struct {
	sensor  string
	label   string
	celsius float64
}

func readTempInputs(hwmonPath string) ([]tempReading, error) {
	entries, err := os.ReadDir(hwmonPath)
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}

	var temps []tempReading
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "temp") || !strings.HasSuffix(name, "_input") {
			continue
		}
		sensor := strings.TrimSuffix(name, "_input")

		raw, err := os.ReadFile(filepath.Join(hwmonPath, name))
		if err != nil {
			continue
		}
		millideg, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil {
			continue
		}

		label := ""
		if lb, err := os.ReadFile(filepath.Join(hwmonPath, sensor+"_label")); err == nil {
			label = strings.TrimSpace(string(lb))
		}

		temps = append(temps, tempReading{
			sensor:  sensor,
			label:   label,
			celsius: millideg / 1000.0,
		})
	}
	return temps, nil
}
