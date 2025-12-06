#!/bin/bash

# Network Collector 종합 테스트 스크립트
# 모든 기능을 자동으로 테스트하고 결과를 보고합니다.

set -e

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 테스트 결과 추적
PASSED=0
FAILED=0
WARNINGS=0

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((FAILED++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

log_section() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

test_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((PASSED++))
}

test_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((FAILED++))
}

test_warn() {
    echo -e "${YELLOW}⚠ WARN${NC}: $1"
    ((WARNINGS++))
}

# ============================================
# 1. Kubernetes 리소스 상태 확인
# ============================================
test_kubernetes_resources() {
    log_section "1. Kubernetes 리소스 상태 확인"
    
    # Pods 확인
    log_test "Pods 상태 확인"
    PODS=$(kubectl get pods -l app=network-collector-api -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    if [ -n "$PODS" ]; then
        for pod in $PODS; do
            STATUS=$(kubectl get pod $pod -o jsonpath='{.status.phase}' 2>/dev/null)
            if [ "$STATUS" = "Running" ]; then
                test_pass "API Pod $pod: $STATUS"
            else
                test_fail "API Pod $pod: $STATUS"
            fi
        done
    else
        test_fail "API Pod를 찾을 수 없습니다"
    fi
    
    PODS=$(kubectl get pods -l app=network-collector-frontend -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    if [ -n "$PODS" ]; then
        for pod in $PODS; do
            STATUS=$(kubectl get pod $pod -o jsonpath='{.status.phase}' 2>/dev/null)
            if [ "$STATUS" = "Running" ]; then
                test_pass "Frontend Pod $pod: $STATUS"
            else
                test_fail "Frontend Pod $pod: $STATUS"
            fi
        done
    else
        test_fail "Frontend Pod를 찾을 수 없습니다"
    fi
    
    PODS=$(kubectl get pods -l app=network-collector -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
    if [ -n "$PODS" ]; then
        for pod in $PODS; do
            STATUS=$(kubectl get pod $pod -o jsonpath='{.status.phase}' 2>/dev/null)
            if [ "$STATUS" = "Running" ]; then
                test_pass "Collector Pod $pod: $STATUS"
            else
                test_fail "Collector Pod $pod: $STATUS"
            fi
        done
    else
        test_fail "Collector Pod를 찾을 수 없습니다"
    fi
    
    # Services 확인
    log_test "Services 확인"
    API_SVC=$(kubectl get svc network-collector-api -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
    if [ -n "$API_SVC" ]; then
        test_pass "API Service: $API_SVC"
    else
        test_fail "API Service를 찾을 수 없습니다"
    fi
    
    FRONTEND_SVC=$(kubectl get svc network-collector-frontend -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
    if [ -n "$FRONTEND_SVC" ]; then
        test_pass "Frontend Service: $FRONTEND_SVC"
    else
        test_fail "Frontend Service를 찾을 수 없습니다"
    fi
    
    # Ingress 확인
    log_test "Ingress 확인"
    INGRESS=$(kubectl get ingress network-collector-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
    if [ -n "$INGRESS" ] || kubectl get ingress network-collector-ingress &>/dev/null; then
        test_pass "Ingress 리소스 존재"
    else
        test_warn "Ingress 리소스를 찾을 수 없습니다"
    fi
}

# ============================================
# 2. 데이터베이스 연결 및 스키마 확인
# ============================================
test_database() {
    log_section "2. 데이터베이스 연결 및 스키마 확인"
    
    DB_POD=$(kubectl get pods -l app=mariadb -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -z "$DB_POD" ]; then
        test_fail "MariaDB Pod를 찾을 수 없습니다"
        return
    fi
    
    log_test "데이터베이스 연결 테스트"
    if kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -e "SELECT 1" openstack_monitor &>/dev/null; then
        test_pass "데이터베이스 연결 성공"
    else
        test_fail "데이터베이스 연결 실패"
        return
    fi
    
    log_test "스키마 확인"
    # ports.device_id 컬럼 확인
    DEVICE_ID_TYPE=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT DATA_TYPE FROM information_schema.COLUMNS 
        WHERE TABLE_SCHEMA = 'openstack_monitor' 
        AND TABLE_NAME = 'ports' 
        AND COLUMN_NAME = 'device_id';
    " openstack_monitor 2>/dev/null || echo "")
    
    if [ "$DEVICE_ID_TYPE" = "varchar" ]; then
        DEVICE_ID_LEN=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
            SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.COLUMNS 
            WHERE TABLE_SCHEMA = 'openstack_monitor' 
            AND TABLE_NAME = 'ports' 
            AND COLUMN_NAME = 'device_id';
        " openstack_monitor 2>/dev/null || echo "")
        if [ "$DEVICE_ID_LEN" = "255" ]; then
            test_pass "ports.device_id: VARCHAR(255)"
        else
            test_warn "ports.device_id: VARCHAR($DEVICE_ID_LEN) (예상: 255)"
        fi
    else
        test_warn "ports.device_id 타입: $DEVICE_ID_TYPE (예상: varchar)"
    fi
    
    # instances.hypervisor_id 컬럼 확인
    HYPERVISOR_ID_TYPE=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT DATA_TYPE FROM information_schema.COLUMNS 
        WHERE TABLE_SCHEMA = 'openstack_monitor' 
        AND TABLE_NAME = 'instances' 
        AND COLUMN_NAME = 'hypervisor_id';
    " openstack_monitor 2>/dev/null || echo "")
    
    if [ "$HYPERVISOR_ID_TYPE" = "varchar" ]; then
        HYPERVISOR_ID_LEN=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
            SELECT CHARACTER_MAXIMUM_LENGTH FROM information_schema.COLUMNS 
            WHERE TABLE_SCHEMA = 'openstack_monitor' 
            AND TABLE_NAME = 'instances' 
            AND COLUMN_NAME = 'hypervisor_id';
        " openstack_monitor 2>/dev/null || echo "")
        if [ "$HYPERVISOR_ID_LEN" = "255" ]; then
            test_pass "instances.hypervisor_id: VARCHAR(255)"
        else
            test_warn "instances.hypervisor_id: VARCHAR($HYPERVISOR_ID_LEN) (예상: 255)"
        fi
    else
        test_warn "instances.hypervisor_id 타입: $HYPERVISOR_ID_TYPE (예상: varchar)"
    fi
    
    log_test "데이터 확인"
    INSTANCE_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT COUNT(*) FROM instances;
    " openstack_monitor 2>/dev/null || echo "0")
    
    if [ "$INSTANCE_COUNT" -gt 0 ]; then
        test_pass "Instances: $INSTANCE_COUNT 개"
    else
        test_warn "Instances: 0 개 (데이터 수집 필요)"
    fi
    
    NETWORK_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT COUNT(*) FROM networks;
    " openstack_monitor 2>/dev/null || echo "0")
    
    if [ "$NETWORK_COUNT" -gt 0 ]; then
        test_pass "Networks: $NETWORK_COUNT 개"
    else
        test_warn "Networks: 0 개"
    fi
    
    PROJECT_COUNT=$(kubectl exec $DB_POD -- mysql -u openstack_monitor -proot -N -e "
        SELECT COUNT(*) FROM projects;
    " openstack_monitor 2>/dev/null || echo "0")
    
    if [ "$PROJECT_COUNT" -gt 0 ]; then
        test_pass "Projects: $PROJECT_COUNT 개"
    else
        test_warn "Projects: 0 개"
    fi
}

# ============================================
# 3. API 엔드포인트 테스트
# ============================================
test_api_endpoints() {
    log_section "3. API 엔드포인트 테스트"
    
    # API Pod에서 직접 테스트
    API_POD=$(kubectl get pods -l app=network-collector-api -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -z "$API_POD" ]; then
        test_fail "API Pod를 찾을 수 없습니다"
        return
    fi
    
    # Health check
    log_test "Health Check"
    if kubectl exec $API_POD -- wget -qO- http://localhost:8080/health 2>/dev/null | grep -q "ok\|healthy"; then
        test_pass "Health Check: OK"
    else
        test_fail "Health Check: 실패"
    fi
    
    # 로그인 테스트
    log_test "로그인 API"
    AUTH_TOKEN=$(kubectl get secret network-collector-secrets -o jsonpath='{.data.AUTH_TOKEN}' 2>/dev/null | base64 -d 2>/dev/null || echo "")
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "AUTH_TOKEN을 찾을 수 없습니다"
        return
    fi
    
    LOGIN_RESPONSE=$(kubectl exec $API_POD -- sh -c "
        wget -qO- --post-data='{\"token\":\"$AUTH_TOKEN\"}' \
        --header='Content-Type: application/json' \
        http://localhost:8080/api/v1/auth/login 2>/dev/null || echo 'FAIL'
    " 2>/dev/null)
    
    if echo "$LOGIN_RESPONSE" | grep -q "token\|success"; then
        test_pass "로그인 API: 성공"
        # 토큰 추출
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4 || echo "")
    else
        test_fail "로그인 API: 실패"
        TOKEN=""
    fi
    
    if [ -z "$TOKEN" ]; then
        TOKEN="$AUTH_TOKEN"
    fi
    
    # API 엔드포인트 테스트
    log_test "API 엔드포인트 테스트"
    ENDPOINTS=(
        "/api/v1/instances"
        "/api/v1/projects"
        "/api/v1/networks"
        "/api/v1/hypervisors"
        "/api/v1/flavors"
        "/api/v1/volumes"
    )
    
    for endpoint in "${ENDPOINTS[@]}"; do
        RESPONSE=$(kubectl exec $API_POD -- sh -c "
            wget -qO- --header='Authorization: Bearer $TOKEN' \
            http://localhost:8080$endpoint 2>/dev/null || echo 'FAIL'
        " 2>/dev/null)
        
        if echo "$RESPONSE" | grep -q "\[\]\|{"; then
            test_pass "$endpoint: 응답 성공"
        else
            test_fail "$endpoint: 응답 실패"
        fi
    done
}

# ============================================
# 4. Collector 상태 확인
# ============================================
test_collector() {
    log_section "4. Collector 상태 확인"
    
    COLLECTOR_POD=$(kubectl get pods -l app=network-collector -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -z "$COLLECTOR_POD" ]; then
        test_fail "Collector Pod를 찾을 수 없습니다"
        return
    fi
    
    log_test "Collector Pod 상태"
    STATUS=$(kubectl get pod $COLLECTOR_POD -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$STATUS" = "Running" ]; then
        test_pass "Collector Pod: $STATUS"
    else
        test_fail "Collector Pod: $STATUS"
    fi
    
    log_test "OpenStack 인증 정보 확인"
    AUTH_URL=$(kubectl exec $COLLECTOR_POD -- env | grep OPENSTACK_AUTH_URL | cut -d'=' -f2 || echo "")
    if [ -n "$AUTH_URL" ] && [ "$AUTH_URL" != "change-me-in-production" ]; then
        test_pass "OPENSTACK_AUTH_URL 설정됨"
    else
        test_fail "OPENSTACK_AUTH_URL이 설정되지 않았습니다"
    fi
    
    USERNAME=$(kubectl exec $COLLECTOR_POD -- env | grep OPENSTACK_USERNAME | cut -d'=' -f2 || echo "")
    if [ -n "$USERNAME" ] && [ "$USERNAME" != "change-me-in-production" ]; then
        test_pass "OPENSTACK_USERNAME 설정됨"
    else
        test_fail "OPENSTACK_USERNAME이 설정되지 않았습니다"
    fi
    
    log_test "Collector 로그 확인 (최근 20줄)"
    RECENT_LOGS=$(kubectl logs $COLLECTOR_POD --tail=20 2>/dev/null || echo "")
    
    if echo "$RECENT_LOGS" | grep -qi "error\|failed"; then
        ERROR_COUNT=$(echo "$RECENT_LOGS" | grep -ci "error\|failed" || echo "0")
        test_warn "Collector 로그에 $ERROR_COUNT 개의 에러/실패 메시지 발견"
    else
        test_pass "Collector 로그에 에러 없음"
    fi
    
    if echo "$RECENT_LOGS" | grep -qi "collection completed\|successfully"; then
        test_pass "Collector 수집 완료 메시지 확인"
    else
        test_warn "Collector 수집 완료 메시지를 찾을 수 없습니다"
    fi
}

# ============================================
# 5. 프론트엔드 접근성 테스트
# ============================================
test_frontend() {
    log_section "5. 프론트엔드 접근성 테스트"
    
    FRONTEND_POD=$(kubectl get pods -l app=network-collector-frontend -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -z "$FRONTEND_POD" ]; then
        test_fail "Frontend Pod를 찾을 수 없습니다"
        return
    fi
    
    log_test "Frontend Pod 상태"
    STATUS=$(kubectl get pod $FRONTEND_POD -o jsonpath='{.status.phase}' 2>/dev/null)
    if [ "$STATUS" = "Running" ]; then
        test_pass "Frontend Pod: $STATUS"
    else
        test_fail "Frontend Pod: $STATUS"
    fi
    
    log_test "Frontend HTTP 응답"
    RESPONSE=$(kubectl exec $FRONTEND_POD -- wget -qO- http://localhost/ 2>/dev/null || echo "")
    if echo "$RESPONSE" | grep -q "React App\|root"; then
        test_pass "Frontend HTML 응답 확인"
    else
        test_fail "Frontend HTML 응답 실패"
    fi
    
    log_test "Nginx 설정 확인"
    NGINX_CONFIG=$(kubectl exec $FRONTEND_POD -- cat /etc/nginx/conf.d/default.conf 2>/dev/null || echo "")
    if echo "$NGINX_CONFIG" | grep -q "try_files.*index.html"; then
        test_pass "Nginx React Router 설정 확인"
    else
        test_warn "Nginx React Router 설정을 찾을 수 없습니다"
    fi
}

# ============================================
# 6. Ingress 및 외부 접근 테스트
# ============================================
test_ingress() {
    log_section "6. Ingress 및 외부 접근 테스트"
    
    log_test "Ingress 리소스 확인"
    if kubectl get ingress network-collector-ingress &>/dev/null; then
        test_pass "Ingress 리소스 존재"
        
        INGRESS_CLASS=$(kubectl get ingress network-collector-ingress -o jsonpath='{.spec.ingressClassName}' 2>/dev/null || echo "")
        if [ -n "$INGRESS_CLASS" ]; then
            test_pass "Ingress Class: $INGRESS_CLASS"
        else
            test_warn "Ingress Class가 설정되지 않았습니다"
        fi
    else
        test_fail "Ingress 리소스를 찾을 수 없습니다"
    fi
    
    # Ingress Controller 확인
    log_test "Ingress Controller 확인"
    INGRESS_POD=$(kubectl get pods -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [ -n "$INGRESS_POD" ]; then
        test_pass "Ingress Controller Pod: $INGRESS_POD"
    else
        test_warn "Ingress Controller Pod를 찾을 수 없습니다"
    fi
}

# ============================================
# 7. 종합 요약
# ============================================
print_summary() {
    log_section "테스트 결과 요약"
    
    TOTAL=$((PASSED + FAILED + WARNINGS))
    
    echo -e "${GREEN}통과: $PASSED${NC}"
    echo -e "${YELLOW}경고: $WARNINGS${NC}"
    echo -e "${RED}실패: $FAILED${NC}"
    echo -e "총 테스트: $TOTAL"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        if [ $WARNINGS -eq 0 ]; then
            echo -e "${GREEN}✓ 모든 테스트 통과!${NC}"
            return 0
        else
            echo -e "${YELLOW}⚠ 일부 경고가 있지만 기본 기능은 정상입니다.${NC}"
            return 0
        fi
    else
        echo -e "${RED}✗ 일부 테스트 실패. 위의 에러를 확인하세요.${NC}"
        return 1
    fi
}

# ============================================
# 메인 실행
# ============================================
main() {
    echo -e "${CYAN}"
    echo "╔════════════════════════════════════════╗"
    echo "║  Network Collector 종합 테스트 스크립트  ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${NC}"
    
    test_kubernetes_resources
    test_database
    test_api_endpoints
    test_collector
    test_frontend
    test_ingress
    
    print_summary
}

main "$@"

