#!/bin/bash

# OpenStack Monitoring System 제거 스크립트
# 사용법: ./scripts/undeploy.sh [k8s|helm] [namespace]

set -e

DEPLOYMENT_TYPE=${1:-k8s}
NAMESPACE=${2:-default}

# 색상 출력
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Kubernetes 리소스 제거
undeploy_k8s() {
    log_info "Kubernetes 리소스 제거 중..."
    
    # Deployment 제거
    kubectl delete deployment network-collector -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete deployment network-collector-api -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete deployment network-collector-frontend -n "$NAMESPACE" --ignore-not-found=true
    
    # Service 제거
    kubectl delete service network-collector-api -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete service network-collector-frontend -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete service mariadb -n "$NAMESPACE" --ignore-not-found=true
    
    # StatefulSet 제거
    kubectl delete statefulset mariadb -n "$NAMESPACE" --ignore-not-found=true
    
    # Ingress 제거
    kubectl delete ingress network-collector-ingress -n "$NAMESPACE" --ignore-not-found=true
    
    # ConfigMap 제거
    kubectl delete configmap network-collector-config -n "$NAMESPACE" --ignore-not-found=true
    kubectl delete configmap network-collector-frontend-config -n "$NAMESPACE" --ignore-not-found=true
    
    # Secret 제거
    read -p "Secret도 제거하시겠습니까? (y/N): " DELETE_SECRET
    if [ "$DELETE_SECRET" = "y" ] || [ "$DELETE_SECRET" = "Y" ]; then
        kubectl delete secret network-collector-secrets -n "$NAMESPACE" --ignore-not-found=true
    fi
    
    # PVC 제거 (선택적)
    read -p "데이터베이스 PVC도 제거하시겠습니까? (y/N): " DELETE_PVC
    if [ "$DELETE_PVC" = "y" ] || [ "$DELETE_PVC" = "Y" ]; then
        kubectl delete pvc -l app=mariadb -n "$NAMESPACE" --ignore-not-found=true
    fi
    
    log_info "Kubernetes 리소스 제거 완료"
}

# Helm 제거
undeploy_helm() {
    log_info "Helm Chart 제거 중..."
    
    helm uninstall network-collector -n "$NAMESPACE" || {
        log_warn "Helm release를 찾을 수 없습니다."
    }
    
    # PVC 제거 (선택적)
    read -p "데이터베이스 PVC도 제거하시겠습니까? (y/N): " DELETE_PVC
    if [ "$DELETE_PVC" = "y" ] || [ "$DELETE_PVC" = "Y" ]; then
        kubectl delete pvc -l app=mariadb -n "$NAMESPACE" --ignore-not-found=true
    fi
    
    log_info "Helm Chart 제거 완료"
}

# 메인 실행
main() {
    log_warn "OpenStack Monitoring System 제거 시작"
    log_info "배포 타입: $DEPLOYMENT_TYPE"
    log_info "네임스페이스: $NAMESPACE"
    
    read -p "정말로 제거하시겠습니까? (y/N): " CONFIRM
    if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
        log_info "제거 취소됨"
        exit 0
    fi
    
    if [ "$DEPLOYMENT_TYPE" = "helm" ]; then
        undeploy_helm
    else
        undeploy_k8s
    fi
    
    log_info "제거 완료!"
}

main "$@"

