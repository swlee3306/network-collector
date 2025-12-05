#!/bin/bash

# Docker 이미지 빌드 스크립트
# 사용법: ./scripts/build-images.sh [registry] [tag]

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

IMAGE_REGISTRY=${1:-""}
IMAGE_TAG=${2:-latest}

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 이미지 이름 생성
image_name() {
    local component=$1
    if [ -n "$IMAGE_REGISTRY" ]; then
        echo "${IMAGE_REGISTRY}/${component}:${IMAGE_TAG}"
    else
        echo "${component}:${IMAGE_TAG}"
    fi
}

# Collector 이미지 빌드
build_collector() {
    log_info "Collector 이미지 빌드 중..."
    
    local image=$(image_name "network-collector")
    
    cd "$PROJECT_ROOT/backend"
    docker build -t "$image" -f Dockerfile.collector .
    cd "$PROJECT_ROOT"
    
    log_info "Collector 이미지 빌드 완료: $image"
}

# API 이미지 빌드
build_api() {
    log_info "API 이미지 빌드 중..."
    
    local image=$(image_name "network-collector-api")
    
    cd "$PROJECT_ROOT/backend"
    docker build -t "$image" -f Dockerfile.api .
    cd "$PROJECT_ROOT"
    
    log_info "API 이미지 빌드 완료: $image"
}

# Frontend 이미지 빌드
build_frontend() {
    log_info "Frontend 이미지 빌드 중..."
    
    local image=$(image_name "network-collector-frontend")
    
    cd "$PROJECT_ROOT/frontend"
    docker build -t "$image" -f Dockerfile .
    cd "$PROJECT_ROOT"
    
    log_info "Frontend 이미지 빌드 완료: $image"
}

# 모든 이미지 빌드
build_all() {
    log_info "모든 이미지 빌드 시작..."
    log_info "레지스트리: ${IMAGE_REGISTRY:-local}"
    log_info "태그: $IMAGE_TAG"
    
    build_collector
    build_api
    build_frontend
    
    log_info "모든 이미지 빌드 완료!"
    
    echo
    echo "빌드된 이미지:"
    docker images | grep -E "network-collector|IMAGE"
}

# 메인 실행
main() {
    if ! command -v docker &> /dev/null; then
        log_warn "Docker가 설치되어 있지 않습니다."
        exit 1
    fi
    
    build_all
}

main "$@"

