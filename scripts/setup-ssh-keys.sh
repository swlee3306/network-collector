#!/bin/bash

# Kubernetes 노드들에 SSH 키를 배포하는 스크립트
# 사용법: ./scripts/setup-ssh-keys.sh [node1] [node2] ...
# 예시: ./scripts/setup-ssh-keys.sh k8s-master-01 k8s-worker-01 k8s-worker-02

set -e

# 스크립트가 있는 디렉토리로 이동 (프로젝트 루트)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# SSH 키 생성 함수
generate_ssh_key() {
    local key_path="$HOME/.ssh/id_rsa_network_collector"
    
    if [ -f "$key_path" ]; then
        log_info "SSH 키가 이미 존재합니다: $key_path"
        read -p "기존 키를 덮어쓰시겠습니까? (y/N): " overwrite
        if [ "$overwrite" != "y" ] && [ "$overwrite" != "Y" ]; then
            log_info "기존 키를 사용합니다."
            return 0
        fi
    fi
    
    log_step "SSH 키 생성 중..."
    ssh-keygen -t rsa -b 4096 -f "$key_path" -N "" -C "network-collector-deploy"
    log_info "SSH 키 생성 완료: $key_path"
    
    # SSH config에 추가
    local ssh_config="$HOME/.ssh/config"
    if [ ! -f "$ssh_config" ]; then
        touch "$ssh_config"
        chmod 600 "$ssh_config"
    fi
    
    log_info "SSH 키 경로: $key_path"
    echo "다음 명령어로 공개키를 확인할 수 있습니다:"
    echo "  cat ${key_path}.pub"
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

# SSH 키 경로
SSH_KEY="$HOME/.ssh/id_rsa_network_collector"
SSH_PUB_KEY="${SSH_KEY}.pub"

# SSH 키가 없으면 생성
if [ ! -f "$SSH_KEY" ]; then
    log_warn "SSH 키가 없습니다. 생성합니다..."
    generate_ssh_key
else
    log_info "SSH 키를 찾았습니다: $SSH_KEY"
fi

# 공개키 읽기
if [ ! -f "$SSH_PUB_KEY" ]; then
    log_error "공개키를 찾을 수 없습니다: $SSH_PUB_KEY"
    exit 1
fi

PUBLIC_KEY=$(cat "$SSH_PUB_KEY")
log_info "공개키 내용:"
echo "$PUBLIC_KEY"
echo

# 각 노드에 SSH 키 배포
log_step "각 노드에 SSH 키 배포 중..."
for node in "${NODES[@]}"; do
    log_info "노드 처리 중: $node"
    
    # 노드 형식 확인
    if [[ "$node" == *"@"* ]]; then
        # SSH 형식: user@host
        NODE_USER=$(echo $node | cut -d'@' -f1)
        NODE_HOST=$(echo $node | cut -d'@' -f2)
        
        log_info "  사용자: $NODE_USER, 호스트: $NODE_HOST"
        
        # SSH 연결 테스트
        log_info "  SSH 연결 테스트 중..."
        if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$node" "echo 'Connection successful'" 2>/dev/null; then
            log_info "  ✓ SSH 연결 성공"
        else
            log_warn "  SSH 연결 실패. 비밀번호를 입력해야 할 수 있습니다."
        fi
        
        # authorized_keys에 공개키 추가
        log_info "  공개키 배포 중..."
        ssh "$node" "
            mkdir -p ~/.ssh
            chmod 700 ~/.ssh
            if ! grep -q '${PUBLIC_KEY}' ~/.ssh/authorized_keys 2>/dev/null; then
                echo '${PUBLIC_KEY}' >> ~/.ssh/authorized_keys
                chmod 600 ~/.ssh/authorized_keys
                echo '공개키가 추가되었습니다.'
            else
                echo '공개키가 이미 존재합니다.'
            fi
        " || {
            log_error "  공개키 배포 실패: $node"
            log_warn "  수동으로 배포하세요:"
            echo "    ssh-copy-id -i $SSH_PUB_KEY $node"
            continue
        }
        
        log_info "  ✓ 공개키 배포 완료: $node"
        
    else
        # 호스트명만 있는 경우
        log_warn "  호스트명만 제공되었습니다: $node"
        log_info "  사용자명을 입력하세요 (엔터 시 현재 사용자: $USER):"
        read -p "  사용자명: " NODE_USER
        NODE_USER=${NODE_USER:-$USER}
        
        NODE_FULL="${NODE_USER}@${node}"
        log_info "  전체 주소: $NODE_FULL"
        
        # SSH 연결 테스트
        if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$NODE_FULL" "echo 'Connection successful'" 2>/dev/null; then
            log_info "  ✓ SSH 연결 성공"
        else
            log_warn "  SSH 연결 실패. 비밀번호를 입력해야 할 수 있습니다."
        fi
        
        # authorized_keys에 공개키 추가
        log_info "  공개키 배포 중..."
        ssh "$NODE_FULL" "
            mkdir -p ~/.ssh
            chmod 700 ~/.ssh
            if ! grep -q '${PUBLIC_KEY}' ~/.ssh/authorized_keys 2>/dev/null; then
                echo '${PUBLIC_KEY}' >> ~/.ssh/authorized_keys
                chmod 600 ~/.ssh/authorized_keys
                echo '공개키가 추가되었습니다.'
            else
                echo '공개키가 이미 존재합니다.'
            fi
        " || {
            log_error "  공개키 배포 실패: $NODE_FULL"
            log_warn "  수동으로 배포하세요:"
            echo "    ssh-copy-id -i $SSH_PUB_KEY $NODE_FULL"
            continue
        }
        
        log_info "  ✓ 공개키 배포 완료: $NODE_FULL"
    fi
    
    # SSH 키로 비밀번호 없이 접속 테스트
    log_info "  비밀번호 없이 접속 테스트 중..."
    if ssh -i "$SSH_KEY" -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$node" "echo 'Passwordless SSH successful'" 2>/dev/null; then
        log_info "  ✓ 비밀번호 없이 접속 성공!"
    else
        log_warn "  ⚠ 비밀번호 없이 접속 실패. SSH 키 인증이 제대로 설정되지 않았을 수 있습니다."
    fi
    
    echo
done

log_info "SSH 키 설정 완료!"
echo
log_info "다음 단계:"
echo "  1. deploy-images-to-nodes.sh 스크립트를 실행하세요"
echo "  2. 또는 SSH 키를 사용하여 수동으로 접속 테스트:"
echo "     ssh -i $SSH_KEY <node>"
echo
log_info "SSH 키 위치:"
echo "  개인키: $SSH_KEY"
echo "  공개키: $SSH_PUB_KEY"

