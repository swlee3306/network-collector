# Quick Start Guide: 오픈스택 통합 모니터링 시스템

**Date**: 2025-12-05  
**Feature**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

## Overview

이 가이드는 오픈스택 통합 모니터링 시스템을 빠르게 시작하는 방법을 설명합니다.

## Prerequisites

- Kubernetes 클러스터 (v1.19+)
- kubectl 설치 및 클러스터 접근 권한
- Helm 3.0+ (Helm 배포 시)
- Docker (이미지 빌드 시)
- OpenStack 환경 접근 권한 (Keystone, Nova, Neutron, Cinder API)
- MariaDB 10.6+ (또는 Kubernetes에서 배포)

## Quick Start Steps

### 방법 1: 배포 스크립트 사용 (권장)

가장 간단한 방법은 제공된 배포 스크립트를 사용하는 것입니다.

#### 1. 이미지 빌드 (선택적)

이미지를 빌드하지 않고 배포하려면, 이미 빌드된 이미지가 레지스트리에 있어야 합니다.

```bash
# 모든 이미지 빌드
./scripts/build-images.sh [registry] [tag]

# 예시: 로컬 이미지 빌드
./scripts/build-images.sh

# 예시: 레지스트리에 푸시할 이미지 빌드
./scripts/build-images.sh your-registry.io latest
```

#### 2. 배포 실행

```bash
# Helm을 사용한 배포 (권장)
./scripts/deploy.sh helm [namespace]

# 또는 Kubernetes 매니페스트를 사용한 배포
./scripts/deploy.sh k8s [namespace]
```

스크립트가 다음 정보를 대화형으로 요청합니다:
- DB_PASSWORD
- OPENSTACK_AUTH_URL
- OPENSTACK_USERNAME
- OPENSTACK_PASSWORD
- OPENSTACK_PROJECT_ID (UUID 형식)
- JWT_SECRET
- AUTH_TOKEN

### 방법 2: Helm Chart 직접 사용

#### 1. OpenStack 인증 정보 확인

프로젝트 ID는 UUID 형식이어야 합니다. 프로젝트 이름이 아닙니다.

```bash
# OpenStack CLI로 프로젝트 ID 확인
openstack project list
```

#### 2. Helm Chart로 배포

```bash
cd backend/deployments/helm

helm install network-collector . \
  --set secrets.dbPassword='your-db-password' \
  --set secrets.openstackAuthURL='http://keystone:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='your-openstack-password' \
  --set secrets.openstackProjectID='your-project-id-uuid' \
  --set secrets.jwtSecret='your-jwt-secret' \
  --set secrets.authToken='your-auth-token' \
  --set collector.image.repository='your-registry/network-collector' \
  --set collector.image.tag='latest' \
  --set api.image.repository='your-registry/network-collector-api' \
  --set api.image.tag='latest' \
  --set frontend.image.repository='your-registry/network-collector-frontend' \
  --set frontend.image.tag='latest'
```

**예시:**
```bash
cd backend/deployments/helm

helm install network-collector . \
  --set secrets.dbPassword='root' \
  --set secrets.openstackAuthURL='http://192.168.219.121:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='lsu95080$' \
  --set secrets.openstackProjectID='4c1c2921a66043809a0e6f05c323fa0a' \
  --set secrets.jwtSecret='change-me-in-production' \
  --set secrets.authToken='change-me-in-production'
```

#### 3. values.yaml 파일 사용 (선택적)

```bash
# values.yaml 파일 편집
vim backend/deployments/helm/values.yaml

# 설치
helm install network-collector backend/deployments/helm -f backend/deployments/helm/values.yaml \
  --set secrets.dbPassword='your-db-password' \
  --set secrets.openstackAuthURL='http://keystone:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='your-openstack-password' \
  --set secrets.openstackProjectID='your-project-id-uuid' \
  --set secrets.jwtSecret='your-jwt-secret' \
  --set secrets.authToken='your-auth-token'
```

### 방법 3: Kubernetes 매니페스트 직접 사용

#### 1. Secret 생성

```bash
kubectl create secret generic network-collector-secrets \
  --from-literal=DB_PASSWORD='your-db-password' \
  --from-literal=OPENSTACK_AUTH_URL='http://keystone:5000/v3' \
  --from-literal=OPENSTACK_USERNAME='admin' \
  --from-literal=OPENSTACK_PASSWORD='your-openstack-password' \
  --from-literal=OPENSTACK_PROJECT_ID='your-project-id-uuid' \
  --from-literal=JWT_SECRET='your-jwt-secret' \
  --from-literal=AUTH_TOKEN='your-auth-token'
```

**예시:**
```bash
kubectl create secret generic network-collector-secrets \
  --from-literal=DB_PASSWORD='root' \
  --from-literal=OPENSTACK_AUTH_URL='http://192.168.219.121:5000/v3' \
  --from-literal=OPENSTACK_USERNAME='admin' \
  --from-literal=OPENSTACK_PASSWORD='lsu95080$' \
  --from-literal=OPENSTACK_PROJECT_ID='4c1c2921a66043809a0e6f05c323fa0a' \
  --from-literal=JWT_SECRET='change-me-in-production' \
  --from-literal=AUTH_TOKEN='change-me-in-production'
```

