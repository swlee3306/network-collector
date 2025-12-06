#!/bin/bash

# 데이터베이스 스키마 수정 스크립트
# 이 스크립트는 다음을 수행합니다:
# 1. ports.device_id 컬럼을 VARCHAR(255) NULL로 수정 (외래 키 제약 제거)
# 2. instances.hypervisor_id 컬럼을 VARCHAR(255) NULL로 수정 (외래 키 제약 제거)

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

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_info "데이터베이스 스키마 수정 중..."

# MariaDB Pod 이름
DB_POD=$(kubectl get pods -l app=mariadb -o jsonpath='{.items[0].metadata.name}')

if [ -z "$DB_POD" ]; then
    log_error "MariaDB Pod를 찾을 수 없습니다."
    exit 1
fi

log_info "DB Pod: $DB_POD"

# 먼저 현재 외래 키 제약 확인
log_info "현재 외래 키 제약 확인 중..."
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
AND REFERENCED_TABLE_NAME IS NOT NULL
ORDER BY TABLE_NAME, CONSTRAINT_NAME;
" 2>/dev/null || log_warn "외래 키 제약 확인 중 오류 발생 (계속 진행)"

# ports.device_id에 대한 외래 키 제약 찾기 및 제거
log_info "ports.device_id 외래 키 제약 제거 중..."
PORTS_FK=$(kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -N -e "
SELECT CONSTRAINT_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND TABLE_NAME = 'ports'
AND COLUMN_NAME = 'device_id'
AND REFERENCED_TABLE_NAME IS NOT NULL
LIMIT 1;
" 2>/dev/null || echo "")

if [ -n "$PORTS_FK" ]; then
    log_info "제거할 제약: $PORTS_FK"
    kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor <<EOF
SET FOREIGN_KEY_CHECKS = 0;
ALTER TABLE ports DROP FOREIGN KEY $PORTS_FK;
SET FOREIGN_KEY_CHECKS = 1;
EOF
else
    log_warn "ports.device_id에 대한 외래 키 제약을 찾을 수 없습니다."
fi

# instances.hypervisor_id에 대한 외래 키 제약 찾기 및 제거
log_info "instances.hypervisor_id 외래 키 제약 제거 중..."
INSTANCES_FK=$(kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -N -e "
SELECT CONSTRAINT_NAME
FROM information_schema.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND TABLE_NAME = 'instances'
AND COLUMN_NAME = 'hypervisor_id'
AND REFERENCED_TABLE_NAME IS NOT NULL
LIMIT 1;
" 2>/dev/null || echo "")

if [ -n "$INSTANCES_FK" ]; then
    log_info "제거할 제약: $INSTANCES_FK"
    kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor <<EOF
SET FOREIGN_KEY_CHECKS = 0;
ALTER TABLE instances DROP FOREIGN KEY $INSTANCES_FK;
SET FOREIGN_KEY_CHECKS = 1;
EOF
else
    log_warn "instances.hypervisor_id에 대한 외래 키 제약을 찾을 수 없습니다."
fi

# 마이그레이션 SQL 실행
log_info "컬럼 타입 수정 중..."

kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor <<'EOF'
SET FOREIGN_KEY_CHECKS = 0;

-- ports.device_id 컬럼 수정
ALTER TABLE ports MODIFY COLUMN device_id VARCHAR(255) NULL;

-- instances.hypervisor_id 컬럼 수정
ALTER TABLE instances MODIFY COLUMN hypervisor_id VARCHAR(255) NULL;

SET FOREIGN_KEY_CHECKS = 1;

SELECT 'Schema update completed successfully' AS result;
EOF

if [ $? -eq 0 ]; then
    log_info "스키마 수정 완료!"
    
    # 수정 후 확인
    log_info "수정된 스키마 확인 중..."
    kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor -e "
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH,
    IS_NULLABLE
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = 'openstack_monitor'
AND ((TABLE_NAME = 'ports' AND COLUMN_NAME = 'device_id')
     OR (TABLE_NAME = 'instances' AND COLUMN_NAME = 'hypervisor_id'))
ORDER BY TABLE_NAME, COLUMN_NAME;
" 2>/dev/null || log_warn "스키마 확인 중 오류 발생"
else
    log_error "스키마 수정 실패"
    exit 1
fi
