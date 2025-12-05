#!/bin/bash

# 기존 Docker 이미지를 사용하여 Kubernetes에 배포하는 스크립트
# 사용법: ./scripts/use-existing-images.sh

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# 색상 출력
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

log_question() {
    echo -e "${BLUE}[?]${NC} $1"
}

# 현재 Docker 이미지 확인
echo "=== 현재 Docker 이미지 ==="
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}" | head -20

echo ""
log_question "사용할 이미지를 선택하세요:"

# Collector 이미지 선택
echo ""
log_info "Collector 이미지:"
read -p "  이미지 이름 (예: network-collector:latest 또는 my-collector:v1.0): " COLLECTOR_IMAGE

if [ -z "$COLLECTOR_IMAGE" ]; then
    COLLECTOR_IMAGE="network-collector:latest"
    log_info "기본값 사용: $COLLECTOR_IMAGE"
fi

# 이미지 존재 확인
if ! docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${COLLECTOR_IMAGE}$"; then
    log_warn "이미지 '$COLLECTOR_IMAGE'를 찾을 수 없습니다."
    read -p "계속하시겠습니까? (y/N): " CONTINUE
    if [ "$CONTINUE" != "y" ] && [ "$CONTINUE" != "Y" ]; then
        exit 1
    fi
fi

# API 이미지 선택
echo ""
log_info "API 이미지:"
read -p "  이미지 이름 (예: network-collector-api:latest 또는 my-api:v1.0): " API_IMAGE

if [ -z "$API_IMAGE" ]; then
    API_IMAGE="network-collector-api:latest"
    log_info "기본값 사용: $API_IMAGE"
fi

# Frontend 이미지 선택
echo ""
log_info "Frontend 이미지:"
read -p "  이미지 이름 (예: network-collector-frontend:latest 또는 my-frontend:v1.0): " FRONTEND_IMAGE

if [ -z "$FRONTEND_IMAGE" ]; then
    FRONTEND_IMAGE="network-collector-frontend:latest"
    log_info "기본값 사용: $FRONTEND_IMAGE"
fi

# 표준 이름으로 태그할지 확인
echo ""
log_question "이미지를 표준 이름(network-collector:latest 등)으로 태그하시겠습니까? (y/N): "
read -p "  " TAG_STANDARD

if [ "$TAG_STANDARD" = "y" ] || [ "$TAG_STANDARD" = "Y" ]; then
    log_info "이미지 태그 중..."
    
    if [ "$COLLECTOR_IMAGE" != "network-collector:latest" ]; then
        docker tag "$COLLECTOR_IMAGE" network-collector:latest
        log_info "Tagged $COLLECTOR_IMAGE as network-collector:latest"
    fi
    
    if [ "$API_IMAGE" != "network-collector-api:latest" ]; then
        docker tag "$API_IMAGE" network-collector-api:latest
        log_info "Tagged $API_IMAGE as network-collector-api:latest"
    fi
    
    if [ "$FRONTEND_IMAGE" != "network-collector-frontend:latest" ]; then
        docker tag "$FRONTEND_IMAGE" network-collector-frontend:latest
        log_info "Tagged $FRONTEND_IMAGE as network-collector-frontend:latest"
    fi
    
    COLLECTOR_IMAGE="network-collector:latest"
    API_IMAGE="network-collector-api:latest"
    FRONTEND_IMAGE="network-collector-frontend:latest"
fi

# Deployment에 이미지 적용할지 확인
echo ""
log_question "Deployment에 이미지를 적용하시겠습니까? (y/N): "
read -p "  " APPLY_DEPLOYMENT

if [ "$APPLY_DEPLOYMENT" = "y" ] || [ "$APPLY_DEPLOYMENT" = "Y" ]; then
    log_info "Deployment 이미지 업데이트 중..."
    
    kubectl set image deployment/network-collector collector="$COLLECTOR_IMAGE" 2>/dev/null || \
        log_warn "network-collector deployment를 찾을 수 없습니다."
    
    kubectl set image deployment/network-collector-api api="$API_IMAGE" 2>/dev/null || \
        log_warn "network-collector-api deployment를 찾을 수 없습니다."
    
    kubectl set image deployment/network-collector-frontend frontend="$FRONTEND_IMAGE" 2>/dev/null || \
        log_warn "network-collector-frontend deployment를 찾을 수 없습니다."
    
    log_info "Deployment 이미지 업데이트 완료"
fi

# 다른 노드로 배포할지 확인
echo ""
log_question "다른 Kubernetes 노드에도 이미지를 배포하시겠습니까? (y/N): "
read -p "  " DEPLOY_TO_NODES

if [ "$DEPLOY_TO_NODES" = "y" ] || [ "$DEPLOY_TO_NODES" = "Y" ]; then
    log_info "이미지를 tar 파일로 저장 중..."
    docker save "$COLLECTOR_IMAGE" "$API_IMAGE" "$FRONTEND_IMAGE" -o /tmp/network-collector-images.tar
    
    log_info "배포할 노드 목록을 입력하세요 (공백으로 구분):"
    read -p "  " NODES
    
    for node in $NODES; do
        log_info "$node에 이미지 배포 중..."
        scp /tmp/network-collector-images.tar "$node:/tmp/" 2>/dev/null || \
            log_warn "$node로 복사 실패 (SSH 설정 확인 필요)"
        
        ssh "$node" "docker load -i /tmp/network-collector-images.tar && rm /tmp/network-collector-images.tar" 2>/dev/null || \
            log_warn "$node에서 이미지 로드 실패"
    done
    
    rm -f /tmp/network-collector-images.tar
    log_info "노드 배포 완료"
fi

echo ""
log_info "=== 완료 ==="
echo ""
echo "사용된 이미지:"
echo "  Collector: $COLLECTOR_IMAGE"
echo "  API: $API_IMAGE"
echo "  Frontend: $FRONTEND_IMAGE"
echo ""
echo "확인:"
echo "  docker images | grep network-collector"
echo "  kubectl get pods -o wide"

