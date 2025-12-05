#!/bin/bash

# OpenStack Monitoring System 배포 스크립트
# 사용법: ./scripts/deploy.sh [k8s|helm]

set -e

DEPLOYMENT_TYPE=${1:-k8s}
NAMESPACE=${2:-default}
IMAGE_REGISTRY=${IMAGE_REGISTRY:-""}
IMAGE_TAG=${IMAGE_TAG:-latest}

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

# 전제 조건 확인
check_prerequisites() {
    log_info "전제 조건 확인 중..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl이 설치되어 있지 않습니다."
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Kubernetes 클러스터에 연결할 수 없습니다."
        exit 1
    fi
    
    if [ "$DEPLOYMENT_TYPE" = "helm" ] && ! command -v helm &> /dev/null; then
        log_error "Helm이 설치되어 있지 않습니다."
        exit 1
    fi
    
    log_info "전제 조건 확인 완료"
}

# Secret 값 입력 받기
get_secrets() {
    log_info "Secret 값 입력 중..."
    
    read -sp "DB_PASSWORD: " DB_PASSWORD
    echo
    read -p "OPENSTACK_AUTH_URL [http://keystone:5000/v3]: " OPENSTACK_AUTH_URL
    OPENSTACK_AUTH_URL=${OPENSTACK_AUTH_URL:-http://keystone:5000/v3}
    read -p "OPENSTACK_USERNAME [admin]: " OPENSTACK_USERNAME
    OPENSTACK_USERNAME=${OPENSTACK_USERNAME:-admin}
    read -sp "OPENSTACK_PASSWORD: " OPENSTACK_PASSWORD
    echo
    read -p "OPENSTACK_PROJECT_ID: " OPENSTACK_PROJECT_ID
    read -sp "JWT_SECRET: " JWT_SECRET
    echo
    read -sp "AUTH_TOKEN: " AUTH_TOKEN
    echo
    
    if [ -z "$DB_PASSWORD" ] || [ -z "$OPENSTACK_PASSWORD" ] || [ -z "$OPENSTACK_PROJECT_ID" ] || [ -z "$JWT_SECRET" ] || [ -z "$AUTH_TOKEN" ]; then
        log_error "필수 Secret 값이 입력되지 않았습니다."
        exit 1
    fi
}

# Kubernetes 배포
deploy_k8s() {
    log_info "Kubernetes 매니페스트로 배포 중..."
    
    # Secret 생성
    log_info "Secret 생성 중..."
    kubectl create secret generic network-collector-secrets \
        --from-literal=DB_PASSWORD="$DB_PASSWORD" \
        --from-literal=OPENSTACK_AUTH_URL="$OPENSTACK_AUTH_URL" \
        --from-literal=OPENSTACK_USERNAME="$OPENSTACK_USERNAME" \
        --from-literal=OPENSTACK_PASSWORD="$OPENSTACK_PASSWORD" \
        --from-literal=OPENSTACK_PROJECT_ID="$OPENSTACK_PROJECT_ID" \
        --from-literal=JWT_SECRET="$JWT_SECRET" \
        --from-literal=AUTH_TOKEN="$AUTH_TOKEN" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # ConfigMap 생성
    log_info "ConfigMap 생성 중..."
    kubectl apply -f backend/deployments/k8s/configmap.yaml -n "$NAMESPACE"
    
    # MariaDB 배포
    log_info "MariaDB 배포 중..."
    kubectl apply -f backend/deployments/k8s/mariadb/statefulset.yaml -n "$NAMESPACE"
    
    # MariaDB 준비 대기
    log_info "MariaDB 준비 대기 중..."
    kubectl wait --for=condition=ready pod -l app=mariadb -n "$NAMESPACE" --timeout=300s || {
        log_warn "MariaDB 준비 시간 초과. 계속 진행합니다..."
    }
    
    # Collector 배포
    log_info "Collector 배포 중..."
    kubectl apply -f backend/deployments/k8s/collector/deployment.yaml -n "$NAMESPACE"
    
    # API 배포
    log_info "API Service 배포 중..."
    kubectl apply -f backend/deployments/k8s/api/deployment.yaml -n "$NAMESPACE"
    
    # Frontend 배포
    log_info "Frontend 배포 중..."
    kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml -n "$NAMESPACE"
    
    # Ingress 배포 (선택적)
    read -p "Ingress를 배포하시겠습니까? (y/N): " DEPLOY_INGRESS
    if [ "$DEPLOY_INGRESS" = "y" ] || [ "$DEPLOY_INGRESS" = "Y" ]; then
        log_info "Ingress 배포 중..."
        kubectl apply -f backend/deployments/k8s/ingress.yaml -n "$NAMESPACE"
    fi
    
    log_info "Kubernetes 배포 완료"
}

# Helm 배포
deploy_helm() {
    log_info "Helm Chart로 배포 중..."
    
    cd backend/deployments/helm
    
    helm upgrade --install network-collector . \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --set secrets.dbPassword="$DB_PASSWORD" \
        --set secrets.openstackAuthURL="$OPENSTACK_AUTH_URL" \
        --set secrets.openstackUsername="$OPENSTACK_USERNAME" \
        --set secrets.openstackPassword="$OPENSTACK_PASSWORD" \
        --set secrets.openstackProjectID="$OPENSTACK_PROJECT_ID" \
        --set secrets.jwtSecret="$JWT_SECRET" \
        --set secrets.authToken="$AUTH_TOKEN" \
        --set collector.image.repository="${IMAGE_REGISTRY}network-collector" \
        --set collector.image.tag="$IMAGE_TAG" \
        --set api.image.repository="${IMAGE_REGISTRY}network-collector-api" \
        --set api.image.tag="$IMAGE_TAG" \
        --set frontend.image.repository="${IMAGE_REGISTRY}network-collector-frontend" \
        --set frontend.image.tag="$IMAGE_TAG"
    
    cd - > /dev/null
    
    log_info "Helm 배포 완료"
}

# 배포 상태 확인
check_status() {
    log_info "배포 상태 확인 중..."
    
    echo
    echo "=== Pod 상태 ==="
    kubectl get pods -n "$NAMESPACE" | grep -E "network-collector|mariadb"
    
    echo
    echo "=== 서비스 상태 ==="
    kubectl get services -n "$NAMESPACE" | grep -E "network-collector|mariadb"
    
    echo
    log_info "배포 상태 확인 완료"
}

# 메인 실행
main() {
    log_info "OpenStack Monitoring System 배포 시작"
    log_info "배포 타입: $DEPLOYMENT_TYPE"
    log_info "네임스페이스: $NAMESPACE"
    
    check_prerequisites
    get_secrets
    
    if [ "$DEPLOYMENT_TYPE" = "helm" ]; then
        deploy_helm
    else
        deploy_k8s
    fi
    
    check_status
    
    log_info "배포 완료!"
    log_info "상태 확인: kubectl get pods -n $NAMESPACE"
}

main "$@"

