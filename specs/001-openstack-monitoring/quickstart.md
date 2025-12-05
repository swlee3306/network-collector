# Quick Start Guide: 오픈스택 통합 모니터링 시스템

**Date**: 2025-12-05  
**Feature**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

## Overview

이 가이드는 오픈스택 통합 모니터링 시스템을 빠르게 시작하는 방법을 설명합니다.

## Prerequisites

- Kubernetes 클러스터 (v1.24+)
- Helm 3.x
- OpenStack 환경 접근 권한 (Keystone, Nova, Neutron, Cinder API)
- MariaDB 10.6+ (또는 Kubernetes에서 배포)

## Quick Start Steps

### 1. 저장소 클론 및 설정

```bash
git clone <repository-url>
cd network-collector
```

### 2. OpenStack 인증 정보 설정

Kubernetes Secret 생성:

```bash
kubectl create secret generic openstack-credentials \
  --from-literal=username=<openstack-username> \
  --from-literal=password=<openstack-password> \
  --from-literal=project-id=<openstack-project-id> \
  --from-literal=auth-url=<keystone-auth-url> \
  --from-literal=domain-name=<openstack-domain>
```

### 3. 데이터베이스 설정

MariaDB 배포 (또는 기존 MariaDB 사용):

```bash
# Helm을 사용한 MariaDB 배포
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install mariadb bitnami/mariadb \
  --set auth.rootPassword=<root-password> \
  --set auth.database=openstack_monitor \
  --set persistence.enabled=true
```

### 4. 애플리케이션 배포

Helm Chart로 배포:

```bash
cd deployments/helm/openstack-monitor
helm install openstack-monitor . \
  --set openstack.authUrl=<keystone-auth-url> \
  --set database.host=mariadb \
  --set database.name=openstack_monitor \
  --set database.user=root \
  --set database.password=<root-password>
```

### 5. 접근 확인

프론트엔드 접근:

```bash
# Ingress 또는 LoadBalancer를 통한 접근
kubectl get ingress openstack-monitor-frontend
```

또는 포트 포워딩:

```bash
kubectl port-forward svc/openstack-monitor-api 8080:80
# 브라우저에서 http://localhost:8080 접근
```

## Configuration

### Collector 서비스 설정

환경 변수:
- `OPENSTACK_AUTH_URL`: Keystone 인증 URL
- `OPENSTACK_USERNAME`: OpenStack 사용자명
- `OPENSTACK_PASSWORD`: OpenStack 비밀번호
- `OPENSTACK_PROJECT_ID`: OpenStack 프로젝트 ID
- `OPENSTACK_DOMAIN_NAME`: OpenStack 도메인 이름
- `DATABASE_URL`: MariaDB 연결 문자열
- `COLLECTION_INTERVAL`: 수집 주기 (기본: 1분)

### API 서비스 설정

환경 변수:
- `DATABASE_URL`: MariaDB 연결 문자열
- `JWT_SECRET`: JWT 토큰 서명 키
- `SSE_ENABLED`: Server-Sent Events 활성화 (기본: true)

### Frontend 설정

환경 변수:
- `REACT_APP_API_URL`: API 서비스 URL

## Development Setup

### 로컬 개발 환경

1. **Backend 개발**:

```bash
cd backend
go mod download
go run cmd/collector/main.go
go run cmd/api/main.go
```

2. **Frontend 개발**:

```bash
cd frontend
npm install
npm start
```

3. **데이터베이스 마이그레이션**:

```bash
cd backend
go run cmd/migrate/main.go up
```

## Testing

### 단위 테스트

```bash
cd backend
go test ./...
```

### 통합 테스트

```bash
cd backend
go test -tags=integration ./tests/integration/...
```

### API 테스트

```bash
# API 서버 실행 후
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

## Troubleshooting

### Collector가 데이터를 수집하지 않음

1. OpenStack 인증 정보 확인
2. 네트워크 연결 확인
3. Collector 로그 확인: `kubectl logs <collector-pod>`

### API가 응답하지 않음

1. 데이터베이스 연결 확인
2. API 서비스 로그 확인: `kubectl logs <api-pod>`
3. 헬스 체크 엔드포인트 확인: `curl http://localhost:8080/health`

### 토폴로지가 표시되지 않음

1. 네트워크 리소스 수집 확인
2. 토폴로지 분석 로그 확인
3. 프론트엔드 콘솔 오류 확인

## Next Steps

- [data-model.md](./data-model.md): 데이터 모델 상세
- [contracts/api.yaml](./contracts/api.yaml): API 계약
- [plan.md](./plan.md): 구현 계획

