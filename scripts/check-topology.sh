#!/bin/bash

# 토폴로지 데이터 상태 확인 스크립트

set +e

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_section() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

# DB Pod 찾기
DB_POD=$(kubectl get pods -l app=mariadb -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -z "$DB_POD" ]; then
    log_error "MariaDB Pod를 찾을 수 없습니다"
    exit 1
fi

log_section "토폴로지 데이터 상태 확인"

# 1. 토폴로지 노드 확인
log_info "토폴로지 노드 확인"
NODE_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
    SELECT COUNT(*) FROM topology_nodes;
" openstack_monitor 2>/dev/null || echo "0")

if [ "$NODE_COUNT" -gt 0 ]; then
    log_info "토폴로지 노드: ${NODE_COUNT}개"
    
    # 노드 타입별 개수
    kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -e "
        SELECT node_type, COUNT(*) as count 
        FROM topology_nodes 
        GROUP BY node_type;
    " openstack_monitor 2>/dev/null || true
else
    log_warn "토폴로지 노드가 없습니다"
fi

# 2. 토폴로지 엣지 확인
log_info "토폴로지 엣지 확인"
EDGE_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
    SELECT COUNT(*) FROM topology_edges;
" openstack_monitor 2>/dev/null || echo "0")

if [ "$EDGE_COUNT" -gt 0 ]; then
    log_info "토폴로지 엣지: ${EDGE_COUNT}개"
else
    log_warn "토폴로지 엣지가 없습니다"
fi

# 3. 인스턴스와 포트 관계 확인
log_section "인스턴스-포트 관계 확인"

INSTANCE_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
    SELECT COUNT(*) FROM instances;
" openstack_monitor 2>/dev/null || echo "0")

PORT_WITH_DEVICE_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
    SELECT COUNT(*) FROM ports WHERE device_id IS NOT NULL;
" openstack_monitor 2>/dev/null || echo "0")

log_info "인스턴스: ${INSTANCE_COUNT}개"
log_info "DeviceID가 설정된 포트: ${PORT_WITH_DEVICE_COUNT}개"

if [ "$INSTANCE_COUNT" -gt 0 ] && [ "$PORT_WITH_DEVICE_COUNT" -eq 0 ]; then
    log_warn "인스턴스는 있지만 포트의 DeviceID가 설정되지 않았습니다"
    log_info "포트 수집 시 인스턴스가 아직 수집되지 않았을 수 있습니다"
fi

# 4. 포트의 DeviceID 샘플 확인
log_info "포트 DeviceID 샘플 (최대 5개)"
kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -e "
    SELECT 
        p.open_stack_id as port_os_id,
        p.device_id,
        p.device_owner,
        i.open_stack_id as instance_os_id,
        i.name as instance_name
    FROM ports p
    LEFT JOIN instances i ON p.device_id = i.id
    WHERE p.device_id IS NOT NULL
    LIMIT 5;
" openstack_monitor 2>/dev/null || true

# 5. Collector 로그 확인
log_section "Collector 로그 확인 (토폴로지 관련)"

COLLECTOR_POD=$(kubectl get pods -l app=network-collector -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -n "$COLLECTOR_POD" ]; then
    log_info "Collector Pod: $COLLECTOR_POD"
    
    # 토폴로지 관련 로그 확인
    TOPOLOGY_LOGS=$(kubectl logs $COLLECTOR_POD --tail=100 2>/dev/null | grep -i "topology\|BuildTopology" || echo "")
    
    if [ -n "$TOPOLOGY_LOGS" ]; then
        log_info "토폴로지 관련 로그:"
        echo "$TOPOLOGY_LOGS" | tail -20
    else
        log_warn "토폴로지 관련 로그를 찾을 수 없습니다"
    fi
    
    # 에러 로그 확인
    ERROR_LOGS=$(kubectl logs $COLLECTOR_POD --tail=100 2>/dev/null | grep -i "error\|failed\|fail" | grep -i "topology\|port\|device" || echo "")
    
    if [ -n "$ERROR_LOGS" ]; then
        log_warn "토폴로지 관련 에러 로그:"
        echo "$ERROR_LOGS" | tail -10
    fi
else
    log_warn "Collector Pod를 찾을 수 없습니다"
fi

# 6. 토폴로지 생성 테스트
log_section "토폴로지 생성 가능 여부 확인"

# 인스턴스가 있고 포트가 있는 경우
INSTANCE_WITH_PORTS=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
    SELECT COUNT(DISTINCT i.id)
    FROM instances i
    INNER JOIN ports p ON p.device_id = i.id
    WHERE p.device_id IS NOT NULL;
" openstack_monitor 2>/dev/null || echo "0")

if [ "$INSTANCE_WITH_PORTS" -gt 0 ]; then
    log_info "토폴로지를 생성할 수 있는 인스턴스: ${INSTANCE_WITH_PORTS}개"
    
    # 샘플 인스턴스 ID
    SAMPLE_INSTANCE=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT i.id
        FROM instances i
        INNER JOIN ports p ON p.device_id = i.id
        WHERE p.device_id IS NOT NULL
        LIMIT 1;
    " openstack_monitor 2>/dev/null || echo "")
    
    if [ -n "$SAMPLE_INSTANCE" ]; then
        log_info "샘플 인스턴스 ID: ${SAMPLE_INSTANCE:0:8}..."
        
        # 이 인스턴스에 대한 토폴로지 노드가 있는지 확인
        TOPOLOGY_NODE_EXISTS=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
            SELECT COUNT(*) 
            FROM topology_nodes 
            WHERE instance_id = '$SAMPLE_INSTANCE';
        " openstack_monitor 2>/dev/null || echo "0")
        
        if [ "$TOPOLOGY_NODE_EXISTS" -eq 0 ]; then
            log_warn "인스턴스 ${SAMPLE_INSTANCE:0:8}...에 대한 토폴로지 노드가 없습니다"
            log_info "Collector가 토폴로지를 생성하지 않았거나 실패했습니다"
        else
            log_info "인스턴스 ${SAMPLE_INSTANCE:0:8}...에 대한 토폴로지 노드가 있습니다"
        fi
    fi
else
    log_warn "토폴로지를 생성할 수 있는 인스턴스가 없습니다 (포트의 DeviceID가 설정되지 않음)"
fi

# 7. 권장 사항
log_section "권장 사항"

if [ "$NODE_COUNT" -eq 0 ]; then
    log_info "토폴로지 데이터가 없습니다. 다음을 확인하세요:"
    echo "  1. Collector가 최근에 실행되었는지 확인"
    echo "  2. Collector 로그에서 토폴로지 생성 에러 확인"
    echo "  3. 포트의 DeviceID가 올바르게 설정되었는지 확인"
    echo "  4. 수집 순서: Instances -> Ports -> Topology"
    echo ""
    echo "  Collector 재시작: kubectl rollout restart deployment/network-collector"
fi

if [ "$PORT_WITH_DEVICE_COUNT" -eq 0 ] && [ "$INSTANCE_COUNT" -gt 0 ]; then
    log_info "포트의 DeviceID가 설정되지 않았습니다."
    echo "  원인: Port 수집 시 인스턴스가 아직 수집되지 않았을 수 있습니다"
    echo "  해결: Collector를 다시 실행하여 포트를 재수집하세요"
fi

