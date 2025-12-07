#!/bin/bash

# Network Collector API 및 프론트엔드 상세 테스트 스크립트
# 모든 API 엔드포인트와 프론트엔드 페이지를 테스트하고 결과를 보고합니다.

set +e

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
# 설정 및 초기화
# ============================================
init_test() {
    log_section "테스트 초기화"
    
    # Ingress IP/Port 확인
    INGRESS_IP=$(kubectl get ingress network-collector-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || \
                 kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || \
                 echo "localhost")
    
    INGRESS_PORT=$(kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || \
                   echo "30131")
    
    if [ "$INGRESS_IP" = "localhost" ]; then
        # NodePort를 사용하는 경우
        BASE_URL="http://localhost:${INGRESS_PORT}"
    else
        BASE_URL="http://${INGRESS_IP}:${INGRESS_PORT}"
    fi
    
    API_URL="${BASE_URL}/api/v1"
    FRONTEND_URL="${BASE_URL}"
    
    log_info "Base URL: $BASE_URL"
    log_info "API URL: $API_URL"
    log_info "Frontend URL: $FRONTEND_URL"
    
    # curl 설치 확인
    if ! command -v curl &> /dev/null; then
        test_fail "curl이 설치되어 있지 않습니다"
        exit 1
    fi
    
    # jq 설치 확인 (JSON 파싱용)
    if ! command -v jq &> /dev/null; then
        log_warn "jq가 설치되어 있지 않습니다. JSON 파싱이 제한됩니다."
        JQ_AVAILABLE=false
    else
        JQ_AVAILABLE=true
    fi
}

# ============================================
# 1. 인증 테스트
# ============================================
test_authentication() {
    log_section "1. 인증 테스트"
    
    log_test "로그인 API 테스트"
    LOGIN_RESPONSE=$(curl -s -X POST "${API_URL}/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"operator","password":"password"}' 2>/dev/null)
    
    if [ -z "$LOGIN_RESPONSE" ]; then
        test_fail "로그인 API 응답 없음"
        return
    fi
    
    # 토큰 추출
    if [ "$JQ_AVAILABLE" = true ]; then
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty' 2>/dev/null)
        ROLE=$(echo "$LOGIN_RESPONSE" | jq -r '.role // empty' 2>/dev/null)
    else
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4 || echo "")
        ROLE=$(echo "$LOGIN_RESPONSE" | grep -o '"role":"[^"]*' | cut -d'"' -f4 || echo "")
    fi
    
    if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
        test_pass "로그인 성공 (토큰 획득)"
        export AUTH_TOKEN="$TOKEN"
        if [ -n "$ROLE" ] && [ "$ROLE" != "null" ]; then
            test_pass "역할 확인: $ROLE"
        fi
    else
        test_fail "로그인 실패 (토큰 없음)"
        log_error "응답: $LOGIN_RESPONSE"
        export AUTH_TOKEN=""
    fi
    
    log_test "인증 없이 보호된 엔드포인트 접근 테스트"
    PROTECTED_RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/instances" 2>/dev/null)
    HTTP_CODE=$(echo "$PROTECTED_RESPONSE" | tail -n1)
    if [ "$HTTP_CODE" = "401" ]; then
        test_pass "인증 없이 접근 시 401 반환"
    else
        test_fail "인증 없이 접근 시 예상치 못한 응답: $HTTP_CODE"
    fi
}

