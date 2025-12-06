#!/bin/bash

# 원격 서버에 SSH로 접속하여 로그를 분석하는 스크립트
# 사용법: ./scripts/remote-analyze.sh [user@host:port]

set -e

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

# 기본값
SSH_USER="sulee"
SSH_HOST="192.168.219.204"
SSH_PORT="2233"

# 인자 파싱
if [ $# -gt 0 ]; then
    SSH_TARGET="$1"
    if [[ "$SSH_TARGET" == *"@"* ]]; then
        # user@host 형식
        if [[ "$SSH_TARGET" == *":"* ]]; then
            # user@host:port 형식
            SSH_USER=$(echo $SSH_TARGET | cut -d'@' -f1)
            SSH_HOST_PORT=$(echo $SSH_TARGET | cut -d'@' -f2)
            SSH_HOST=$(echo $SSH_HOST_PORT | cut -d':' -f1)
            SSH_PORT=$(echo $SSH_HOST_PORT | cut -d':' -f2)
        else
            SSH_USER=$(echo $SSH_TARGET | cut -d'@' -f1)
            SSH_HOST=$(echo $SSH_TARGET | cut -d'@' -f2)
        fi
    else
        SSH_HOST="$SSH_TARGET"
    fi
fi

SSH_TARGET_FULL="${SSH_USER}@${SSH_HOST}"

log_info "원격 서버에 접속합니다: $SSH_TARGET_FULL (포트: $SSH_PORT)"

# SSH 키 확인
SSH_KEY="$HOME/.ssh/id_rsa_network_collector"
if [ -f "$SSH_KEY" ]; then
    SSH_CMD="ssh -i $SSH_KEY -p $SSH_PORT"
    log_info "SSH 키 사용: $SSH_KEY"
else
    SSH_CMD="ssh -p $SSH_PORT"
    log_warn "SSH 키를 찾을 수 없습니다. 비밀번호 입력이 필요할 수 있습니다."
fi

# 스크립트를 원격 서버에 복사하고 실행
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

log_info "분석 스크립트를 원격 서버에 복사 중..."
$SSH_CMD "$SSH_TARGET_FULL" "mkdir -p /tmp/network-collector-scripts" || true

# analyze-logs.sh를 원격에 복사
scp -P $SSH_PORT ${SSH_KEY:+-i $SSH_KEY} "$SCRIPT_DIR/analyze-logs.sh" "$SSH_TARGET_FULL:/tmp/network-collector-scripts/" || {
    log_error "스크립트 복사 실패"
    exit 1
}

log_info "원격 서버에서 로그 분석 실행 중..."
echo
echo "=========================================="
echo "원격 서버 로그 분석 결과"
echo "=========================================="
echo

# 원격에서 스크립트 실행
$SSH_CMD "$SSH_TARGET_FULL" "chmod +x /tmp/network-collector-scripts/analyze-logs.sh && /tmp/network-collector-scripts/analyze-logs.sh"

echo
log_info "분석 완료!"

