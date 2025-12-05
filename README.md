# OpenStack Monitoring System

오픈스택 환경의 전체 리소스(서버, 프로젝트, 네트워크, 하이퍼바이저, 플레이버, 볼륨)를 모니터링하고, VM부터 물리 호스트까지의 네트워크 토폴로지를 시각화하는 시스템입니다.

## Features

- **네트워크 토폴로지 시각화**: VM부터 물리 호스트까지의 전체 네트워크 경로 시각화
- **리소스 모니터링**: 서버, 프로젝트, 네트워크, 하이퍼바이저, 플레이버, 볼륨 모니터링
- **과거 데이터 조회**: 시간대별 메트릭 조회 및 트렌드 분석
- **실시간 업데이트**: 1분 간격 데이터 수집 및 실시간 대시보드 업데이트

## Architecture

- **Backend**: Go 1.21+ (gophercloud, Gin/Echo, GORM)
- **Frontend**: React/Vue.js (Cytoscape.js, Chart.js)
- **Database**: MariaDB 10.6+
- **Deployment**: Kubernetes + Helm

## Quick Start

자세한 시작 가이드는 다음 문서를 참조하세요:
- [빠른 시작 가이드](specs/001-openstack-monitoring/quickstart.md)
- [배포 가이드](DEPLOYMENT.md)
- [배포 스크립트](scripts/README.md)

### Prerequisites

- Go 1.21+
- MariaDB 10.6+
- Kubernetes cluster
- OpenStack environment access

### Local Development

```bash
# Backend
cd backend
go mod download
go run cmd/collector/main.go
go run cmd/api/main.go

# Frontend
cd frontend
npm install
npm start
```

## Project Structure

```
backend/
├── cmd/              # Application entry points
├── internal/         # Internal packages
├── pkg/              # Public packages
└── tests/            # Test files

frontend/
├── src/              # Source files
└── public/           # Static files
```

## Documentation

- [Feature Specification](specs/001-openstack-monitoring/spec.md)
- [Implementation Plan](specs/001-openstack-monitoring/plan.md)
- [Data Model](specs/001-openstack-monitoring/data-model.md)
- [API Contract](specs/001-openstack-monitoring/contracts/api.yaml)
- [Tasks](specs/001-openstack-monitoring/tasks.md)

## License

[Add your license here]

