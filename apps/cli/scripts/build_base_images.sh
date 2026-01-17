#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REGISTRY="${SHUTTL_BASE_REGISTRY:-shuttl}"
TAG="${SHUTTL_BASE_TAG:-latest}"
PUSH="${SHUTTL_BASE_PUSH:-false}"

LANGUAGES=(python node java csharp go)
ARCHES=(amd64 arm64)

for arch in "${ARCHES[@]}"; do
  bin_path="${ROOT_DIR}/dist/shuttl-linux-${arch}"
  if [[ ! -f "${bin_path}" ]]; then
    echo "Missing CLI binary: ${bin_path}"
    echo "Build it first (e.g., nx run cli:build:linux-${arch})."
    exit 1
  fi
done

for language in "${LANGUAGES[@]}"; do
  for arch in "${ARCHES[@]}"; do
    image="${REGISTRY}/shuttl-base-${language}:${TAG}-${arch}"
    echo "Building ${image}"
    cmd=(
      docker buildx build
      --platform "linux/${arch}"
      --build-arg "LANGUAGE=${language}"
      -f "${ROOT_DIR}/docker/base/Dockerfile"
      -t "${image}"
      "${ROOT_DIR}"
    )
    if [[ "${PUSH}" == "true" ]]; then
      cmd+=(--push)
    else
      cmd+=(--load)
    fi
    "${cmd[@]}"
  done
done

