#!/bin/bash

# OpenStack 엔드포인트 확인 스크립트

echo "=== Collector Pod 정보 ==="
kubectl get pods -l app=network-collector -o wide

echo ""
echo "=== Collector Pod 환경변수 ==="
POD_NAME=$(kubectl get pods -l app=network-collector -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$POD_NAME" ]; then
    echo "Pod: $POD_NAME"
    kubectl get pod "$POD_NAME" -o jsonpath='{.spec.containers[0].env[*]}' | tr ' ' '\n' | grep -E "OPENSTACK_ENDPOINT_TYPE|LOGIN_TYPE"
else
    echo "Collector Pod를 찾을 수 없습니다."
fi

echo ""
echo "=== ConfigMap 확인 ==="
kubectl get configmap network-collector-config -o yaml | grep -A 5 "OPENSTACK_ENDPOINT_TYPE"

echo ""
echo "=== Collector 로그 (최근 50줄, OpenStack 관련) ==="
kubectl logs -l app=network-collector --tail=50 | grep -i "OpenStack\|endpoint\|Creating" || echo "로그를 찾을 수 없습니다."

echo ""
echo "=== Collector 전체 로그 (최근 20줄) ==="
kubectl logs -l app=network-collector --tail=20