**중요:** `OPENSTACK_PROJECT_ID`는 프로젝트 이름이 아닌 **UUID 형식의 프로젝트 ID**를 사용해야 합니다.

#### 2. ConfigMap 생성

```bash
kubectl apply -f backend/deployments/k8s/configmap.yaml
```

#### 3. MariaDB 배포

```bash
kubectl apply -f backend/deployments/k8s/mariadb/statefulset.yaml

# MariaDB가 준비될 때까지 대기
kubectl wait --for=condition=ready pod -l app=mariadb --timeout=300s
```

#### 4. Collector 배포

```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
```

#### 5. API Service 배포

```bash
kubectl apply -f backend/deployments/k8s/api/deployment.yaml
```

#### 6. Frontend 배포

```bash
kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml
```

#### 7. Ingress 배포 (선택적)

```bash
kubectl apply -f backend/deployments/k8s/ingress.yaml
```

## 배포 확인

### Pod 상태 확인

```bash
kubectl get pods -l app=network-collector
kubectl get pods -l app=network-collector-api
kubectl get pods -l app=network-collector-frontend
kubectl get pods -l app=mariadb
```

### 서비스 확인

```bash
kubectl get services
```

### 로그 확인

```bash
# Collector 로그
kubectl logs -f deployment/network-collector

# API 로그
kubectl logs -f deployment/network-collector-api

# Frontend 로그
kubectl logs -f deployment/network-collector-frontend

# MariaDB 로그
kubectl logs -f statefulset/mariadb
```

## 접근 방법

### 포트 포워딩을 통한 접근

```bash
# API 서비스 포트 포워딩
kubectl port-forward svc/network-collector-api 8080:8080

# Frontend 서비스 포트 포워딩
kubectl port-forward svc/network-collector-frontend 3000:80
```

브라우저에서 접근:
- API: http://localhost:8080
- Frontend: http://localhost:3000

### Ingress를 통한 접근

Ingress가 설정된 경우:

```bash
# Ingress 확인
kubectl get ingress network-collector-ingress

# Ingress 호스트 확인
kubectl get ingress network-collector-ingress -o jsonpath='{.spec.rules[0].host}'
```

## Configuration

### Collector 서비스 환경 변수

| 변수명 | 설명 | 필수 | 기본값 |
|--------|------|------|--------|
| `DB_HOST` | 데이터베이스 호스트 | 예 | - |
| `DB_PORT` | 데이터베이스 포트 | 아니오 | 3306 |
| `DB_USER` | 데이터베이스 사용자 | 예 | - |
| `DB_PASSWORD` | 데이터베이스 비밀번호 | 예 | - |
| `DB_NAME` | 데이터베이스 이름 | 예 | openstack_monitor |
| `OPENSTACK_AUTH_URL` | Keystone 인증 URL | 예 | - |
| `OPENSTACK_USERNAME` | OpenStack 사용자명 | 예 | - |
| `OPENSTACK_PASSWORD` | OpenStack 비밀번호 | 예 | - |
| `OPENSTACK_PROJECT_ID` | OpenStack 프로젝트 ID (UUID) | 예 | - |
| `OPENSTACK_DOMAIN_NAME` | OpenStack 도메인 이름 | 아니오 | default |
| `LOG_LEVEL` | 로그 레벨 | 아니오 | info |
| `ENVIRONMENT` | 환경 (development/production) | 아니오 | development |

### API 서비스 환경 변수

| 변수명 | 설명 | 필수 | 기본값 |
|--------|------|------|--------|
| `DB_HOST` | 데이터베이스 호스트 | 예 | - |
| `DB_PORT` | 데이터베이스 포트 | 아니오 | 3306 |
| `DB_USER` | 데이터베이스 사용자 | 예 | - |
| `DB_PASSWORD` | 데이터베이스 비밀번호 | 예 | - |
| `DB_NAME` | 데이터베이스 이름 | 예 | openstack_monitor |
| `SERVER_PORT` | API 서버 포트 | 아니오 | 8080 |
| `SSE_ENABLED` | Server-Sent Events 활성화 | 아니오 | true |
| `JWT_SECRET` | JWT 토큰 서명 키 | 예 | - |
| `AUTH_TOKEN` | API 인증 토큰 | 예 | - |
| `LOG_LEVEL` | 로그 레벨 | 아니오 | info |
| `ENVIRONMENT` | 환경 (development/production) | 아니오 | development |

### Frontend 환경 변수

| 변수명 | 설명 | 필수 | 기본값 |
|--------|------|------|--------|
| `REACT_APP_API_URL` | API 서비스 URL | 예 | - |

## Development Setup

### 로컬 개발 환경

#### 1. Backend 개발

```bash
cd backend
go mod download

# 데이터베이스 마이그레이션
go run cmd/migrate/main.go

# Collector 실행
go run cmd/collector/main.go

# API 서버 실행 (별도 터미널)
go run cmd/api/main.go
```

#### 2. Frontend 개발

```bash
cd frontend
npm install
npm start
```

#### 3. 환경 변수 설정

