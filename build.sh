#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="${OUTPUT_DIR:-}"

# Default build configuration can be overridden by environment variables.
GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
CGO_ENABLED="${CGO_ENABLED:-0}"
GOCACHE="${GOCACHE:-${ROOT_DIR}/.cache/go-build}"

# Format: name:package_path:binary_destination_directory
TARGETS=(
  "auth-rpc:app/services/auth:app/services/auth"
  "user-rpc:app/services/user:app/services/user"
  "inventory-rpc:app/services/inventory:app/services/inventory"
  "product-rpc:app/services/product:app/services/product"
  "cart-rpc:app/services/cart:app/services/cart"
  "coupon-rpc:app/services/coupon:app/services/coupon"
  "order-rpc:app/services/order:app/services/order"
  "user-api:app/api/user:app/api/user"
  "inventory-api:app/api/inventory:app/api/inventory"
  "product-api:app/api/product:app/api/product"
  "cart-api:app/api/cart:app/api/cart"
  "coupon-api:app/api/coupon:app/api/coupon"
  "order-api:app/api/order:app/api/order"
)

usage() {
  echo "Usage: $(basename "$0") [target ...]" >&2
  echo "Targets:" >&2
  for entry in "${TARGETS[@]}"; do
    IFS=":" read -r name _ _ <<<"${entry}"
    echo "  - ${name}" >&2
  done
}

ensure_target_known() {
  local name=$1
  for entry in "${TARGETS[@]}"; do
    IFS=":" read -r candidate _ _ <<<"${entry}"
    if [[ "${candidate}" == "${name}" ]]; then
      return 0
    fi
  done
  echo "Unknown target: ${name}" >&2
  usage
  exit 1
}

targets_to_build=()

if [[ $# -gt 0 ]]; then
  for arg in "$@"; do
    ensure_target_known "${arg}"
    targets_to_build+=("${arg}")
  done
else
  for entry in "${TARGETS[@]}"; do
    IFS=":" read -r name _ _ <<<"${entry}"
    targets_to_build+=("${name}")
  done
fi

mkdir -p "${GOCACHE}"
if [[ -n "${OUTPUT_DIR}" ]]; then
  mkdir -p "${OUTPUT_DIR}"
fi

for entry in "${TARGETS[@]}"; do
  IFS=":" read -r name rel_path dest_path <<<"${entry}"

  should_build=false
  for selected in "${targets_to_build[@]}"; do
    if [[ "${selected}" == "${name}" ]]; then
      should_build=true
      break
    fi
  done

  if [[ "${should_build}" != true ]]; then
    continue
  fi

  pkg="./${rel_path}"
  dest_dir="${ROOT_DIR}/${dest_path}"
  output_path="${dest_dir}/${name}"

  mkdir -p "${dest_dir}"

  echo "==> Building ${name} (${GOOS}/${GOARCH})"
  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED="${CGO_ENABLED}" GOCACHE="${GOCACHE}" \
    go build -o "${output_path}" "${pkg}"

  if [[ -n "${OUTPUT_DIR}" ]]; then
    cp "${output_path}" "${OUTPUT_DIR}/${name}"
  fi
done

if [[ -n "${OUTPUT_DIR}" ]]; then
  echo "All requested targets built beside their Dockerfiles and copied to ${OUTPUT_DIR}"
else
  echo "All requested targets built beside their Dockerfiles"
fi
