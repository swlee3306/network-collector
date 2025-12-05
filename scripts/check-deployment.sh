#!/bin/bash

# 배포 전 점검 스크립트
# 사용법: ./scripts/check-deployment.sh

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

# 마스터 노드 확인
check_master_node() {
    log_section "마스터 노드 확인"
    
    MASTER_NODE=$(kubectl get nodes -l node-role.kubernetes.io/control-plane= -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    
    if [ -z "$MASTER_NODE" ]; then
        log_error "마스터 노드를 찾을 수 없습니다."
        return 1
    fi
    
    log_info "마스터 노드: $MASTER_NODE"
    
    # 마스터 노드 라벨 확인
    log_info "마스터 노드 라벨:"
    kubectl get node "$MASTER_NODE" --show-labels | grep -E "NAME|$MASTER_NODE"
    
    # 마스터 노드 taint 확인
    log_info "마스터 노드 Taint:"
    kubectl describe node "$MASTER_NODE" | grep -A 5 "Taints:" || log_warn "Taint가 없습니다."
    
    return 0
}

# 이미지 확인
check_images() {
    log_section "이미지 확인"
    
    IMAGES=("network-collector:latest" "network-collector-api:latest" "network-collector-frontend:latest")
    
    log_info "현재 노드에서 이미지 확인 중..."
    if command -v docker &> /dev/null; then
        for image in "${IMAGES[@]}"; do
            if docker images | grep -q "$image"; then
                log_info "✓ $image 존재"
            else
                log_error "✗ $image 없음"
            fi
        done
    else
        log_warn "Docker가 설치되어 있지 않습니다. 이미지 확인을 건너뜁니다."
    fi
}

# Deployment 설정 확인
check_deployment_config() {
    log_section "Deployment 설정 확인"
    
    DEPLOYMENTS=("network-collector" "network-collector-api" "network-collector-frontend")
    
    for deployment in "${DEPLOYMENTS[@]}"; do
        if kubectl get deployment "$deployment" -n default &>/dev/null; then
            log_info "Deployment: $deployment"
            
            # nodeSelector 확인
            NODE_SELECTOR=$(kubectl get deployment "$deployment" -n default -o jsonpath='{.spec.template.spec.nodeSelector}' 2>/dev/null || echo "없음")
            log_info "  nodeSelector: $NODE_SELECTOR"
            
            # tolerations 확인
            TOLERATIONS=$(kubectl get deployment "$deployment" -n default -o jsonpath='{.spec.template.spec.tolerations}' 2>/dev/null || echo "없음")
            if [ "$TOLERATIONS" != "없음" ] && [ -n "$TOLERATIONS" ]; then
                log_info "  tolerations: 설정됨"
            else
                log_warn "  tolerations: 없음"
            fi
            
            # imagePullPolicy 확인
            IMAGE_PULL_POLICY=$(kubectl get deployment "$deployment" -n default -o jsonpath='{.spec.template.spec.containers[0].imagePullPolicy}' 2>/dev/null || echo "없음")
            log_info "  imagePullPolicy: $IMAGE_PULL_POLICY"
            
            # 이미지 이름 확인
            IMAGE=$(kubectl get deployment "$deployment" -n default -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || echo "없음")
            log_info "  image: $IMAGE"
        else
            log_warn "Deployment '$deployment'가 존재하지 않습니다."
        fi
    done
}

# Pod 상태 확인
check_pod_status() {
    log_section "Pod 상태 확인"
    
    kubectl get pods -n default -l 'app in (network-collector,network-collector-api,network-collector-frontend)' -o wide
    
    echo
    log_info "Pod 상세 정보:"
    PODS=$(kubectl get pods -n default -l 'app in (network-collector,network-collector-api,network-collector-frontend)' -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")
    
    if [ -z "$PODS" ]; then
        log_warn "Pod가 없습니다."
        return
    fi
    
    for pod in $PODS; do
        echo
        log_info "Pod: $pod"
        
        # Pod 상태
        STATUS=$(kubectl get pod "$pod" -n default -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        log_info "  상태: $STATUS"
        
        # 노드
        NODE=$(kubectl get pod "$pod" -n default -o jsonpath='{.spec.nodeName}' 2>/dev/null || echo "Unknown")
        log_info "  노드: $NODE"
        
        # 이미지 Pull 에러 확인
        if [ "$STATUS" != "Running" ]; then
            log_warn "  이벤트 확인 중..."
            kubectl describe pod "$pod" -n default | grep -A 10 "Events:" | head -15
        fi
    done
}

# ConfigMap 및 Secret 확인
check_resources() {
    log_section "ConfigMap 및 Secret 확인"
    
    # ConfigMap 확인
    if kubectl get configmap network-collector-config -n default &>/dev/null; then
        log_info "✓ ConfigMap 'network-collector-config' 존재"
    else
        log_error "✗ ConfigMap 'network-collector-config' 없음"
    fi
    
    if kubectl get configmap network-collector-frontend-config -n default &>/dev/null; then
        log_info "✓ ConfigMap 'network-collector-frontend-config' 존재"
    else
        log_error "✗ ConfigMap 'network-collector-frontend-config' 없음"
    fi
    
    # Secret 확인
    if kubectl get secret network-collector-secrets -n default &>/dev/null; then
        log_info "✓ Secret 'network-collector-secrets' 존재"
    else
        log_error "✗ Secret 'network-collector-secrets' 없음"
    fi
}

# 메인 실행
main() {
    log_info "배포 상태 점검 시작"
    
    check_master_node
    check_images
    check_resources
    check_deployment_config
    check_pod_status
    
    log_section "점검 완료"
    log_info "문제가 발견되면 위의 오류 메시지를 확인하세요."
}

main "$@"

