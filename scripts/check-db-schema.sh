#!/bin/bash

# 데이터베이스 스키마 확인 스크립트

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

log_info "데이터베이스 스키마 확인 중..."

# MariaDB Pod 이름
DB_POD=$(kubectl get pods -l app=mariadb -o jsonpath='{.items[0].metadata.name}')

if [ -z "$DB_POD" ]; then
    log_error "MariaDB Pod를 찾을 수 없습니다."
    exit 1
fi

log_info "DB Pod: $DB_POD"

# ports.device_id 컬럼 확인
log_info "=== ports.device_id 컬럼 정보 ==="
kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -e "
SELECT 
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND TABLE_NAME = 'ports'
AND COLUMN_NAME = 'device_id';
" 2>/dev/null

# instances.hypervisor_id 컬럼 확인
log_info "=== instances.hypervisor_id 컬럼 정보 ==="
kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -e "
SELECT 
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH,
    IS_NULLABLE,
    COLUMN_DEFAULT
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND TABLE_NAME = 'instances'
AND COLUMN_NAME = 'hypervisor_id';
" 2>/dev/null

# 외래 키 제약 확인
log_info "=== 외래 키 제약 확인 ==="
kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -e "
SELECT 
    TABLE_NAME,
    CONSTRAINT_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND TABLE_NAME IN ('ports', 'instances')
AND ((TABLE_NAME = 'ports' AND COLUMN_NAME = 'device_id')
     OR (TABLE_NAME = 'instances' AND COLUMN_NAME = 'hypervisor_id'))
AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME;
" 2>/dev/null || log_info "외래 키 제약 없음"

