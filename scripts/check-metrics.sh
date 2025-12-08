#!/bin/bash

# 메트릭 수집 기능 확인 스크립트
# 이 스크립트는 인스턴스, 네트워크, 하이퍼바이저 메트릭이 정상적으로 수집되고 있는지 확인합니다.

set -e

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "메트릭 수집 기능 확인"
echo "=========================================="
echo ""

# 데이터베이스 연결 정보 확인
DB_HOST="${DB_HOST:-mariadb}"
DB_PORT="${DB_PORT:-3306}"
DB_USER="${DB_USER:-openstack_monitor}"
DB_NAME="${DB_NAME:-openstack_monitor}"
DB_PASSWORD="${DB_PASSWORD:-}"

if [ -z "$DB_PASSWORD" ]; then
    echo -e "${YELLOW}[WARN] DB_PASSWORD가 설정되지 않았습니다. 환경 변수를 확인하세요.${NC}"
    echo "Kubernetes Secret에서 비밀번호를 가져오는 중..."
    
    # Kubernetes Secret에서 비밀번호 가져오기
    if command -v kubectl &> /dev/null; then
        DB_PASSWORD=$(kubectl get secret network-collector-secrets -o jsonpath='{.data.DB_PASSWORD}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
        if [ -z "$DB_PASSWORD" ]; then
            echo -e "${RED}[ERROR] DB_PASSWORD를 가져올 수 없습니다.${NC}"
            echo "환경 변수 DB_PASSWORD를 설정하거나 Kubernetes Secret을 확인하세요."
            exit 1
        fi
    else
        echo -e "${RED}[ERROR] kubectl이 설치되어 있지 않습니다.${NC}"
        exit 1
    fi
fi

# MySQL 클라이언트 확인
if ! command -v mysql &> /dev/null; then
    echo -e "${YELLOW}[WARN] mysql 클라이언트가 설치되어 있지 않습니다.${NC}"
    echo "Docker 컨테이너를 사용하여 확인합니다..."
    
    # Docker를 사용하여 확인
    if command -v docker &> /dev/null; then
        MYSQL_CMD="docker run --rm -i --network host mysql:8.0 mysql"
    elif command -v kubectl &> /dev/null; then
        # Kubernetes Pod를 사용하여 확인
        echo "Kubernetes Pod를 사용하여 데이터베이스에 연결합니다..."
        MYSQL_CMD="kubectl run mysql-client --rm -i --restart=Never --image=mysql:8.0 -- mysql"
    else
        echo -e "${RED}[ERROR] mysql 클라이언트, docker, 또는 kubectl이 필요합니다.${NC}"
        exit 1
    fi
else
    MYSQL_CMD="mysql"
fi

echo ""
echo "=========================================="
echo "1. 메트릭 테이블 확인"
echo "=========================================="

# 메트릭 테이블 존재 확인
check_table() {
    local table_name=$1
    local result
    
    if command -v kubectl &> /dev/null && [ -z "$(command -v mysql)" ]; then
        # Kubernetes Pod를 사용
        result=$(kubectl run mysql-check-$table_name --rm -i --restart=Never --image=mysql:8.0 -- \
            mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SHOW TABLES LIKE '$table_name';" 2>/dev/null | grep -c "$table_name" || echo "0")
    else
        result=$($MYSQL_CMD -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SHOW TABLES LIKE '$table_name';" 2>/dev/null | grep -c "$table_name" || echo "0")
    fi
    
    if [ "$result" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} 테이블 '$table_name' 존재"
        return 0
    else
        echo -e "${RED}✗${NC} 테이블 '$table_name' 없음"
        return 1
    fi
}

check_table "instance_metrics"
check_table "network_metrics"
check_table "hypervisor_metrics"

echo ""
echo "=========================================="
echo "2. 메트릭 데이터 확인"
echo "=========================================="

# 메트릭 데이터 확인 함수
check_metrics() {
    local table_name=$1
    local resource_name=$2
    local count
    
    if command -v kubectl &> /dev/null && [ -z "$(command -v mysql)" ]; then
        count=$(kubectl run mysql-count-$table_name --rm -i --restart=Never --image=mysql:8.0 -- \
            mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SELECT COUNT(*) FROM $table_name;" 2>/dev/null | tail -n 1 || echo "0")
    else
        count=$($MYSQL_CMD -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SELECT COUNT(*) FROM $table_name;" 2>/dev/null | tail -n 1 || echo "0")
    fi
    
    if [ "$count" -gt 0 ]; then
        echo -e "${GREEN}✓${NC} $resource_name 메트릭: ${count}개 레코드"
        
        # 최근 메트릭 확인
        if command -v kubectl &> /dev/null && [ -z "$(command -v mysql)" ]; then
            recent=$(kubectl run mysql-recent-$table_name --rm -i --restart=Never --image=mysql:8.0 -- \
                mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
                -e "SELECT MAX(timestamp) FROM $table_name;" 2>/dev/null | tail -n 1 || echo "N/A")
        else
            recent=$($MYSQL_CMD -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
                -e "SELECT MAX(timestamp) FROM $table_name;" 2>/dev/null | tail -n 1 || echo "N/A")
        fi
        echo "  최근 메트릭: $recent"
        return 0
    else
        echo -e "${YELLOW}⚠${NC} $resource_name 메트릭: 데이터 없음"
        return 1
    fi
}

check_metrics "instance_metrics" "인스턴스"
check_metrics "network_metrics" "네트워크"
check_metrics "hypervisor_metrics" "하이퍼바이저"

echo ""
echo "=========================================="
echo "3. 최근 메트릭 상세 정보"
echo "=========================================="

# 최근 메트릭 상세 정보
show_recent_metrics() {
    local table_name=$1
    local resource_name=$2
    
    echo ""
    echo "[$resource_name 최근 메트릭]"
    
    if command -v kubectl &> /dev/null && [ -z "$(command -v mysql)" ]; then
        kubectl run mysql-show-$table_name --rm -i --restart=Never --image=mysql:8.0 -- \
            mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SELECT * FROM $table_name ORDER BY timestamp DESC LIMIT 3\G" 2>/dev/null || echo "조회 실패"
    else
        $MYSQL_CMD -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASSWORD" "$DB_NAME" \
            -e "SELECT * FROM $table_name ORDER BY timestamp DESC LIMIT 3\G" 2>/dev/null || echo "조회 실패"
    fi
}

show_recent_metrics "instance_metrics" "인스턴스"
show_recent_metrics "network_metrics" "네트워크"
show_recent_metrics "hypervisor_metrics" "하이퍼바이저"

echo ""
echo "=========================================="
echo "4. Collector Pod 로그 확인"
echo "=========================================="

if command -v kubectl &> /dev/null; then
    echo "Collector Pod에서 메트릭 수집 로그 확인 중..."
    echo ""
    
    # 메트릭 수집 관련 로그 확인
    kubectl logs -l app=network-collector --tail=50 | grep -i "metric" || echo "메트릭 관련 로그를 찾을 수 없습니다."
    
    echo ""
    echo "최근 Collector 로그 (마지막 20줄):"
    kubectl logs -l app=network-collector --tail=20
else
    echo -e "${YELLOW}[WARN] kubectl이 설치되어 있지 않아 Pod 로그를 확인할 수 없습니다.${NC}"
fi

echo ""
echo "=========================================="
echo "5. API 엔드포인트 확인"
echo "=========================================="

# API 엔드포인트 확인
check_api() {
    local endpoint=$1
    local name=$2
    
    if command -v kubectl &> /dev/null; then
        # Kubernetes Service를 통해 확인
        API_SERVICE=$(kubectl get svc network-collector-api -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")
        if [ -n "$API_SERVICE" ]; then
            # Port-forward를 사용하여 확인
            response=$(kubectl run curl-test --rm -i --restart=Never --image=curlimages/curl:latest -- \
                curl -s -o /dev/null -w "%{http_code}" "http://$API_SERVICE:8080$endpoint" 2>/dev/null || echo "000")
            
            if [ "$response" = "200" ]; then
                echo -e "${GREEN}✓${NC} $name API 엔드포인트: 정상 (HTTP $response)"
                return 0
            else
                echo -e "${YELLOW}⚠${NC} $name API 엔드포인트: HTTP $response"
                return 1
            fi
        else
            echo -e "${YELLOW}⚠${NC} $name API 엔드포인트: Service를 찾을 수 없음"
            return 1
        fi
    else
        echo -e "${YELLOW}⚠${NC} kubectl이 없어 API 엔드포인트를 확인할 수 없습니다."
        return 1
    fi
}

# 인스턴스 목록을 가져와서 첫 번째 인스턴스의 메트릭 확인
if command -v kubectl &> /dev/null; then
    API_SERVICE=$(kubectl get svc network-collector-api -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")
    if [ -n "$API_SERVICE" ]; then
        # 인스턴스 목록 가져오기
        INSTANCE_ID=$(kubectl run curl-instances --rm -i --restart=Never --image=curlimages/curl:latest -- \
            curl -s "http://$API_SERVICE:8080/api/v1/instances" 2>/dev/null | \
            grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
        
        if [ -n "$INSTANCE_ID" ]; then
            check_api "/api/v1/instances/$INSTANCE_ID/metrics" "인스턴스 메트릭"
        else
            echo -e "${YELLOW}⚠${NC} 인스턴스를 찾을 수 없어 메트릭 API를 테스트할 수 없습니다."
        fi
    fi
fi

echo ""
echo "=========================================="
echo "확인 완료"
echo "=========================================="
echo ""
echo "메트릭 수집이 정상적으로 동작하는지 확인하려면:"
echo "1. Collector Pod 로그에서 'Collecting metrics...' 메시지 확인"
echo "2. 데이터베이스에서 메트릭 레코드 확인"
echo "3. API 엔드포인트를 통해 메트릭 조회 테스트"
echo ""
echo "메트릭이 수집되지 않는다면:"
echo "1. Collector Pod가 정상 실행 중인지 확인: kubectl get pods -l app=network-collector"
echo "2. Collector 로그 확인: kubectl logs -l app=network-collector | grep -i metric"
echo "3. 데이터베이스 연결 확인"
echo ""

