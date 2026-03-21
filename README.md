# network-collector

## 한 줄 소개
OpenStack 자원, 메트릭, 네트워크 토폴로지를 수집하고 조회하는 모니터링 시스템입니다.

## 저장소 성격
- 분류: 플랫폼 / 인프라
- 목적: OpenStack 자원 수집, 메트릭 조회, 네트워크 토폴로지 가시화
- 핵심 기술: Go, React, OpenStack, Gin, GORM, MariaDB, Kubernetes, Helm

## 현재 구현 범위
- OpenStack 자원 수집
  - instances
  - projects
  - networks
  - hypervisors
  - flavors
  - volumes
  - ports
  - routers
- 인스턴스, 네트워크, 하이퍼바이저 메트릭 수집과 조회
- VM 기준 / Host 기준 / Network 기준 토폴로지 조회
- 로그인, 보호된 API, SSE 이벤트 스트림
- React 기반 대시보드와 상세 페이지
- Kubernetes 매니페스트와 Helm chart 기반 배포

## 구성

### Backend
- Go 기반 API 서버
- Go 기반 collector 서비스
- MariaDB 저장소
- OpenStack API 연동
- Prometheus `/metrics` 엔드포인트

### Frontend
- React + TypeScript
- Dashboard, Resource List, Detail, Metrics, Topology 화면
- Cytoscape.js 기반 토폴로지 시각화
- Chart.js 기반 메트릭 차트

## 실행 방법

### 1. 저장소 클론
```bash
git clone https://github.com/swlee3306/network-collector.git
cd network-collector
```

### 2. 환경 변수 준비
`.env.example`을 참고해 필요한 값을 준비합니다.

주요 값:
```bash
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=root
export DB_PASSWORD=your-password
export DB_NAME=openstack_monitor

export OPENSTACK_AUTH_URL=http://<keystone>:5000/v3
export OPENSTACK_USERNAME=admin
export OPENSTACK_PASSWORD=secret
export OPENSTACK_PROJECT_ID=<project-id>
export OPENSTACK_DOMAIN_NAME=default
export OPENSTACK_ENDPOINT_TYPE=public

export JWT_SECRET=change-me
export AUTH_TOKEN=change-me
export LOGIN_TYPE=internal
export SERVER_PORT=8080
```

### 3. Backend 실행
```bash
cd backend
go mod download

# API 서버
go run ./cmd/api

# Collector
go run ./cmd/collector
```

### 4. Frontend 실행
```bash
cd frontend
npm install
npm start
```

## 주요 엔드포인트

### 공용 엔드포인트
```http
GET /health
GET /metrics
POST /api/v1/auth/login
```

### 보호된 리소스 엔드포인트
```http
GET /api/v1/instances
GET /api/v1/instances/:id
GET /api/v1/projects
GET /api/v1/projects/compare
GET /api/v1/projects/:id
GET /api/v1/projects/:id/summary
GET /api/v1/networks
GET /api/v1/hypervisors
GET /api/v1/flavors
GET /api/v1/volumes
```

### 토폴로지 / 메트릭 / 이벤트
```http
GET /api/v1/topology/instances/:id
GET /api/v1/topology/hosts/:id
GET /api/v1/topology/networks/:id

GET /api/v1/metrics/instances/:id
GET /api/v1/metrics/networks/:id
GET /api/v1/metrics/hypervisors/:id

GET /api/v1/events/stream
```

## 프로젝트 구조
```text
network-collector/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   ├── collector/
│   │   └── migrate/
│   ├── internal/
│   ├── pkg/
│   └── tests/
├── frontend/
│   ├── src/
│   └── public/
├── deployments/
│   ├── k8s/
│   └── helm/
├── docs/
├── scripts/
└── specs/
```

## 배포
- Kubernetes 매니페스트는 `deployments/k8s` 아래에 있습니다.
- Helm chart는 `deployments/helm` 아래에 있습니다.
- 보조 스크립트는 `scripts/`에 정리돼 있습니다.

자세한 내용:
- [배포 가이드](DEPLOYMENT.md)
- [스크립트 안내](scripts/README.md)

## 검증
```bash
cd backend && go test ./...
cd frontend && npm run build
```

현재 frontend build는 성공하지만 ESLint 경고가 일부 남아 있습니다.

## 핀 저장소용 설명 문구
OpenStack 자원, 메트릭, 네트워크 토폴로지를 수집하고 시각화하는 모니터링 시스템
