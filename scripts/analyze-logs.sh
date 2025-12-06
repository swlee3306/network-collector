#!/bin/bash

# 네트워크 컬렉터 시스템의 모든 로그를 분석하는 스크립트
# 사용법: ./scripts/analyze-logs.sh [ssh_user@ssh_host]

set -e

# 색상 출력
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

log_section() {
    echo -e "\n${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}\n"
}

# SSH 연결 설정
if [ $# -gt 0 ]; then
    SSH_TARGET="$1"
    SSH_CMD="ssh $SSH_TARGET"
    log_info "원격 서버에 접속합니다: $SSH_TARGET"
else
    SSH_CMD=""
    log_info "로컬에서 실행합니다."
fi

# SSH 키 확인
SSH_KEY="$HOME/.ssh/id_rsa_network_collector"
if [ -f "$SSH_KEY" ] && [ -n "$SSH_CMD" ]; then
    SSH_CMD="ssh -i $SSH_KEY $SSH_TARGET"
    log_info "SSH 키 사용: $SSH_KEY"
fi

# 실행 함수
run_cmd() {
    if [ -n "$SSH_CMD" ]; then
        $SSH_CMD "$1"
    else
        eval "$1"
    fi
}

# Pod 상태 확인
check_pod_status() {
    log_section "Pod 상태 확인"
    
    run_cmd "kubectl get pods -A -o wide | grep -E 'network-collector|mariadb|ingress' || true"
    
    echo
    log_info "Pod 상세 상태:"
    PODS=$(run_cmd "kubectl get pods -A -o jsonpath='{.items[*].metadata.name}'" 2>/dev/null | tr ' ' '\n' | grep -E 'network-collector|mariadb' || echo "")
    
    if [ -n "$PODS" ]; then
        for pod in $PODS; do
            NS=$(run_cmd "kubectl get pod $pod -A -o jsonpath='{.metadata.namespace}'" 2>/dev/null || echo "default")
            STATUS=$(run_cmd "kubectl get pod $pod -n $NS -o jsonpath='{.status.phase}'" 2>/dev/null || echo "Unknown")
            RESTARTS=$(run_cmd "kubectl get pod $pod -n $NS -o jsonpath='{.status.containerStatuses[0].restartCount}'" 2>/dev/null || echo "0")
            echo "  Pod: $pod (Namespace: $NS) | 상태: $STATUS | 재시작: $RESTARTS"
        done
    fi
}

# 서비스 상태 확인
check_service_status() {
    log_section "서비스 상태 확인"
    
    run_cmd "kubectl get services -A | grep -E 'network-collector|mariadb|ingress' || true"
    
    echo
    log_info "엔드포인트 확인:"
    run_cmd "kubectl get endpoints -A | grep -E 'network-collector|mariadb' || true"
}

# Ingress 상태 확인
check_ingress_status() {
    log_section "Ingress 상태 확인"
    
    run_cmd "kubectl get ingress -A -o wide || true"
    
    echo
    log_info "Ingress 상세 정보:"
    run_cmd "kubectl describe ingress network-collector-ingress -n default 2>/dev/null || true"
}

# API 로그 분석
analyze_api_logs() {
    log_section "API 로그 분석"
    
    API_PODS=$(run_cmd "kubectl get pods -l app=network-collector-api -o jsonpath='{.items[*].metadata.name}'" 2>/dev/null || echo "")
    
    if [ -z "$API_PODS" ]; then
        log_warn "API Pod를 찾을 수 없습니다."
        return
    fi
    
    for pod in $API_PODS; do
        log_info "API Pod: $pod"
        echo "--- 최근 50줄 로그 ---"
        run_cmd "kubectl logs $pod --tail=50 2>/dev/null || true"
        echo
        echo "--- 에러 로그 ---"
        run_cmd "kubectl logs $pod --tail=100 2>/dev/null | grep -i 'error\|fatal\|panic' || echo '에러 없음'"
        echo
    done
}

# Frontend 로그 분석
analyze_frontend_logs() {
    log_section "Frontend 로그 분석"
    
    FRONTEND_PODS=$(run_cmd "kubectl get pods -l app=network-collector-frontend -o jsonpath='{.items[*].metadata.name}'" 2>/dev/null || echo "")
    
    if [ -z "$FRONTEND_PODS" ]; then
        log_warn "Frontend Pod를 찾을 수 없습니다."
        return
    fi
    
    for pod in $FRONTEND_PODS; do
        log_info "Frontend Pod: $pod"
        echo "--- 최근 50줄 로그 ---"
        run_cmd "kubectl logs $pod --tail=50 2>/dev/null || true"
        echo
    done
}

# Collector 로그 분석
analyze_collector_logs() {
    log_section "Collector 로그 분석"
    
    COLLECTOR_PODS=$(run_cmd "kubectl get pods -l app=network-collector -o jsonpath='{.items[*].metadata.name}'" 2>/dev/null || echo "")
    
    if [ -z "$COLLECTOR_PODS" ]; then
        log_warn "Collector Pod를 찾을 수 없습니다."
        return
    fi
    
    for pod in $COLLECTOR_PODS; do
        log_info "Collector Pod: $pod"
        echo "--- 최근 50줄 로그 ---"
        run_cmd "kubectl logs $pod --tail=50 2>/dev/null || true"
        echo
        echo "--- 에러 로그 ---"
        run_cmd "kubectl logs $pod --tail=200 2>/dev/null | grep -i 'error\|fatal\|panic\|failed' || echo '에러 없음'"
        echo
    done
}

# 데이터베이스 연결 확인
check_database() {
    log_section "데이터베이스 상태 확인"
    
    # MariaDB Pod 상태
    DB_PODS=$(run_cmd "kubectl get pods -l app=mariadb -o jsonpath='{.items[*].metadata.name}'" 2>/dev/null || echo "")
    
    if [ -z "$DB_PODS" ]; then
        log_warn "MariaDB Pod를 찾을 수 없습니다."
        return
    fi
    
    for pod in $DB_PODS; do
        log_info "MariaDB Pod: $pod"
        echo "--- 최근 30줄 로그 ---"
        run_cmd "kubectl logs $pod --tail=30 2>/dev/null || true"
        echo
    done
    
    # 데이터베이스 연결 테스트
    log_info "데이터베이스 연결 테스트:"
    API_POD=$(run_cmd "kubectl get pods -l app=network-collector-api -o jsonpath='{.items[0].metadata.name}'" 2>/dev/null || echo "")
    if [ -n "$API_POD" ]; then
        run_cmd "kubectl exec $API_POD -- env | grep -E 'DB_|DATABASE' || true"
    fi
}

# API 엔드포인트 테스트
test_api_endpoints() {
    log_section "API 엔드포인트 테스트"
    
    # Ingress를 통한 접근
    INGRESS_IP=$(run_cmd "kubectl get ingress network-collector-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type==\"InternalIP\")].address}' 2>/dev/null || echo ''")
    INGRESS_PORT=$(run_cmd "kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.spec.ports[?(@.port==80)].nodePort}' 2>/dev/null || echo '30131'")
    
    if [ -n "$INGRESS_IP" ]; then
        log_info "Ingress 접근 테스트: http://$INGRESS_IP:$INGRESS_PORT"
        
        echo "--- 헬스체크 ---"
        run_cmd "curl -s -o /dev/null -w 'HTTP Status: %{http_code}\n' http://$INGRESS_IP:$INGRESS_PORT/api/v1/health || echo '연결 실패'"
        
        echo "--- 로그인 테스트 ---"
        run_cmd "curl -s -X POST http://$INGRESS_IP:$INGRESS_PORT/api/v1/auth/login -H 'Content-Type: application/json' -d '{\"username\":\"test\",\"password\":\"test\"}' | head -100 || echo '로그인 실패'"
    fi
    
    # 서비스 직접 접근
    log_info "서비스 직접 접근 테스트:"
    API_SVC=$(run_cmd "kubectl get svc network-collector-api -o jsonpath='{.spec.clusterIP}'" 2>/dev/null || echo "")
    if [ -n "$API_SVC" ]; then
        API_POD=$(run_cmd "kubectl get pods -l app=network-collector-api -o jsonpath='{.items[0].metadata.name}'" 2>/dev/null || echo "")
        if [ -n "$API_POD" ]; then
            echo "--- Pod 내부에서 API 테스트 ---"
            run_cmd "kubectl exec $API_POD -- curl -s http://localhost:8080/health || echo 'API 응답 없음'"
        fi
    fi
}

# 이벤트 확인
check_events() {
    log_section "Kubernetes 이벤트 확인"
    
    run_cmd "kubectl get events -A --sort-by='.lastTimestamp' | grep -E 'network-collector|mariadb' | tail -20 || true"
}

# 리소스 사용량 확인
check_resources() {
    log_section "리소스 사용량 확인"
    
    run_cmd "kubectl top pods -A 2>/dev/null | grep -E 'network-collector|mariadb' || echo '메트릭스 서버가 없거나 접근 불가'"
    echo
    run_cmd "kubectl top nodes 2>/dev/null || echo '메트릭스 서버가 없거나 접근 불가'"
}

# ConfigMap 및 Secret 확인
check_config() {
    log_section "ConfigMap 및 Secret 확인"
    
    log_info "ConfigMap:"
    run_cmd "kubectl get configmap network-collector-config -o yaml 2>/dev/null | grep -E '^  [A-Z_]|^data:' || true"
    run_cmd "kubectl get configmap network-collector-frontend-config -o yaml 2>/dev/null | grep -E '^  [A-Z_]|^data:' || true"
    
    echo
    log_info "Secret 존재 확인 (값은 표시하지 않음):"
    run_cmd "kubectl get secret network-collector-secrets -o jsonpath='{.data}' 2>/dev/null | jq 'keys' 2>/dev/null || kubectl get secret network-collector-secrets -o jsonpath='{.data}' 2>/dev/null | grep -o '\"[^\"]*\"' || echo 'Secret 확인 실패'"
}

# 네트워크 연결 확인
check_network() {
    log_section "네트워크 연결 확인"
    
    API_POD=$(run_cmd "kubectl get pods -l app=network-collector-api -o jsonpath='{.items[0].metadata.name}'" 2>/dev/null || echo "")
    if [ -n "$API_POD" ]; then
        log_info "API Pod에서 데이터베이스 연결 테스트:"
        run_cmd "kubectl exec $API_POD -- ping -c 2 mariadb 2>/dev/null || echo 'ping 실패'"
    fi
}

# 종합 요약
summary() {
    log_section "종합 요약"
    
    echo "=== Pod 상태 ==="
    run_cmd "kubectl get pods -A | grep -E 'network-collector|mariadb' || true"
    
    echo
    echo "=== 서비스 상태 ==="
    run_cmd "kubectl get svc -A | grep -E 'network-collector|mariadb' || true"
    
    echo
    echo "=== 주요 문제점 ==="
    
    # 실패한 Pod 확인
    FAILED_PODS=$(run_cmd "kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.status.phase}{\"\\n\"}{end}'" 2>/dev/null | grep -E 'network-collector|mariadb' | grep -v Running || echo "")
    if [ -n "$FAILED_PODS" ]; then
        log_error "실패한 Pod:"
        echo "$FAILED_PODS"
    else
        log_info "모든 Pod가 정상 실행 중입니다."
    fi
    
    # 재시작된 Pod 확인
    RESTARTED_PODS=$(run_cmd "kubectl get pods -A -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.status.containerStatuses[0].restartCount}{\"\\n\"}{end}'" 2>/dev/null | grep -E 'network-collector|mariadb' | awk -F'\t' '$2 > 0' || echo "")
    if [ -n "$RESTARTED_PODS" ]; then
        log_warn "재시작된 Pod:"
        echo "$RESTARTED_PODS"
    fi
}

# 메인 실행
main() {
    log_section "네트워크 컬렉터 시스템 로그 분석 시작"
    
    check_pod_status
    check_service_status
    check_ingress_status
    check_config
    check_database
    check_network
    analyze_api_logs
    analyze_frontend_logs
    analyze_collector_logs
    test_api_endpoints
    check_events
    check_resources
    summary
    
    log_section "로그 분석 완료"
    log_info "상세 로그를 보려면:"
    echo "  kubectl logs -f <pod-name>"
    echo "  kubectl describe pod <pod-name>"
}

main

