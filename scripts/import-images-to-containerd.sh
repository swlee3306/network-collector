#!/bin/bash

# Docker 이미지를 containerd로 import하는 스크립트
# 사용법: ./scripts/import-images-to-containerd.sh

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 컨테이너 런타임 확인
check_runtime() {
    if command -v ctr &> /dev/null; then
        RUNTIME="containerd"
        log_info "containerd 감지됨"
        return 0
    elif command -v crictl &> /dev/null; then
        RUNTIME="cri-o"
        log_info "CRI-O 감지됨 (crictl 사용)"
        return 0
    else
        log_error "containerd 또는 CRI-O를 찾을 수 없습니다."
        log_info "Docker만 사용하는 경우 이 스크립트는 필요하지 않습니다."
        return 1
    fi
}

# 이미지 import (containerd)
import_to_containerd() {
    local image=$1
    
    log_info "이미지 import 중: $image"
    
    # Docker에서 이미지를 tar로 저장
    TEMP_TAR=$(mktemp /tmp/docker-image-XXXXXX.tar)
    docker save "$image" -o "$TEMP_TAR"
    
    # containerd로 import
    if ctr -n k8s.io images import "$TEMP_TAR" 2>/dev/null; then
        log_info "✓ $image를 containerd로 import 완료"
        rm -f "$TEMP_TAR"
        return 0
    else
        log_warn "✗ $image import 실패"
        rm -f "$TEMP_TAR"
        return 1
    fi
}

# 이미지 import (CRI-O - crictl은 직접 import를 지원하지 않음)
import_to_crio() {
    local image=$1
    
    log_warn "CRI-O는 직접 import를 지원하지 않습니다."
    log_info "대신 워커 노드에 이미지를 배포하거나 레지스트리를 사용하세요."
    return 1
}

# 모든 이미지 import
import_all_images() {
    IMAGES=("network-collector:latest" "network-collector-api:latest" "network-collector-frontend:latest")
    
    log_info "모든 이미지를 컨테이너 런타임으로 import합니다..."
    
    for image in "${IMAGES[@]}"; do
        # Docker에 이미지가 있는지 확인
        if ! docker image inspect "$image" &> /dev/null; then
            log_warn "Docker에 이미지가 없습니다: $image"
            log_info "먼저 './scripts/build-images.sh'를 실행하세요."
            continue
        fi
        
        if [ "$RUNTIME" = "containerd" ]; then
            import_to_containerd "$image"
        elif [ "$RUNTIME" = "cri-o" ]; then
            import_to_crio "$image"
        fi
    done
}

# 메인 실행
main() {
    log_info "Docker 이미지를 Kubernetes 컨테이너 런타임으로 import 시작"
    
    if ! check_runtime; then
        exit 1
    fi
    
    import_all_images
    
    log_info "Import 완료!"
    log_info "확인: crictl images | grep network-collector"
}

main "$@"

