#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIHOMO_VERSION="${MIHOMO_VERSION:-v1.19.27}"
MIHOMO_BIN_DIR="${MIHOMO_BIN_DIR:-${ROOT_DIR}/.local/bin}"
MIHOMO_BIN="${ADMIN_PLUS_PROXY_MIHOMO_BINARY_PATH:-${MIHOMO_BIN_DIR}/mihomo}"
MIHOMO_BIN_DIR="$(dirname "${MIHOMO_BIN}")"

detect_os() {
  case "$(uname -s | tr '[:upper:]' '[:lower:]')" in
    linux) echo "linux" ;;
    darwin) echo "darwin" ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported arch: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

asset_name() {
  local os="$1"
  local arch="$2"
  case "${os}/${arch}" in
    linux/amd64) echo "mihomo-linux-amd64-compatible-${MIHOMO_VERSION}.gz" ;;
    linux/arm64) echo "mihomo-linux-arm64-${MIHOMO_VERSION}.gz" ;;
    darwin/amd64) echo "mihomo-darwin-amd64-compatible-${MIHOMO_VERSION}.gz" ;;
    darwin/arm64) echo "mihomo-darwin-arm64-${MIHOMO_VERSION}.gz" ;;
    *)
      echo "unsupported platform: ${os}/${arch}" >&2
      exit 1
      ;;
  esac
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

if [ -x "${MIHOMO_BIN}" ]; then
  echo "${MIHOMO_BIN}"
  exit 0
fi

require_command curl
require_command gzip

os="$(detect_os)"
arch="$(detect_arch)"
asset="$(asset_name "${os}" "${arch}")"
url="https://github.com/MetaCubeX/mihomo/releases/download/${MIHOMO_VERSION}/${asset}"

mkdir -p "${MIHOMO_BIN_DIR}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "${tmp_dir}"' EXIT

echo "downloading Mihomo core ${MIHOMO_VERSION} for ${os}/${arch}" >&2
curl -fsSL "${url}" -o "${tmp_dir}/mihomo.gz"
gzip -dc "${tmp_dir}/mihomo.gz" > "${tmp_dir}/mihomo"
chmod +x "${tmp_dir}/mihomo"
mv "${tmp_dir}/mihomo" "${MIHOMO_BIN}"

echo "${MIHOMO_BIN}"
