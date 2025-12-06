#!/bin/bash

# DB 스키마 수정 후 Collector 재시작 및 테스트 스크립트

set +e

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_section() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

log_section "Collector 재시작 및 테스트"

# 1. Collector 재시작
log_info "Collector Pod 재시작 중..."
kubectl rollout restart deployment/network-collector

log_info "재시작 완료 대기 중..."
kubectl rollout status deployment/network-collector --timeout=60s || log_warn "재시작 상태 확인 시간 초과"

# 2. 잠시 대기 (Pod가 완전히 시작될 때까지)
log_info "Pod 시작 대기 중 (10초)..."
sleep 10

# 3. Collector 로그 확인
log_section "Collector 로그 확인 (최근 30줄)"
kubectl logs deployment/network-collector --tail=30 | tail -20

# 4. 데이터 확인
log_section "데이터 수집 확인"
DB_POD=$(kubectl get pods -l app=mariadb -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$DB_POD" ]; then
    log_info "데이터베이스 데이터 확인..."
    kubectl exec $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -e "
        SELECT 
            'Instances' AS resource, COUNT(*) AS count FROM instances
        UNION ALL
        SELECT 'Networks', COUNT(*) FROM networks
        UNION ALL
        SELECT 'Projects', COUNT(*) FROM projects
        UNION ALL
        SELECT 'Ports', COUNT(*) FROM ports
        UNION ALL
        SELECT 'Routers', COUNT(*) FROM routers
        UNION ALL
        SELECT 'Hypervisors', COUNT(*) FROM hypervisors
        UNION ALL
        SELECT 'Flavors', COUNT(*) FROM flavors;
    " 2>/dev/null || log_warn "데이터 확인 실패"
fi

# 5. 종합 테스트 실행
log_section "종합 테스트 실행"
./test-all.sh

