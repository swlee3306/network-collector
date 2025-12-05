# Research: 오픈스택 통합 모니터링 시스템

**Date**: 2025-12-05  
**Feature**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

## Research Objectives

이 문서는 오픈스택 통합 모니터링 시스템 구현을 위한 기술 조사 결과를 정리합니다. 주요 연구 영역:
1. OpenStack API 클라이언트 라이브러리 (Go)
2. 네트워크 토폴로지 수집 방법
3. 프론트엔드 시각화 라이브러리 선택
4. 실시간 데이터 업데이트 전략

## 1. OpenStack API 클라이언트 라이브러리 (Go)

### Decision: gophercloud

**Rationale**:
- Go 언어용 공식 OpenStack SDK
- 모든 주요 OpenStack 서비스 지원 (Keystone, Nova, Neutron, Cinder, Glance)
- 활발한 커뮤니티와 문서화
- 프로덕션 환경에서 검증됨

**Alternatives Considered**:
- 직접 REST API 호출: 구현 복잡도 높음, 인증/에러 처리 직접 구현 필요
- 다른 Go 라이브러리: gophercloud가 가장 성숙하고 널리 사용됨

**Key Capabilities**:
- Keystone: 인증 및 토큰 관리
- Nova: 인스턴스(VM) 정보 수집
- Neutron: 네트워크 리소스 및 토폴로지 정보 수집
- Cinder: 볼륨 정보 수집
- Glance: 이미지 정보 (필요시)

**Implementation Notes**:
- 인증: v3 API 사용 (프로젝트 스코프)
- 에러 처리: gophercloud의 표준 에러 타입 활용
- 재시도 로직: exponential backoff 패턴 적용

## 2. 네트워크 토폴로지 수집 방법

### Decision: Neutron API를 통한 단계적 경로 추적

**Rationale**:
- OpenStack Neutron API는 네트워크 리소스의 연결 관계를 제공
- VM → Port → Network → Router → Physical Network 순서로 추적 가능
- Nova API와 Neutron API 조합으로 VM-호스트 매핑 가능

**Approach**:
1. **VM 정보 수집** (Nova API):
   - `GET /servers/{server_id}` - VM 상세 정보
   - `GET /servers/{server_id}/os-hypervisors` - 호스팅 하이퍼바이저 정보

2. **네트워크 포트 정보 수집** (Neutron API):
   - `GET /ports?device_id={server_id}` - VM에 연결된 포트
   - `GET /ports/{port_id}` - 포트 상세 정보 (네트워크 ID, 서브넷 ID)

3. **네트워크 경로 추적** (Neutron API):
   - `GET /networks/{network_id}` - 네트워크 정보
   - `GET /routers` - 라우터 정보 및 외부 네트워크 연결
   - `GET /agents` - 네트워크 에이전트 정보 (물리 호스트 매핑)

4. **물리 호스트 매핑**:
   - Nova API의 하이퍼바이저 정보와 Neutron 에이전트 정보를 결합
   - OVS (Open vSwitch) 에이전트 정보로 물리 네트워크 인터페이스 식별

**Challenges & Solutions**:
- **Challenge**: 일부 네트워크 요소 접근 불가
  - **Solution**: 부분 데이터 표시, 접근 불가능한 요소는 명시적 표시 (FR-007)
- **Challenge**: 동적 VM 생성/삭제
  - **Solution**: 1분 수집 주기에 자동 반영 (FR-006)
- **Challenge**: 복잡한 다중 경로
  - **Solution**: 모든 경로 수집 및 시각적 구분 (FR-007)

**Data Structure**:
- 그래프 모델: 노드(VM, Port, Network, Router, Host) + 엣지(연결 관계)
- MariaDB에 노드-엣지 테이블로 저장
- 조회 시 그래프 알고리즘으로 경로 탐색

## 3. 프론트엔드 시각화 라이브러리

### Decision: Cytoscape.js (네트워크 토폴로지) + Chart.js (메트릭)

**Rationale for Cytoscape.js**:
- 네트워크 그래프 시각화에 특화
- 대규모 그래프(10,000+ 노드) 성능 최적화
- 인터랙티브 기능 (줌, 팬, 노드 선택)
- 레이아웃 알고리즘 내장 (hierarchical, force-directed 등)
- 다중 경로 시각화 지원

**Alternatives Considered**:
- **D3.js**: 더 유연하지만 구현 복잡도 높음, 네트워크 그래프 전용 기능 부족
- **vis.js**: 네트워크 모듈 있으나 Cytoscape.js보다 기능 제한적
- **sigma.js**: 경량이지만 대규모 그래프 성능 이슈

**Rationale for Chart.js**:
- 메트릭 차트에 적합 (라인, 바, 파이 차트)
- 간단한 API, 빠른 구현
- 반응형 디자인 지원

**Implementation Strategy**:
- Cytoscape.js: 네트워크 토폴로지 시각화 전용
- Chart.js: CPU, 메모리, 트래픽 등 시계열 메트릭 차트
- React/Vue.js 컴포넌트로 래핑하여 재사용성 확보

## 4. 실시간 데이터 업데이트 전략

### Decision: Server-Sent Events (SSE)

**Rationale**:
- 단방향 서버 푸시로 충분 (클라이언트→서버 요청 불필요)
- HTTP 기반으로 구현 간단
- WebSocket보다 가벼움
- 자동 재연결 지원
- 1분 이내 업데이트 요구사항 충족 (FR-009)

