#!/bin/bash

# 모든 Kubernetes 노드에 이미지를 배포하는 스크립트
# 사용법: ./scripts/deploy-images-to-nodes.sh [node1] [node2] ...
# 
# SSH 키 설정이 필요하면 먼저 실행:
# ./scripts/setup-ssh-keys.sh [node1] [node2] ...

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

# 노드 목록 (인자로 받거나 자동 감지)
if [ $# -eq 0 ]; then
    log_info "노드 목록을 자동으로 감지합니다..."
    # kubectl을 사용하여 노드 목록 가져오기
    if command -v kubectl &> /dev/null && kubectl cluster-info &> /dev/null; then
        NODES=($(kubectl get nodes -o jsonpath='{.items[*].metadata.name}'))
        log_info "감지된 노드: ${NODES[@]}"
    else
        log_error "kubectl을 사용할 수 없거나 클러스터에 연결할 수 없습니다."
        log_info "사용법: $0 <node1> <node2> ..."
        log_info "예시: $0 k8s-master-01 k8s-worker-01"
        exit 1
    fi
else
    NODES=("$@")
fi

if [ ${#NODES[@]} -eq 0 ]; then
    log_error "노드가 지정되지 않았습니다."
    exit 1
fi

# SSH 키 확인 및 설정
SSH_KEY="$HOME/.ssh/id_rsa_network_collector"
check_ssh_connection() {
    local node=$1
    if [[ "$node" == *"@"* ]]; then
        # SSH 형식
        if ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$node" "echo 'test'" &>/dev/null; then
            return 0
        fi
    else
        # 호스트명만
        if ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$node" "echo 'test'" &>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# SSH 키가 있으면 사용, 없으면 기본 키 사용
if [ -f "$SSH_KEY" ]; then
    log_info "전용 SSH 키를 사용합니다: $SSH_KEY"
    SSH_OPTS="-i $SSH_KEY"
else
    log_warn "전용 SSH 키를 찾을 수 없습니다. 기본 SSH 키를 사용합니다."
    log_info "SSH 키를 설정하려면: ./scripts/setup-ssh-keys.sh ${NODES[@]}"
    SSH_OPTS=""
fi

# 첫 번째 노드에서 이미지 빌드
BUILD_NODE=${NODES[0]}
log_info "이미지 빌드를 시작합니다 (노드: $BUILD_NODE)..."

# 원격 노드인지 확인
if [[ "$BUILD_NODE" == *"@"* ]]; then
    # SSH 형식: user@host
    SSH_USER=$(echo $BUILD_NODE | cut -d'@' -f1)
    SSH_HOST=$(echo $BUILD_NODE | cut -d'@' -f2)
    SSH_CMD="ssh $SSH_OPTS $SSH_USER@$SSH_HOST"
    SCP_CMD="scp $SSH_OPTS"
    REMOTE_PATH="/tmp/network-collector"
else
    # 로컬 노드
    SSH_CMD=""
    SCP_CMD=""
    REMOTE_PATH="$PROJECT_ROOT"
fi

# 이미지 빌드
log_info "Collector 이미지 빌드 중..."
if [ -n "$SSH_CMD" ]; then
    $SSH_CMD "cd $REMOTE_PATH/backend && docker build -t network-collector:latest -f Dockerfile.collector ."
else
    cd "$PROJECT_ROOT/backend"
    docker build -t network-collector:latest -f Dockerfile.collector .
    cd "$PROJECT_ROOT"
fi

log_info "API 이미지 빌드 중..."
if [ -n "$SSH_CMD" ]; then
    $SSH_CMD "cd $REMOTE_PATH/backend && docker build -t network-collector-api:latest -f Dockerfile.api ."
else
    cd "$PROJECT_ROOT/backend"
    docker build -t network-collector-api:latest -f Dockerfile.api .
    cd "$PROJECT_ROOT"
fi

log_info "Frontend 이미지 빌드 중..."
if [ -n "$SSH_CMD" ]; then
    $SSH_CMD "cd $REMOTE_PATH/frontend && docker build -t network-collector-frontend:latest -f Dockerfile ."
else
    cd "$PROJECT_ROOT/frontend"
    docker build -t network-collector-frontend:latest -f Dockerfile .
    cd "$PROJECT_ROOT"
fi

# 이미지 저장
log_info "이미지를 tar 파일로 저장 중..."
if [ -n "$SSH_CMD" ]; then
    $SSH_CMD "docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o /tmp/network-collector-images.tar"
    IMAGE_TAR="/tmp/network-collector-images.tar"
else
    docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o /tmp/network-collector-images.tar
    IMAGE_TAR="/tmp/network-collector-images.tar"
fi

# 다른 노드에 이미지 배포
for node in "${NODES[@]}"; do
    if [ "$node" == "$BUILD_NODE" ]; then
        log_info "빌드 노드 ($node)는 이미 이미지가 있습니다."
        continue
    fi
    
    log_info "이미지를 $node에 배포 중..."
    
    # 노드 형식 확인
    if [[ "$node" == *"@"* ]]; then
        # SSH 형식
        NODE_USER=$(echo $node | cut -d'@' -f1)
        NODE_HOST=$(echo $node | cut -d'@' -f2)
        
        # 이미지 tar 복사
        if [ -n "$SSH_CMD" ]; then
            # 빌드 노드에서 대상 노드로 직접 복사
            $SSH_CMD "scp $SSH_OPTS $IMAGE_TAR $NODE_USER@$NODE_HOST:/tmp/"
        else
            scp $SSH_OPTS "$IMAGE_TAR" "$node:/tmp/"
        fi
        
        # 이미지 로드 (Docker 또는 containerd)
        ssh $SSH_OPTS "$node" "
            if command -v docker &> /dev/null; then
                docker load -i /tmp/network-collector-images.tar && rm /tmp/network-collector-images.tar
            elif command -v ctr &> /dev/null; then
                ctr -n k8s.io images import /tmp/network-collector-images.tar && rm /tmp/network-collector-images.tar
            else
                echo 'ERROR: Docker 또는 containerd를 찾을 수 없습니다.'
                exit 1
            fi
        "
    else
        # 호스트명만 있는 경우
        # 실제로는 원격 노드일 수 있으므로 SSH로 접속 시도
        CURRENT_USER=${SUDO_USER:-$USER}
        NODE_FULL="${CURRENT_USER}@${node}"
        
        # 로컬 노드인지 확인
        if [ "$node" == "localhost" ] || [ "$node" == "$(hostname)" ]; then
            # 실제 로컬 노드
            if [ "$node" != "$BUILD_NODE" ]; then
                if command -v docker &> /dev/null; then
                    docker load -i "$IMAGE_TAR"
                elif command -v ctr &> /dev/null; then
                    ctr -n k8s.io images import "$IMAGE_TAR"
                else
                    log_error "Docker 또는 containerd를 찾을 수 없습니다."
                    exit 1
                fi
            fi
        else
            # 원격 노드로 간주하고 SSH로 접속
            log_info "원격 노드로 접속: $NODE_FULL"
            
            # 이미지 tar 복사
            scp $SSH_OPTS "$IMAGE_TAR" "$NODE_FULL:/tmp/" || {
                log_error "이미지 tar 복사 실패: $NODE_FULL"
                exit 1
            }
            
            # 이미지 로드
            ssh $SSH_OPTS "$NODE_FULL" "
                if command -v docker &> /dev/null; then
                    docker load -i /tmp/network-collector-images.tar && rm /tmp/network-collector-images.tar
                elif command -v ctr &> /dev/null; then
                    ctr -n k8s.io images import /tmp/network-collector-images.tar && rm /tmp/network-collector-images.tar
                else
                    echo 'ERROR: Docker 또는 containerd를 찾을 수 없습니다.'
                    exit 1
                fi
            " || {
                log_error "SSH 접속 또는 이미지 로드 실패: $NODE_FULL"
                exit 1
            }
        fi
    fi
    
    log_info "$node에 이미지 배포 완료"
done

# 임시 파일 정리
if [ -z "$SSH_CMD" ]; then
    rm -f "$IMAGE_TAR"
fi

log_info "모든 노드에 이미지 배포 완료!"

# 확인
log_info "배포된 이미지 확인:"
for node in "${NODES[@]}"; do
    if [[ "$node" == *"@"* ]]; then
        log_info "$node:"
        ssh $SSH_OPTS "$node" "
            if command -v docker &> /dev/null; then
                docker images | grep network-collector || true
            elif command -v crictl &> /dev/null; then
                crictl images | grep network-collector || true
            fi
        " || true
    else
        log_info "$node:"
        if command -v docker &> /dev/null; then
            docker images | grep network-collector || true
        elif command -v crictl &> /dev/null; then
            crictl images | grep network-collector || true
        fi
    fi
done

