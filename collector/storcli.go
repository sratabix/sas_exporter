package collector

import (
	"bufio"
	"bytes"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// storCLITempRe matches lines like:
//
//	ROC temperature(Degree Celsius) = 64
//	Ctrl temperature(Degree Celcius) = 64   ← Broadcom typo in older firmware
var storCLITempRe = regexp.MustCompile(`(?i)^(\w+)\s+temperature\(Degree\s+Cel[sc]ius\)[^0-9]*(\d+)`)
var storCLIControllerRe = regexp.MustCompile(`(?i)^Controller\s*=\s*(\d+)`)

// StorCLICollector collects controller temperatures via the storcli utility.
// It targets MegaRAID controllers; if storcli is not present or finds no
// controllers it emits a tool_up=0 metric and no temperature metrics.
type StorCLICollector struct {
	path string
}

func NewStorCLICollector(path string) *StorCLICollector {
	return &StorCLICollector{path: path}
}

// Describe intentionally emits nothing, making this an unchecked collector.
// controllerTempDesc is owned by HwmonCollector and toolUpDesc by IrcuCollector;
// both descriptors are shared and can only be described by one collector each.
func (c *StorCLICollector) Describe(_ chan<- *prometheus.Desc) {}

func (c *StorCLICollector) Collect(ch chan<- prometheus.Metric) {
	out, err := runTool(c.path, "/cALL", "show", "temperature")
	if err != nil {
		if !binaryNotFound(err) {
			log.Printf("sas_exporter: storcli: %v", err)
		}
		ch <- prometheus.MustNewConstMetric(toolUpDesc, prometheus.GaugeValue, 0, "storcli")
		return
	}
	ch <- prometheus.MustNewConstMetric(toolUpDesc, prometheus.GaugeValue, 1, "storcli")

	var ctrl string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if m := storCLIControllerRe.FindStringSubmatch(line); m != nil {
			ctrl = m[1]
			continue
		}

		if ctrl == "" {
			continue
		}

		if m := storCLITempRe.FindStringSubmatch(line); m != nil {
			sensor := strings.ToLower(m[1]) // "roc" or "ctrl"
			val, err := strconv.ParseFloat(m[2], 64)
			if err != nil {
				continue
			}
			ch <- prometheus.MustNewConstMetric(
				controllerTempDesc, prometheus.GaugeValue, val,
				ctrl, sensor, m[1]+" temperature",
			)
		}
	}
}
