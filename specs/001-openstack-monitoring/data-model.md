# Data Model: 오픈스택 통합 모니터링 시스템

**Date**: 2025-12-05  
**Feature**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

## Overview

이 문서는 오픈스택 통합 모니터링 시스템의 데이터 모델을 정의합니다. 관계형 데이터베이스(MariaDB)에 저장되며, 시계열 데이터 조회에 최적화됩니다.

## Core Entities

### 1. Instance (서버 인스턴스)

**Purpose**: 오픈스택에서 실행 중인 가상 머신 정보

**Attributes**:
- `id` (UUID, PK): 인스턴스 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 인스턴스 ID
- `name` (String): 인스턴스 이름
- `status` (Enum): 상태 (ACTIVE, SHUTOFF, ERROR, etc.)
- `project_id` (UUID, FK → Project): 소속 프로젝트
- `flavor_id` (UUID, FK → Flavor): 사용 중인 플레이버
- `hypervisor_id` (UUID, FK → Hypervisor): 호스팅 하이퍼바이저
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Project, Flavor, Hypervisor
- Has many: InstanceMetrics, TopologyNodes (as VM node)

**Validation Rules**:
- `openstack_id`는 필수이며 고유해야 함
- `status`는 OpenStack 표준 상태 값만 허용

### 2. Project

**Purpose**: 오픈스택 리소스를 그룹화하는 단위

**Attributes**:
- `id` (UUID, PK): 프로젝트 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 프로젝트 ID
- `name` (String): 프로젝트 이름
- `description` (String, Optional): 프로젝트 설명
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Has many: Instances, Networks, Volumes

**Validation Rules**:
- `openstack_id`는 필수이며 고유해야 함

### 3. Network

**Purpose**: 네트워크 리소스 정보

**Attributes**:
- `id` (UUID, PK): 네트워크 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 네트워크 ID
- `name` (String): 네트워크 이름
- `status` (String): 네트워크 상태
- `project_id` (UUID, FK → Project, Optional): 소속 프로젝트
- `shared` (Boolean): 공유 네트워크 여부
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Project (optional)
- Has many: Subnets, Ports

### 4. Subnet

**Purpose**: 서브넷 정보

**Attributes**:
- `id` (UUID, PK): 서브넷 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 서브넷 ID
- `network_id` (UUID, FK → Network): 소속 네트워크
- `name` (String): 서브넷 이름
- `cidr` (String): CIDR 주소
- `gateway_ip` (String, Optional): 게이트웨이 IP
- `created_at` (Timestamp): 생성 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Network

### 5. Port

**Purpose**: 네트워크 포트 정보 (VM과 네트워크 연결)

**Attributes**:
- `id` (UUID, PK): 포트 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 포트 ID
- `network_id` (UUID, FK → Network): 소속 네트워크
- `device_id` (UUID, Optional): 연결된 디바이스 ID (VM ID)
- `device_owner` (String): 디바이스 소유자 타입
- `mac_address` (String): MAC 주소
- `status` (String): 포트 상태
- `fixed_ips` (JSON): 고정 IP 주소 목록
- `created_at` (Timestamp): 생성 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Network
- Belongs to: Instance (via device_id, optional)
- Has one: TopologyNode (as port node)

**Validation Rules**:
- `device_id`가 있으면 유효한 Instance를 참조해야 함

### 6. Router

**Purpose**: 라우터 정보

**Attributes**:
- `id` (UUID, PK): 라우터 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 라우터 ID
- `name` (String): 라우터 이름
- `status` (String): 라우터 상태
- `external_gateway_info` (JSON): 외부 게이트웨이 정보
- `created_at` (Timestamp): 생성 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Has many: TopologyNodes (as router node)

### 7. Hypervisor

**Purpose**: 하이퍼바이저 정보

**Attributes**:
- `id` (UUID, PK): 하이퍼바이저 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 하이퍼바이저 ID
- `hostname` (String, UNIQUE): 호스트 이름
- `host_ip` (String): 호스트 IP 주소
- `status` (String): 하이퍼바이저 상태
- `state` (String): 하이퍼바이저 상태 (up/down)
- `vcpus_used` (Integer): 사용 중인 vCPU 수
- `vcpus_total` (Integer): 전체 vCPU 수
- `memory_used` (BigInteger): 사용 중인 메모리 (MB)
- `memory_total` (BigInteger): 전체 메모리 (MB)
- `local_gb_used` (BigInteger): 사용 중인 디스크 (GB)
- `local_gb_total` (BigInteger): 전체 디스크 (GB)
- `running_vms` (Integer): 실행 중인 VM 수
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Has many: Instances
- Has one: TopologyNode (as host node)