# ============================================
# 2. 리소스 API 테스트
# ============================================
test_resource_apis() {
    log_section "2. 리소스 API 테스트"
    
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "인증 토큰이 없어 리소스 API 테스트를 스킵합니다"
        return
    fi
    
    # 리소스 목록 API 테스트
    RESOURCES=(
        "instances:Instance"
        "projects:Project"
        "networks:Network"
        "hypervisors:Hypervisor"
        "flavors:Flavor"
        "volumes:Volume"
    )
    
    for resource_info in "${RESOURCES[@]}"; do
        RESOURCE=$(echo "$resource_info" | cut -d':' -f1)
        RESOURCE_NAME=$(echo "$resource_info" | cut -d':' -f2)
        
        log_test "${RESOURCE_NAME} 목록 API 테스트"
        
        RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/${RESOURCE}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
        BODY=$(echo "$RESPONSE" | sed '$d')
        
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "${RESOURCE_NAME} 목록 API: HTTP 200"
            
            # JSON 형식 검증
            if [ "$JQ_AVAILABLE" = true ]; then
                if echo "$BODY" | jq empty 2>/dev/null; then
                    test_pass "${RESOURCE_NAME} 응답: 유효한 JSON"
                    
                    # data 필드 확인
                    if echo "$BODY" | jq -e '.data' >/dev/null 2>&1; then
                        test_pass "${RESOURCE_NAME} 응답: data 필드 존재"
                        
                        # 배열인지 확인
                        DATA_TYPE=$(echo "$BODY" | jq -r '.data | type' 2>/dev/null)
                        if [ "$DATA_TYPE" = "array" ]; then
                            COUNT=$(echo "$BODY" | jq '.data | length' 2>/dev/null)
                            test_pass "${RESOURCE_NAME} 응답: 배열 (${COUNT}개 항목)"
                            
                            # 첫 번째 항목의 필수 필드 확인
                            if [ "$COUNT" -gt 0 ]; then
                                FIRST_ITEM=$(echo "$BODY" | jq '.data[0]' 2>/dev/null)
                                
                                # 공통 필드 확인
                                if echo "$FIRST_ITEM" | jq -e '.id' >/dev/null 2>&1; then
                                    test_pass "${RESOURCE_NAME} 항목: id 필드 존재"
                                else
                                    test_warn "${RESOURCE_NAME} 항목: id 필드 없음"
                                fi
                                
                                # 리소스별 필수 필드 확인
                                case "$RESOURCE" in
                                    "instances")
                                        if echo "$FIRST_ITEM" | jq -e '.name, .status, .openstack_id' >/dev/null 2>&1; then
                                            test_pass "${RESOURCE_NAME} 항목: 필수 필드 존재"
                                        else
                                            test_warn "${RESOURCE_NAME} 항목: 일부 필수 필드 없음"
                                        fi
                                        ;;
                                    "hypervisors")
                                        if echo "$FIRST_ITEM" | jq -e '.hostname, .state, .status' >/dev/null 2>&1; then
                                            test_pass "${RESOURCE_NAME} 항목: 필수 필드 존재"
                                        else
                                            test_warn "${RESOURCE_NAME} 항목: 일부 필수 필드 없음"
                                        fi
                                        ;;
                                    "networks")
                                        if echo "$FIRST_ITEM" | jq -e '.name, .status' >/dev/null 2>&1; then
                                            test_pass "${RESOURCE_NAME} 항목: 필수 필드 존재"
                                        else
                                            test_warn "${RESOURCE_NAME} 항목: 일부 필수 필드 없음"
                                        fi
                                        ;;
                                esac
                            fi
                        else
                            test_warn "${RESOURCE_NAME} 응답: data가 배열이 아님 (${DATA_TYPE})"
                        fi
                    else
                        test_warn "${RESOURCE_NAME} 응답: data 필드 없음"
                    fi
                else
                    test_fail "${RESOURCE_NAME} 응답: 유효하지 않은 JSON"
                fi
            else
                # jq 없이 기본 검증
                if echo "$BODY" | grep -q "data"; then
                    test_pass "${RESOURCE_NAME} 응답: data 필드 포함"
                else
                    test_warn "${RESOURCE_NAME} 응답: data 필드 확인 불가 (jq 필요)"
                fi
            fi
        else
            test_fail "${RESOURCE_NAME} 목록 API: HTTP ${HTTP_CODE}"
            log_error "응답: ${BODY:0:200}"
        fi
        
        # 개별 리소스 조회 테스트 (ID가 있는 경우)
        if [ "$JQ_AVAILABLE" = true ] && [ "$COUNT" -gt 0 ]; then
            FIRST_ID=$(echo "$BODY" | jq -r '.data[0].id' 2>/dev/null)
            if [ -n "$FIRST_ID" ] && [ "$FIRST_ID" != "null" ]; then
                log_test "${RESOURCE_NAME} 개별 조회 API 테스트 (ID: ${FIRST_ID:0:8}...)"
                
                DETAIL_RESPONSE=$(curl -s -w "\n%{http_code}" \
                    -H "Authorization: Bearer ${AUTH_TOKEN}" \
                    "${API_URL}/${RESOURCE}/${FIRST_ID}" 2>/dev/null)
                
                DETAIL_HTTP_CODE=$(echo "$DETAIL_RESPONSE" | tail -n1)
                DETAIL_BODY=$(echo "$DETAIL_RESPONSE" | sed '$d')
                
                if [ "$DETAIL_HTTP_CODE" = "200" ]; then
                    test_pass "${RESOURCE_NAME} 개별 조회: HTTP 200"
                    
                    if echo "$DETAIL_BODY" | jq -e '.data' >/dev/null 2>&1; then
                        test_pass "${RESOURCE_NAME} 개별 조회: data 필드 존재"
                    else
                        test_warn "${RESOURCE_NAME} 개별 조회: data 필드 없음"
                    fi
                elif [ "$DETAIL_HTTP_CODE" = "404" ]; then
                    test_warn "${RESOURCE_NAME} 개별 조회: 404 (리소스 없음)"
                else
                    test_fail "${RESOURCE_NAME} 개별 조회: HTTP ${DETAIL_HTTP_CODE}"
                fi
            fi
        fi
    done
}

