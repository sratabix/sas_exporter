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
	"SAS controller temperature in Celsius, read from the mpt3sas/mpt2sas hwmon sysfs interface.",
	[]string{"pci_address", "sensor", "label"},
	nil,
)

// HwmonCollector reads controller temperatures from the mpt3sas/mpt2sas hwmon
// sysfs interface. It emits no metrics (silently) if no matching hwmon devices
// are found, which is expected on kernels that predate hwmon support in those
// drivers or where the feature is not compiled in.
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
		// Not Linux, or /sys/class/hwmon doesn't exist — skip silently.
		return
	}

	for _, entry := range entries {
		hwmonPath := filepath.Join(c.root, entry.Name())

		if !isSASDriver(hwmonPath) {
			continue
		}

		pciAddr := hwmonPCIAddress(hwmonPath)
		temps, err := readTempInputs(hwmonPath)
		if err != nil {
			log.Printf("sas_exporter: hwmon %s: %v", entry.Name(), err)
			continue
		}

		for _, t := range temps {
			ch <- prometheus.MustNewConstMetric(
				controllerTempDesc, prometheus.GaugeValue, t.celsius,
				pciAddr, t.sensor, t.label,
			)
		}
	}
}

// isSASDriver returns true if the hwmon device belongs to mpt3sas or mpt2sas.
func isSASDriver(hwmonPath string) bool {
	// /sys/class/hwmon/hwmonN/device/driver -> ../../bus/pci/drivers/mpt3sas
	target, err := os.Readlink(filepath.Join(hwmonPath, "device", "driver"))
	if err != nil {
		return false
	}
	name := filepath.Base(target)
	return name == "mpt3sas" || name == "mpt2sas"
}

// hwmonPCIAddress resolves the hwmon device symlink to its PCI device path
// and returns the final component, e.g. "0000:03:00.0".
func hwmonPCIAddress(hwmonPath string) string {
	devicePath, err := filepath.EvalSymlinks(filepath.Join(hwmonPath, "device"))
	if err != nil {
		return ""
	}
	return filepath.Base(devicePath)
}

type tempReading struct {
	sensor  string  // e.g. "temp1"
	label   string  // e.g. "IOCTemp", empty if no label file
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
		sensor := strings.TrimSuffix(name, "_input") // "temp1"

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
