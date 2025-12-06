#!/bin/bash

# 데이터베이스 스키마 수정 스크립트

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

# 마이그레이션 SQL 실행
log_info "Port 테이블의 device_id 컬럼 수정 중..."

kubectl exec -i $DB_POD -- mysql -u openstack_monitor -proot openstack_monitor <<EOF
-- Drop foreign key constraint if exists
SET @constraint_name = (SELECT CONSTRAINT_NAME FROM information_schema.TABLE_CONSTRAINTS WHERE TABLE_SCHEMA = 'openstack_monitor' AND TABLE_NAME = 'ports' AND CONSTRAINT_TYPE = 'FOREIGN KEY' AND CONSTRAINT_NAME LIKE '%device_id%' LIMIT 1);
SET @sql = IF(@constraint_name IS NOT NULL, CONCAT('ALTER TABLE ports DROP FOREIGN KEY ', @constraint_name), 'SELECT "No foreign key constraint found"');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Modify device_id column to allow NULL and increase length
ALTER TABLE ports MODIFY COLUMN device_id VARCHAR(255) NULL;

-- Also fix hypervisor_id in instances table
ALTER TABLE instances MODIFY COLUMN hypervisor_id VARCHAR(255) NULL;

SELECT 'Schema update completed successfully' AS result;
EOF

if [ $? -eq 0 ]; then
    log_info "스키마 수정 완료!"
else
    log_error "스키마 수정 실패"
    exit 1
fi

