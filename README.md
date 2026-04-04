# sas_exporter

Prometheus exporter for LSI SAS HBA controllers. Exposes controller info and drive temperatures via `sas2ircu` / `sas3ircu`.

## Requirements

- Linux
- `sas3ircu` and/or `sas2ircu` in `$PATH` (or specify paths via flags)
- Root privileges (ircu tools require direct PCI access)

## Install

```sh
curl -fsSL https://github.com/sratabix/sas_exporter/releases/latest/download/install.sh | sudo sh
```

Metrics are exposed on `:9856/metrics`.

## Manage

```sh
sudo sas_exporter update   # update to latest release
sudo sas_exporter remove   # stop service and uninstall
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--web.listen-address` | `:9856` | Address to expose metrics on |
| `--web.telemetry-path` | `/metrics` | Path to expose metrics on |
| `--sas3ircu` | `sas3ircu` | Path to sas3ircu binary |
| `--sas2ircu` | `sas2ircu` | Path to sas2ircu binary |
| `--hwmon.path` | `/sys/class/hwmon` | Path to hwmon sysfs root |

Override flags by editing `/etc/systemd/system/sas_exporter.service`.

## Metrics

| Metric | Description |
|---|---|
| `sas_controller_info` | Controller metadata (type, firmware, BIOS, PCI address) |
| `sas_physical_device_info` | Per-drive metadata (state, protocol, drive type, model, serial) |
| `sas_physical_device_temperature_celsius` | Drive temperature |
| `sas_controller_temperature_celsius` | Controller temperature via hwmon (kernel ≥ 5.x, if supported) |
| `sas_exporter_tool_up` | 1 if the ircu tool ran successfully, 0 otherwise |

## Build from source

```sh
make build
# binary at ./bin/sas_exporter
```