# ============================================
# 3. Topology API 테스트
# ============================================
test_topology_apis() {
    log_section "3. Topology API 테스트"
    
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "인증 토큰이 없어 Topology API 테스트를 스킵합니다"
        return
    fi
    
    # Instance ID 가져오기
    INSTANCES_RESPONSE=$(curl -s \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        "${API_URL}/instances" 2>/dev/null)
    
    if [ "$JQ_AVAILABLE" = true ]; then
        INSTANCE_ID=$(echo "$INSTANCES_RESPONSE" | jq -r '.data[0].id // empty' 2>/dev/null)
        HYPERVISOR_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/hypervisors" 2>/dev/null | \
                       jq -r '.data[0].id // empty' 2>/dev/null)
        NETWORK_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/networks" 2>/dev/null | \
                    jq -r '.data[0].id // empty' 2>/dev/null)
    else
        INSTANCE_ID=$(echo "$INSTANCES_RESPONSE" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4 || echo "")
    fi
    
    # Instance Topology
    if [ -n "$INSTANCE_ID" ] && [ "$INSTANCE_ID" != "null" ]; then
        log_test "Instance Topology API 테스트"
        TOPOLOGY_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/topology/instances/${INSTANCE_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$TOPOLOGY_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Instance Topology API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Instance Topology: 404 (데이터 없음)"
        else
            test_fail "Instance Topology API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Instance Topology: 테스트할 인스턴스 없음"
    fi
    
    # Host Topology
    if [ -n "$HYPERVISOR_ID" ] && [ "$HYPERVISOR_ID" != "null" ]; then
        log_test "Host Topology API 테스트"
        TOPOLOGY_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/topology/hosts/${HYPERVISOR_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$TOPOLOGY_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Host Topology API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Host Topology: 404 (데이터 없음)"
        else
            test_fail "Host Topology API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Host Topology: 테스트할 하이퍼바이저 없음"
    fi
    
    # Network Topology
    if [ -n "$NETWORK_ID" ] && [ "$NETWORK_ID" != "null" ]; then
        log_test "Network Topology API 테스트"
        TOPOLOGY_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/topology/networks/${NETWORK_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$TOPOLOGY_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Network Topology API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Network Topology: 404 (데이터 없음)"
        else
            test_fail "Network Topology API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Network Topology: 테스트할 네트워크 없음"
    fi
}

# ============================================
# 4. Metrics API 테스트
# ============================================
test_metrics_apis() {
    log_section "4. Metrics API 테스트"
    
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "인증 토큰이 없어 Metrics API 테스트를 스킵합니다"
        return
    fi
    
    # Instance ID 가져오기
    if [ "$JQ_AVAILABLE" = true ]; then
        INSTANCE_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/instances" 2>/dev/null | \
                     jq -r '.data[0].id // empty' 2>/dev/null)
        NETWORK_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/networks" 2>/dev/null | \
                    jq -r '.data[0].id // empty' 2>/dev/null)
        HYPERVISOR_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/hypervisors" 2>/dev/null | \
                       jq -r '.data[0].id // empty' 2>/dev/null)
    else
        INSTANCE_ID=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/instances" 2>/dev/null | \
                     grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4 || echo "")
    fi
    
    # Instance Metrics
    if [ -n "$INSTANCE_ID" ] && [ "$INSTANCE_ID" != "null" ]; then
        log_test "Instance Metrics API 테스트"
        METRICS_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/metrics/instances/${INSTANCE_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$METRICS_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Instance Metrics API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Instance Metrics: 404 (데이터 없음)"
        else
            test_fail "Instance Metrics API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Instance Metrics: 테스트할 인스턴스 없음"
    fi
    
    # Network Metrics
    if [ -n "$NETWORK_ID" ] && [ "$NETWORK_ID" != "null" ]; then
        log_test "Network Metrics API 테스트"
        METRICS_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/metrics/networks/${NETWORK_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$METRICS_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Network Metrics API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Network Metrics: 404 (데이터 없음)"
        else
            test_fail "Network Metrics API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Network Metrics: 테스트할 네트워크 없음"
    fi
    
    # Hypervisor Metrics
    if [ -n "$HYPERVISOR_ID" ] && [ "$HYPERVISOR_ID" != "null" ]; then
        log_test "Hypervisor Metrics API 테스트"
        METRICS_RESPONSE=$(curl -s -w "\n%{http_code}" \
            -H "Authorization: Bearer ${AUTH_TOKEN}" \
            "${API_URL}/metrics/hypervisors/${HYPERVISOR_ID}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$METRICS_RESPONSE" | tail -n1)
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "Hypervisor Metrics API: HTTP 200"
        elif [ "$HTTP_CODE" = "404" ]; then
            test_warn "Hypervisor Metrics: 404 (데이터 없음)"
        else
            test_fail "Hypervisor Metrics API: HTTP ${HTTP_CODE}"
        fi
    else
        test_warn "Hypervisor Metrics: 테스트할 하이퍼바이저 없음"
    fi
}

# ============================================
# 5. Project API 테스트
# ============================================
test_project_apis() {
    log_section "5. Project API 테스트"
    
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "인증 토큰이 없어 Project API 테스트를 스킵합니다"
        return
    fi
    
    # Project Compare API
    log_test "Project Compare API 테스트"
    if [ "$JQ_AVAILABLE" = true ]; then
        PROJECTS_RESPONSE=$(curl -s -H "Authorization: Bearer ${AUTH_TOKEN}" "${API_URL}/projects" 2>/dev/null)
        PROJECT_IDS=($(echo "$PROJECTS_RESPONSE" | jq -r '.data[].id' 2>/dev/null | head -2))
        
        if [ ${#PROJECT_IDS[@]} -ge 2 ]; then
            COMPARE_RESPONSE=$(curl -s -w "\n%{http_code}" \
                -H "Authorization: Bearer ${AUTH_TOKEN}" \
                "${API_URL}/projects/compare?project_ids=${PROJECT_IDS[0]},${PROJECT_IDS[1]}" 2>/dev/null)
            
            HTTP_CODE=$(echo "$COMPARE_RESPONSE" | tail -n1)
            if [ "$HTTP_CODE" = "200" ]; then
                test_pass "Project Compare API: HTTP 200"
            else
                test_fail "Project Compare API: HTTP ${HTTP_CODE}"
            fi
        else
            test_warn "Project Compare: 비교할 프로젝트가 2개 미만"
        fi
        
        # Project Summary API
        if [ ${#PROJECT_IDS[@]} -ge 1 ]; then
            log_test "Project Summary API 테스트"
            SUMMARY_RESPONSE=$(curl -s -w "\n%{http_code}" \
                -H "Authorization: Bearer ${AUTH_TOKEN}" \
                "${API_URL}/projects/${PROJECT_IDS[0]}/summary" 2>/dev/null)
            
            HTTP_CODE=$(echo "$SUMMARY_RESPONSE" | tail -n1)
            if [ "$HTTP_CODE" = "200" ]; then
                test_pass "Project Summary API: HTTP 200"
            elif [ "$HTTP_CODE" = "404" ]; then
                test_warn "Project Summary: 404 (프로젝트 없음)"
            else
                test_fail "Project Summary API: HTTP ${HTTP_CODE}"
            fi
        fi
    else
        test_warn "Project API: jq가 없어 상세 테스트 스킵"
    fi
}

# ============================================
# 6. 프론트엔드 페이지 테스트
# ============================================
test_frontend_pages() {
    log_section "6. 프론트엔드 페이지 테스트"
    
    PAGES=(
        "/:Dashboard"
        "/login:Login"
        "/topology:Topology"
        "/metrics:Metrics"
        "/projects/compare:Project Comparison"
    )
    
    for page_info in "${PAGES[@]}"; do
        PATH=$(echo "$page_info" | cut -d':' -f1)
        PAGE_NAME=$(echo "$page_info" | cut -d':' -f2)
        
        log_test "${PAGE_NAME} 페이지 접근 테스트"
        
        RESPONSE=$(curl -s -w "\n%{http_code}" \
            -L "${FRONTEND_URL}${PATH}" 2>/dev/null)
        
        HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
        BODY=$(echo "$RESPONSE" | sed '$d')
        
        if [ "$HTTP_CODE" = "200" ]; then
            test_pass "${PAGE_NAME} 페이지: HTTP 200"
            
            # HTML 내용 확인
            if echo "$BODY" | grep -q "<!doctype html\|<html\|<div.*root\|React"; then
                test_pass "${PAGE_NAME} 페이지: HTML 내용 확인"
            else
                test_warn "${PAGE_NAME} 페이지: HTML 내용 확인 실패"
            fi
        else
            test_fail "${PAGE_NAME} 페이지: HTTP ${HTTP_CODE}"
        fi
    done
    
    # React Router 테스트 (존재하지 않는 경로가 index.html로 리다이렉트되는지)
    log_test "React Router 리다이렉트 테스트"
    RESPONSE=$(curl -s -w "\n%{http_code}" \
        -L "${FRONTEND_URL}/nonexistent-page" 2>/dev/null)
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    if [ "$HTTP_CODE" = "200" ]; then
        test_pass "React Router: 존재하지 않는 경로도 200 반환 (SPA 리다이렉트)"
    else
        test_warn "React Router: 존재하지 않는 경로가 ${HTTP_CODE} 반환"
    fi
}

# ============================================
# 7. Events API (SSE) 테스트
# ============================================
test_events_api() {
    log_section "7. Events API (SSE) 테스트"
    
    if [ -z "$AUTH_TOKEN" ]; then
        test_fail "인증 토큰이 없어 Events API 테스트를 스킵합니다"
        return
    fi
    
    log_test "Events Stream API 테스트 (5초 타임아웃)"
    
    # SSE 연결 테스트 (타임아웃 5초)
    TIMEOUT=5
    START_TIME=$(date +%s)
    
    RESPONSE=$(timeout ${TIMEOUT} curl -s -N \
        -H "Authorization: Bearer ${AUTH_TOKEN}" \
        "${API_URL}/events/stream" 2>/dev/null || echo "TIMEOUT")
    
    END_TIME=$(date +%s)
    ELAPSED=$((END_TIME - START_TIME))
    
    if [ "$RESPONSE" = "TIMEOUT" ] || [ $ELAPSED -ge $TIMEOUT ]; then
        test_pass "Events Stream API: 연결 성공 (타임아웃까지 연결 유지)"
    elif echo "$RESPONSE" | grep -q "data:\|event:"; then
        test_pass "Events Stream API: 이벤트 수신"
    elif echo "$RESPONSE" | grep -q "error\|401\|403"; then
        test_fail "Events Stream API: 오류 응답"
    else
        test_warn "Events Stream API: 응답 확인 불가"
    fi
}

# ============================================
# 8. 종합 요약
# ============================================
print_summary() {
    log_section "테스트 요약"
    
    TOTAL=$((PASSED + FAILED + WARNINGS))
    
    echo -e "${GREEN}통과: ${PASSED}${NC}"
    echo -e "${YELLOW}경고: ${WARNINGS}${NC}"
    echo -e "${RED}실패: ${FAILED}${NC}"
    echo -e "총 테스트: ${TOTAL}"
    echo ""
    
    if [ $FAILED -eq 0 ]; then
        echo -e "${GREEN}✓ 모든 테스트 통과!${NC}"
        exit 0
    else
        echo -e "${RED}✗ 일부 테스트 실패${NC}"
        exit 1
    fi
}

# ============================================
# 메인 실행
# ============================================
main() {
    echo -e "${CYAN}"
    echo "=========================================="
    echo "  Network Collector API & Frontend Test"
    echo "=========================================="
    echo -e "${NC}"
    
    init_test
    test_authentication
    test_resource_apis
    test_topology_apis
    test_metrics_apis
    test_project_apis
    test_frontend_pages
    test_events_api
    print_summary
}

main