**Alternatives Considered**:
- **WebSocket**: 양방향 통신이 필요 없어 과도함, 구현 복잡도 높음
- **폴링**: 서버 부하 증가, 실시간성 떨어짐
- **Long Polling**: SSE보다 복잡하고 효율성 낮음

**Implementation**:
- API 서비스에서 SSE 엔드포인트 제공: `GET /api/v1/events/stream`
- Collector 서비스가 데이터 수집 완료 시 이벤트 발행
- API 서비스가 이벤트를 SSE 클라이언트에게 브로드캐스트
- 프론트엔드에서 EventSource API로 구독

**Event Types**:
- `resource.updated`: 리소스 정보 업데이트
- `topology.updated`: 토폴로지 정보 업데이트
- `error`: 수집 오류 발생

## 5. 데이터베이스 스키마 설계

### Decision: 관계형 모델 + 시계열 최적화

**Rationale**:
- 리소스 간 관계 모델링 필요 (프로젝트-인스턴스, 인스턴스-네트워크 등)
- 시계열 데이터 조회 최적화 (인덱싱, 파티셔닝)
- MariaDB의 파티셔닝 기능으로 30일 이상 데이터 관리 용이

**Key Design Decisions**:
- **파티셔닝**: 날짜별 파티션으로 오래된 데이터 삭제 성능 최적화 (FR-017)
- **인덱싱**: 리소스 ID + 타임스탬프 복합 인덱스로 조회 성능 확보
- **정규화**: 리소스 정보는 정규화, 메트릭은 비정규화 (조회 성능 우선)

## 6. 오류 처리 및 재시도 전략

### Decision: 부분 실패/전체 실패 구분 + Exponential Backoff

**Rationale**:
- FR-012에 따라 일관된 오류 처리 전략 필요
- 부분 실패 시 사용자에게 최대한 정보 제공
- 전체 실패 시 명확한 오류 상태 표시

**Implementation**:
- **부분 실패**: 일부 리소스만 수집 실패 → 수집 성공한 부분 데이터 저장 및 표시
- **전체 실패**: 모든 리소스 수집 실패 → 오류 상태 표시, 이전 데이터 표시 안 함
- **재시도**: Exponential backoff (1초, 2초, 4초, 8초) 최대 3회
- **오픈스택 서비스 중단**: 백그라운드에서 주기적 재시도 계속

## 7. 성능 최적화 전략

### Decision: 캐싱 + 배치 처리 + 데이터베이스 최적화

**Rationale**:
- SC-001, SC-002, SC-007 성능 요구사항 충족
- 대규모 환경(10,000 VM)에서도 20% 이내 성능 저하 유지 (FR-014)

**Strategies**:
- **토폴로지 캐싱**: 자주 조회되는 토폴로지는 메모리 캐시 (Redis 선택사항)
- **배치 수집**: 여러 리소스를 배치로 수집하여 API 호출 최소화
- **데이터베이스 최적화**: 
  - 파티셔닝으로 조회 범위 제한
  - 적절한 인덱스 설계
  - 연결 풀링
- **비동기 처리**: 토폴로지 분석은 백그라운드 작업으로 처리

## 8. 프론트엔드 프레임워크 선택

### Decision: React (권장) 또는 Vue.js

**Rationale**:
- 둘 다 네트워크 시각화 라이브러리와 잘 통합됨
- 컴포넌트 기반 아키텍처로 재사용성 확보
- 풍부한 생태계

**Recommendation**: React
- 더 큰 커뮤니티와 생태계
- Cytoscape.js와의 통합 예제 풍부
- TypeScript 지원 우수

**Alternatives Considered**:
- **Vue.js**: 더 간단한 학습 곡선, 하지만 생태계 규모는 React보다 작음
- **Angular**: 과도한 복잡도, 이 프로젝트에는 불필요

## 9. Kubernetes 배포 전략

### Decision: Helm Charts + StatefulSet (MariaDB) + Deployment (서비스)

**Rationale**:
- Helm으로 배포 관리 간소화
- StatefulSet으로 MariaDB 영구 저장소 보장
- Deployment로 서비스 수평 확장

**Key Components**:
- **Collector**: Deployment + CronJob (1분 간격)
- **API**: Deployment + Service (ClusterIP/LoadBalancer)
- **Frontend**: Deployment + Service (Ingress)
- **MariaDB**: StatefulSet + PersistentVolumeClaim

## 10. 관찰 가능성 (Observability)

### Decision: 구조화된 로깅 + Prometheus 메트릭

**Rationale**:
- 운영 환경에서 디버깅 및 모니터링 필수
- 성능 요구사항 검증 (SC-003: 99% 수집 성공률)

**Implementation**:
- **로깅**: JSON 형식 구조화 로깅 (logrus 또는 zap)
- **메트릭**: Prometheus 형식
  - 수집 성공/실패 횟수
  - API 응답 시간 (p50, p95, p99)
  - 데이터베이스 쿼리 시간
- **트레이싱**: (선택사항) OpenTelemetry

## Summary

모든 기술 선택이 명세 요구사항과 제약사항을 충족합니다:
- ✅ Go 언어 및 gophercloud로 OpenStack API 통합
- ✅ Neutron API를 통한 네트워크 토폴로지 수집
- ✅ Cytoscape.js로 네트워크 시각화
- ✅ SSE로 실시간 데이터 업데이트
- ✅ MariaDB로 관계형 데이터 + 시계열 데이터 저장
- ✅ Kubernetes + Helm으로 배포

**Next Steps**: Phase 1에서 상세 데이터 모델 및 API 계약 정의

