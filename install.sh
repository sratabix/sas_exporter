#!/bin/sh
set -e

REPO="sratabix/sas_exporter"
BINARY="sas_exporter"
BINDIR="/usr/local/bin"
UNITDIR="/etc/systemd/system"

die() { echo "Error: $*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "must be run as root."

case "$(uname -m)" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  *)       die "unsupported architecture: $(uname -m)" ;;
esac

VERSION="${1:-$(curl -sfLo /dev/null -w '%{url_effective}' \
  "https://github.com/${REPO}/releases/latest" | sed 's|.*/tag/||')}"
[ -n "$VERSION" ] || die "could not determine latest version."

echo "Installing ${BINARY} ${VERSION} (linux/${ARCH})..."

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

curl -fL "https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}_linux_${ARCH}" \
  -o "${tmp}/${BINARY}"
install -Dm755 "${tmp}/${BINARY}" "${BINDIR}/${BINARY}"

cat > "${UNITDIR}/sas_exporter.service" <<'EOF'
[Unit]
Description=SAS HBA Prometheus Exporter
Documentation=https://github.com/sratabix/sas_exporter
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/sas_exporter
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now sas_exporter

echo ""
echo "${BINARY} ${VERSION} installed and running on :9856"
echo "  update:  sas_exporter update"
echo "  remove:  sas_exporter remove"
