#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATIC_REPO_URL="${STATIC_REPO_URL:-https://github.com/the-open-agent/static.git}"
STATIC_REPO_DIR="${STATIC_REPO_DIR:-}"
ICONFONT_URL="${ICONFONT_URL:-https://cdn.open-ct.com/icon/iconfont.js}"

if [[ -z "${STATIC_REPO_DIR}" ]]; then
  TMP_DIR="$(mktemp -d)"
  trap 'rm -rf "${TMP_DIR}"' EXIT
  STATIC_REPO_DIR="${TMP_DIR}/static"
  git clone --depth 1 "${STATIC_REPO_URL}" "${STATIC_REPO_DIR}"
fi

require_path() {
  local path="$1"
  if [[ ! -e "${path}" ]]; then
    echo "missing required embedded static asset: ${path}" >&2
    exit 1
  fi
}

PUBLIC_DIR="${ROOT_DIR}/web/public"

require_path "${STATIC_REPO_DIR}/img"
require_path "${STATIC_REPO_DIR}/flag-icons"
require_path "${STATIC_REPO_DIR}/gravatar"

rm -rf "${PUBLIC_DIR}/img" "${PUBLIC_DIR}/flag-icons" "${PUBLIC_DIR}/gravatar" "${PUBLIC_DIR}/icon"
cp -R "${STATIC_REPO_DIR}/img" "${PUBLIC_DIR}/img"
cp -R "${STATIC_REPO_DIR}/flag-icons" "${PUBLIC_DIR}/flag-icons"
cp -R "${STATIC_REPO_DIR}/gravatar" "${PUBLIC_DIR}/gravatar"
mkdir -p "${PUBLIC_DIR}/icon"
curl -fsSL "${ICONFONT_URL}" -o "${PUBLIC_DIR}/icon/iconfont.js"

sed -i \
  -e 's#https://cdn.openagentai.org/img/openagent.png#/img/openagent.png#g' \
  -e 's#https://cdn.openagentai.org/site/openagent/manifest.json#/manifest.json#g' \
  "${PUBLIC_DIR}/index.html"