### 8. Flavor

**Purpose**: VM 인스턴스 타입 정의

**Attributes**:
- `id` (UUID, PK): 플레이버 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 플레이버 ID
- `name` (String, UNIQUE): 플레이버 이름
- `vcpus` (Integer): vCPU 개수
- `ram` (Integer): 메모리 크기 (MB)
- `disk` (Integer): 디스크 크기 (GB)
- `ephemeral` (Integer): 임시 디스크 크기 (GB)
- `swap` (Integer): 스왑 크기 (MB)
- `is_public` (Boolean): 공개 플레이버 여부
- `created_at` (Timestamp): 생성 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Has many: Instances

### 9. Volume

**Purpose**: 영구 스토리지 볼륨

**Attributes**:
- `id` (UUID, PK): 볼륨 고유 식별자
- `openstack_id` (String, UNIQUE): OpenStack 볼륨 ID
- `name` (String): 볼륨 이름
- `status` (String): 볼륨 상태
- `size` (Integer): 볼륨 크기 (GB)
- `volume_type` (String): 볼륨 타입
- `project_id` (UUID, FK → Project): 소속 프로젝트
- `attached_to` (UUID, FK → Instance, Optional): 연결된 인스턴스
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Project
- Belongs to: Instance (optional, via attached_to)

### 10. TopologyNode

**Purpose**: 네트워크 토폴로지 노드 (VM, 포트, 네트워크, 라우터, 호스트)

**Attributes**:
- `id` (UUID, PK): 노드 고유 식별자
- `node_type` (Enum): 노드 타입 (VM, PORT, NETWORK, ROUTER, HOST, UNKNOWN)
- `openstack_id` (String, Optional): OpenStack 리소스 ID
- `name` (String): 노드 이름
- `instance_id` (UUID, FK → Instance, Optional): VM 노드인 경우 인스턴스 참조
- `port_id` (UUID, FK → Port, Optional): 포트 노드인 경우 포트 참조
- `network_id` (UUID, FK → Network, Optional): 네트워크 노드인 경우 네트워크 참조
- `router_id` (UUID, FK → Router, Optional): 라우터 노드인 경우 라우터 참조
- `hypervisor_id` (UUID, FK → Hypervisor, Optional): 호스트 노드인 경우 하이퍼바이저 참조
- `metadata` (JSON): 추가 메타데이터
- `is_accessible` (Boolean): 접근 가능 여부 (false인 경우 "알 수 없음" 표시)
- `created_at` (Timestamp): 생성 시간
- `updated_at` (Timestamp): 마지막 업데이트 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: Instance, Port, Network, Router, Hypervisor (optional, node_type에 따라)
- Has many: TopologyEdges (as source or target)

**Validation Rules**:
- `node_type`에 따라 해당 FK가 설정되어야 함
- `is_accessible`가 false인 경우 명시적으로 표시 (FR-007)

### 11. TopologyEdge

**Purpose**: 네트워크 토폴로지 연결 관계

**Attributes**:
- `id` (UUID, PK): 엣지 고유 식별자
- `source_node_id` (UUID, FK → TopologyNode): 출발 노드
- `target_node_id` (UUID, FK → TopologyNode): 도착 노드
- `edge_type` (String): 연결 타입 (VIRTUAL, PHYSICAL, etc.)
- `metadata` (JSON): 추가 메타데이터 (대역폭, 지연 등)
- `created_at` (Timestamp): 생성 시간
- `collected_at` (Timestamp): 데이터 수집 시간

**Relationships**:
- Belongs to: TopologyNode (source, target)

**Validation Rules**:
- `source_node_id`와 `target_node_id`는 서로 달라야 함
- 루프 허용 (다중 경로 지원)

**Indexes**:
- (source_node_id, target_node_id): 경로 탐색 성능 최적화

### 12. InstanceMetrics

**Purpose**: 인스턴스별 시계열 메트릭 데이터

**Attributes**:
- `id` (UUID, PK): 메트릭 고유 식별자
- `instance_id` (UUID, FK → Instance): 인스턴스 참조
- `timestamp` (Timestamp): 메트릭 수집 시간
- `cpu_usage_percent` (Float): CPU 사용률 (%)
- `memory_usage_mb` (BigInteger): 메모리 사용량 (MB)
- `memory_total_mb` (BigInteger): 전체 메모리 (MB)
- `disk_read_bytes` (BigInteger): 디스크 읽기 바이트
- `disk_write_bytes` (BigInteger): 디스크 쓰기 바이트
- `network_rx_bytes` (BigInteger): 네트워크 수신 바이트
- `network_tx_bytes` (BigInteger): 네트워크 송신 바이트
- `created_at` (Timestamp): 레코드 생성 시간

