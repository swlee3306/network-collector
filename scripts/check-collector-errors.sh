#!/bin/bash

# Collector의 상세 에러 로그를 확인하는 스크립트

set -e

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_info "Collector Pod의 상세 에러 로그 확인 중..."

# Collector Pod 이름
COLLECTOR_POD=$(kubectl get pods -l app=network-collector -o jsonpath='{.items[0].metadata.name}')

if [ -z "$COLLECTOR_POD" ]; then
    log_error "Collector Pod를 찾을 수 없습니다."
    exit 1
fi

log_info "Pod: $COLLECTOR_POD"
echo

# 최근 100줄 로그에서 에러만 추출
log_info "=== 최근 에러 로그 (100줄) ==="
kubectl logs $COLLECTOR_POD --tail=100 2>/dev/null | grep -i "error\|failed\|fail" || echo "에러 없음"

echo
log_info "=== 상세 에러 메시지 ==="
kubectl logs $COLLECTOR_POD --tail=200 2>/dev/null | grep -E "error|failed|fail" -A 2 -B 2 || echo "상세 에러 없음"

echo
log_info "=== OpenStack 연결 테스트 ==="
# Collector Pod에서 OpenStack 인증 정보 확인
kubectl exec $COLLECTOR_POD -- env | grep OPENSTACK || echo "OpenStack 환경 변수 없음"

echo
log_info "=== Keystone URL 접근 테스트 ==="
KEYSTONE_URL=$(kubectl exec $COLLECTOR_POD -- env | grep OPENSTACK_AUTH_URL | cut -d'=' -f2)
if [ -n "$KEYSTONE_URL" ]; then
    log_info "Keystone URL: $KEYSTONE_URL"
    # Pod 내부에서 curl 테스트 (curl이 있다면)
    kubectl exec $COLLECTOR_POD -- sh -c "command -v curl >/dev/null && curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' $KEYSTONE_URL || echo 'curl이 없거나 접근 실패'" 2>/dev/null || echo "접근 테스트 실패"
else
    log_error "OPENSTACK_AUTH_URL이 설정되지 않았습니다."
fi

echo
log_info "=== 전체 로그 (최근 50줄) ==="
kubectl logs $COLLECTOR_POD --tail=50 2>/dev/null

