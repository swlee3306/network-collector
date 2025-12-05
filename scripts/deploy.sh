#!/bin/bash

# OpenStack Monitoring System 배포 스크립트
# 사용법: ./scripts/deploy.sh [k8s|helm]

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

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
    
    cd "$PROJECT_ROOT/backend/deployments/helm"
    
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
    
    cd "$PROJECT_ROOT"
    
    log_info "Helm 배포 완료"
}

# 이미지 확인 (더 정확한 방법)
check_images() {
    log_info "이미지 확인 중..."
    
    IMAGES=("network-collector:latest" "network-collector-api:latest" "network-collector-frontend:latest")
    MISSING_IMAGES=()
    
    if command -v docker &> /dev/null; then
        for image in "${IMAGES[@]}"; do
            # docker image inspect를 사용하여 더 정확하게 확인
            if docker image inspect "$image" &> /dev/null; then
                log_info "이미지 확인됨: $image"
            else
                MISSING_IMAGES+=("$image")
                log_warn "이미지가 없습니다: $image"
            fi
        done
        
        if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
            log_error "다음 이미지들이 없습니다:"
            for img in "${MISSING_IMAGES[@]}"; do
                echo "  - $img"
            done
            log_error "먼저 './scripts/build-images.sh'를 실행하여 이미지를 빌드하세요."
            return 1
        fi
        
        # Kubernetes 컨테이너 런타임 확인 및 이미지 import
        log_info "Kubernetes 컨테이너 런타임 확인 중..."
        if command -v crictl &> /dev/null; then
            log_info "containerd/cri-o 감지됨. 이미지를 컨테이너 런타임에 import합니다..."
            for image in "${IMAGES[@]}"; do
                # Docker 이미지를 containerd로 import
                if command -v ctr &> /dev/null; then
                    log_info "이미지 import 중: $image"
                    docker save "$image" | ctr -n k8s.io images import - || {
                        log_warn "ctr로 import 실패, crictl로 시도..."
                        # crictl은 직접 import를 지원하지 않으므로 docker save/load 사용
                    }
                fi
            done
        fi
    else
        log_warn "Docker가 설치되어 있지 않습니다. 이미지 확인을 건너뜁니다."
    fi
    
    return 0
}

# 배포 상태 확인
check_status() {
    log_info "배포 상태 확인 중..."
    
    echo
    echo "=== Pod 상태 ==="
    kubectl get pods -n "$NAMESPACE" -o wide | grep -E "network-collector|mariadb" || true
    
    echo
    echo "=== 서비스 상태 ==="
    kubectl get services -n "$NAMESPACE" | grep -E "network-collector|mariadb" || true
    
    echo
    log_info "Pod 상세 정보 확인 중..."
    PODS=$(kubectl get pods -n "$NAMESPACE" -l 'app in (network-collector,network-collector-api,network-collector-frontend)' -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")
    
    if [ -n "$PODS" ]; then
        for pod in $PODS; do
            STATUS=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
            NODE=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.nodeName}' 2>/dev/null || echo "Unknown")
            echo "  Pod: $pod | 상태: $STATUS | 노드: $NODE"
            
            if [ "$STATUS" != "Running" ]; then
                log_warn "  Pod '$pod'가 실행 중이 아닙니다. 이벤트 확인:"
                kubectl describe pod "$pod" -n "$NAMESPACE" | grep -A 5 "Events:" | head -10 || true
            fi
        done
    fi
    
    echo
    log_info "배포 상태 확인 완료"
    log_info "상세 점검: ./scripts/check-deployment.sh"
}

# 메인 실행
main() {
    log_info "OpenStack Monitoring System 배포 시작"
    log_info "배포 타입: $DEPLOYMENT_TYPE"
    log_info "네임스페이스: $NAMESPACE"
    
    check_prerequisites
    
    # 이미지 확인 (Kubernetes 배포 시에만)
    if [ "$DEPLOYMENT_TYPE" = "k8s" ]; then
        if ! check_images; then
            log_error "이미지 확인 실패. 배포를 중단합니다."
            exit 1
        fi
    fi
    
    get_secrets
    
    if [ "$DEPLOYMENT_TYPE" = "helm" ]; then
        deploy_helm
    else
        deploy_k8s
    fi
    
    # 배포 후 잠시 대기
    log_info "Pod 생성 대기 중 (10초)..."
    sleep 10
    
    check_status
    
    log_info "배포 완료!"
    log_info "상태 확인: kubectl get pods -n $NAMESPACE"
    log_info "상세 점검: ./scripts/check-deployment.sh"
}

main "$@"