**Relationships**:
- Belongs to: Instance

**Indexes**:
- (instance_id, timestamp): 시계열 조회 성능 최적화
- (timestamp): 파티셔닝 및 삭제 성능 최적화

**Partitioning**:
- 날짜별 파티션 (월별 또는 주별)
- 30일 이후 파티션 자동 삭제 (FR-017)

### 13. NetworkMetrics

**Purpose**: 네트워크별 시계열 메트릭 데이터

**Attributes**:
- `id` (UUID, PK): 메트릭 고유 식별자
- `network_id` (UUID, FK → Network): 네트워크 참조
- `timestamp` (Timestamp): 메트릭 수집 시간
- `total_bytes` (BigInteger): 총 트래픽 바이트
- `packets_dropped` (Integer): 드롭된 패킷 수
- `connected_vms_count` (Integer): 연결된 VM 수
- `created_at` (Timestamp): 레코드 생성 시간

**Relationships**:
- Belongs to: Network

**Indexes**:
- (network_id, timestamp): 시계열 조회 성능 최적화
- (timestamp): 파티셔닝 및 삭제 성능 최적화

### 14. HypervisorMetrics

**Purpose**: 하이퍼바이저별 시계열 메트릭 데이터

**Attributes**:
- `id` (UUID, PK): 메트릭 고유 식별자
- `hypervisor_id` (UUID, FK → Hypervisor): 하이퍼바이저 참조
- `timestamp` (Timestamp): 메트릭 수집 시간
- `vcpus_used` (Integer): 사용 중인 vCPU 수
- `vcpus_total` (Integer): 전체 vCPU 수
- `memory_used_mb` (BigInteger): 사용 중인 메모리 (MB)
- `memory_total_mb` (BigInteger): 전체 메모리 (MB)
- `running_vms_count` (Integer): 실행 중인 VM 수
- `created_at` (Timestamp): 레코드 생성 시간

**Relationships**:
- Belongs to: Hypervisor

**Indexes**:
- (hypervisor_id, timestamp): 시계열 조회 성능 최적화
- (timestamp): 파티셔닝 및 삭제 성능 최적화

## Data Retention & Cleanup

### Retention Policy

- **최소 보관 기간**: 30일 (FR-015, SC-005)
- **자동 삭제**: FIFO 방식 (FR-017)
- **파티셔닝**: 날짜별 파티션으로 삭제 성능 최적화

### Cleanup Strategy

1. **메트릭 데이터**: 30일 이후 자동 삭제
2. **리소스 데이터**: 최신 상태만 유지 (이력 관리 안 함)
3. **토폴로지 데이터**: 최신 상태만 유지, 오래된 엣지는 정리

## Data Collection Lifecycle

### Initial Collection

1. Collector가 OpenStack API 호출
2. 리소스 정보 수집 (Instances, Networks, Hypervisors, etc.)
3. 네트워크 토폴로지 분석 및 노드/엣지 생성
4. 메트릭 데이터 저장

### Update Cycle (1분 간격)

1. 기존 리소스 업데이트
2. 새로 생성된 리소스 추가
3. 삭제된 리소스 마킹 (soft delete 또는 삭제)
4. 토폴로지 재분석 (변경사항 반영)
5. 메트릭 데이터 추가

### Dynamic VM Handling

- 생성/삭제된 VM은 다음 수집 주기(1분)에 자동 반영 (FR-006)
- 토폴로지 노드/엣지도 자동 업데이트

## Query Patterns

### Common Queries

1. **리소스 목록 조회**: 최신 상태만 조회 (collected_at 최신)
2. **과거 메트릭 조회**: 시간 범위 지정, 인덱스 활용 (SC-007: 3초 이내)
3. **토폴로지 경로 탐색**: 그래프 알고리즘 (BFS/DFS), 인덱스 활용 (SC-001: 5초 이내)
4. **프로젝트별 집계**: JOIN 및 GROUP BY 활용

### Performance Considerations

- **인덱싱**: 모든 FK, 타임스탬프, openstack_id에 인덱스
- **파티셔닝**: 메트릭 테이블은 날짜별 파티션
- **캐싱**: 자주 조회되는 토폴로지는 메모리 캐시 고려

