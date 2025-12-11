#!/bin/bash

# ConfigMap과 Pod 환경변수 확인 스크립트

NAMESPACE=${1:-default}

echo "=== ConfigMap 확인 ==="
kubectl get configmap network-collector-config -n "$NAMESPACE" -o yaml | grep -A 20 "data:"

echo ""
echo "=== Collector Pod 환경변수 확인 ==="
COLLECTOR_POD=$(kubectl get pods -n "$NAMESPACE" -l app=network-collector -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$COLLECTOR_POD" ]; then
    echo "Collector Pod: $COLLECTOR_POD"
    kubectl exec -n "$NAMESPACE" "$COLLECTOR_POD" -- env | grep -E "OPENSTACK_ENDPOINT_TYPE|LOGIN_TYPE" || echo "환경변수를 찾을 수 없습니다."
else
    echo "Collector Pod를 찾을 수 없습니다."
fi

echo ""
echo "=== .env 파일 확인 ==="
if [ -f ".env" ]; then
    echo ".env 파일 내용:"
    grep -E "OPENSTACK_ENDPOINT_TYPE|LOGIN_TYPE" .env || echo "해당 변수를 찾을 수 없습니다."
else
    echo ".env 파일을 찾을 수 없습니다."
fi