Backend 실행 전에 다음 환경 변수를 설정하세요:

```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your-password
export DB_NAME=openstack_monitor
export OPENSTACK_AUTH_URL=http://keystone:5000/v3
export OPENSTACK_USERNAME=admin
export OPENSTACK_PASSWORD=your-password
export OPENSTACK_PROJECT_ID=your-project-id-uuid
export OPENSTACK_DOMAIN_NAME=Default
export JWT_SECRET=your-jwt-secret
export AUTH_TOKEN=your-auth-token
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

통합 테스트를 실행하기 전에 테스트 데이터베이스를 설정해야 합니다:

```bash
export TEST_DB_HOST=localhost
export TEST_DB_PORT=3306
export TEST_DB_USER=root
export TEST_DB_PASSWORD=your-password
export TEST_DB_NAME=openstack_monitor_test
```

### 성능 테스트

```bash
cd backend
go test -tags=performance ./tests/performance/...
```

성능 테스트를 실행하기 전에 성능 테스트 데이터베이스를 설정해야 합니다:

```bash
export PERF_DB_HOST=localhost
export PERF_DB_PORT=3306
export PERF_DB_USER=root
export PERF_DB_PASSWORD=your-password
export PERF_DB_NAME=openstack_monitor_perf
```

### API 테스트

```bash
# API 서버 실행 후
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# 헬스 체크
curl http://localhost:8080/health
```

## 업그레이드

### Helm Chart 업그레이드

```bash
cd backend/deployments/helm

helm upgrade network-collector . \
  --set secrets.dbPassword='your-db-password' \
  --set secrets.openstackAuthURL='http://keystone:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='your-openstack-password' \
  --set secrets.openstackProjectID='your-project-id-uuid' \
  --set secrets.jwtSecret='your-jwt-secret' \
  --set secrets.authToken='your-auth-token'
```

## 제거

### 배포 스크립트 사용

```bash
./scripts/undeploy.sh helm [namespace]
# 또는
./scripts/undeploy.sh k8s [namespace]
```

### Helm Chart 제거

```bash
helm uninstall network-collector
```

### Kubernetes 매니페스트 제거

```bash
kubectl delete -f backend/deployments/k8s/frontend/deployment.yaml
kubectl delete -f backend/deployments/k8s/api/deployment.yaml
kubectl delete -f backend/deployments/k8s/collector/deployment.yaml
kubectl delete -f backend/deployments/k8s/mariadb/statefulset.yaml
kubectl delete -f backend/deployments/k8s/configmap.yaml
kubectl delete secret network-collector-secrets
```

## Troubleshooting

### Collector가 데이터를 수집하지 않음

1. **OpenStack 인증 정보 확인**
   ```bash
   kubectl get secret network-collector-secrets -o jsonpath='{.data.OPENSTACK_AUTH_URL}' | base64 -d
   kubectl get secret network-collector-secrets -o jsonpath='{.data.OPENSTACK_PROJECT_ID}' | base64 -d
   ```

2. **프로젝트 ID 형식 확인**
   - 프로젝트 ID는 UUID 형식이어야 합니다 (예: `4c1c2921a66043809a0e6f05c323fa0a`)
   - 프로젝트 이름이 아닙니다

3. **네트워크 연결 확인**
   ```bash
   kubectl exec -it <collector-pod> -- curl <openstack-auth-url>
   ```

4. **Collector 로그 확인**
   ```bash
   kubectl logs -f deployment/network-collector
   ```

### API가 응답하지 않음

1. **데이터베이스 연결 확인**
   ```bash
   kubectl get pods -l app=mariadb
   kubectl logs -l app=mariadb
   ```

2. **API 서비스 로그 확인**
   ```bash
   kubectl logs -f deployment/network-collector-api
   ```

3. **헬스 체크 엔드포인트 확인**
   ```bash
   kubectl port-forward svc/network-collector-api 8080:8080
   curl http://localhost:8080/health
   ```

### 토폴로지가 표시되지 않음

1. **네트워크 리소스 수집 확인**
   ```bash
   kubectl logs deployment/network-collector | grep -i network
   ```

2. **토폴로지 분석 로그 확인**
   ```bash
   kubectl logs deployment/network-collector | grep -i topology
   ```

3. **프론트엔드 콘솔 오류 확인**
   - 브라우저 개발자 도구에서 네트워크 탭과 콘솔 확인

### Pod가 시작되지 않음

1. **이미지 Pull 실패 확인**
   ```bash
   kubectl describe pod <pod-name>
   ```

2. **Secret/ConfigMap 확인**
   ```bash
   kubectl get secret network-collector-secrets
   kubectl get configmap network-collector-config
   ```

3. **리소스 부족 확인**
   ```bash
   kubectl describe node
   ```

## Next Steps

- [배포 가이드](../DEPLOYMENT.md): 상세한 배포 가이드
- [데이터 모델](./data-model.md): 데이터 모델 상세
- [API 계약](./contracts/api.yaml): API 계약 문서
- [구현 계획](./plan.md): 구현 계획
- [배포 스크립트](../scripts/README.md): 배포 스크립트 사용 가이드
