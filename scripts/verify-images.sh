#!/bin/bash

# 이미지 확인 및 빌드 스크립트
# 사용법: ./scripts/verify-images.sh

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Docker 확인
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker가 설치되어 있지 않습니다."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker 데몬이 실행 중이지 않습니다."
        exit 1
    fi
    
    log_info "Docker 확인 완료"
}

# 이미지 확인 (더 정확한 방법)
check_images() {
    log_section "이미지 확인"
    
    IMAGES=("network-collector:latest" "network-collector-api:latest" "network-collector-frontend:latest")
    MISSING_IMAGES=()
    
    for image in "${IMAGES[@]}"; do
        # docker image inspect를 사용하여 더 정확하게 확인
        if docker image inspect "$image" &> /dev/null; then
            log_info "✓ $image 존재"
            # 이미지 상세 정보 출력
            docker image inspect "$image" --format '  ID: {{.Id}}' 2>/dev/null | head -1 || true
            docker image inspect "$image" --format '  Created: {{.Created}}' 2>/dev/null | head -1 || true
        else
            log_warn "✗ $image 없음"
            MISSING_IMAGES+=("$image")
        fi
    done
    
    if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
        echo
        log_error "다음 이미지들이 없습니다:"
        for img in "${MISSING_IMAGES[@]}"; do
            echo "  - $img"
        done
        return 1
    fi
    
    return 0
}

# 이미지 빌드
build_missing_images() {
    log_section "이미지 빌드"
    
    MISSING_COLLECTOR=false
    MISSING_API=false
    MISSING_FRONTEND=false
    
    if ! docker images | grep -q "network-collector:latest"; then
        MISSING_COLLECTOR=true
    fi
    
    if ! docker images | grep -q "network-collector-api:latest"; then
        MISSING_API=true
    fi
    
    if ! docker images | grep -q "network-collector-frontend:latest"; then
        MISSING_FRONTEND=true
    fi
    
    if [ "$MISSING_COLLECTOR" = true ] || [ "$MISSING_API" = true ] || [ "$MISSING_FRONTEND" = true ]; then
        log_info "누락된 이미지를 빌드합니다..."
        
        if [ "$MISSING_COLLECTOR" = true ]; then
            log_info "Collector 이미지 빌드 중..."
            cd "$PROJECT_ROOT/backend"
            docker build -t network-collector:latest -f Dockerfile.collector .
            cd "$PROJECT_ROOT"
            log_info "Collector 이미지 빌드 완료"
        fi
        
        if [ "$MISSING_API" = true ]; then
            log_info "API 이미지 빌드 중..."
            cd "$PROJECT_ROOT/backend"
            docker build -t network-collector-api:latest -f Dockerfile.api .
            cd "$PROJECT_ROOT"
            log_info "API 이미지 빌드 완료"
        fi
        
        if [ "$MISSING_FRONTEND" = true ]; then
            log_info "Frontend 이미지 빌드 중..."
            cd "$PROJECT_ROOT/frontend"
            docker build -t network-collector-frontend:latest -f Dockerfile .
            cd "$PROJECT_ROOT"
            log_info "Frontend 이미지 빌드 완료"
        fi
        
        log_info "이미지 빌드 완료!"
    else
        log_info "모든 이미지가 존재합니다. 빌드할 필요가 없습니다."
    fi
}

# 메인 실행
main() {
    log_info "이미지 확인 및 빌드 시작"
    
    check_docker
    
    if ! check_images; then
        echo
        read -p "누락된 이미지를 지금 빌드하시겠습니까? (y/N): " BUILD_NOW
        if [ "$BUILD_NOW" = "y" ] || [ "$BUILD_NOW" = "Y" ]; then
            build_missing_images
            
            echo
            log_section "빌드 후 이미지 확인"
            check_images
        else
            log_warn "이미지 빌드를 건너뜁니다."
            log_info "수동으로 빌드하려면: ./scripts/build-images.sh"
            exit 1
        fi
    else
        log_info "모든 이미지가 준비되었습니다!"
    fi
    
    echo
    log_section "최종 이미지 목록"
    docker images | grep -E "network-collector|REPOSITORY" | head -4
}

main "$@"

