package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sratabix/sas_exporter/collector"
)

// version is overridden at build time via -ldflags "-X main.version=..."
var version = "0.1.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "update":
			if err := selfUpdate(); err != nil {
				log.Fatal(err)
			}
			return
		case "remove":
			if err := selfRemove(); err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	var (
		listenAddr   = flag.String("web.listen-address", ":9856", "Address on which to expose metrics.")
		metricsPath  = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		sas3ircuPath = flag.String("sas3ircu", "sas3ircu", "Path to the sas3ircu binary.")
		sas2ircuPath = flag.String("sas2ircu", "sas2ircu", "Path to the sas2ircu binary.")
		hwmonRoot    = flag.String("hwmon.path", "/sys/class/hwmon", "Path to the hwmon sysfs root.")
	)
	flag.Parse()

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		collector.NewIrcuCollector(*sas3ircuPath, *sas2ircuPath),
		collector.NewHwmonCollector(*hwmonRoot),
	)

	http.Handle(*metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html><head><title>SAS Exporter</title></head>
<body>
<h1>SAS HBA Exporter</h1>
<p><a href="` + *metricsPath + `">Metrics</a></p>
<p>Version: ` + version + `</p>
</body></html>`))
	})

	log.Printf("sas_exporter %s listening on %s", version, *listenAddr)
	log.Fatal(http.ListenAndServe(*listenAddr, nil))
}
