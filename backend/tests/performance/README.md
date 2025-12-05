# 성능 테스트

이 디렉토리에는 OpenStack 모니터링 시스템의 성능 테스트가 포함되어 있습니다.

## 성능 요구사항

### SC-001: VM 토폴로지 조회 성능
- **요구사항**: VM을 선택한 후 5초 이내에 해당 VM의 네트워크 토폴로지가 시각적으로 표시
- **테스트**: `TestTopologyQueryPerformance_SC001`

### SC-002: 대규모 환경 토폴로지 조회 성능
- **요구사항**: 최대 10,000개의 VM을 가진 오픈스택 환경에서도 네트워크 토폴로지 조회 시 10초 이내에 응답
- **테스트**: `TestTopologyQueryPerformance_SC002`, `TestLargeScaleEnvironment_10000VMs`

### SC-007: 과거 메트릭 조회 성능
- **요구사항**: 특정 리소스의 과거 메트릭을 시간 범위를 지정하여 조회할 때, 3초 이내에 결과가 표시
- **테스트**: `TestMetricsQueryPerformance_SC007`

## 테스트 실행

### 전제 조건

1. **테스트 데이터베이스 설정**:
   ```bash
   export PERF_DB_HOST=localhost
   export PERF_DB_PORT=3306
   export PERF_DB_USER=root
   export PERF_DB_PASSWORD=your-password
   export PERF_DB_NAME=openstack_monitor_perf
   ```

2. **데이터베이스 생성**:
   ```sql
   CREATE DATABASE openstack_monitor_perf;
   ```

### 단위 성능 테스트 실행

```bash
cd backend
go test -tags=performance ./tests/performance/... -v
```

### 특정 테스트 실행

```bash
# SC-001 테스트
go test -tags=performance ./tests/performance -run TestTopologyQueryPerformance_SC001 -v

# SC-002 테스트
go test -tags=performance ./tests/performance -run TestTopologyQueryPerformance_SC002 -v

# SC-007 테스트
go test -tags=performance ./tests/performance -run TestMetricsQueryPerformance_SC007 -v

# 대규모 환경 테스트
go test -tags=performance ./tests/performance -run TestLargeScaleEnvironment_10000VMs -v
```

### 벤치마크 테스트 실행

```bash
# 토폴로지 조회 벤치마크
go test -tags=performance ./tests/performance -bench=BenchmarkTopologyQuery -benchmem

# 메트릭 조회 벤치마크
go test -tags=performance ./tests/performance -bench=BenchmarkMetricsQuery -benchmem

# 대규모 환경 벤치마크
go test -tags=performance ./tests/performance -bench=BenchmarkLargeScaleTopology -benchmem
```

### 짧은 모드 스킵

성능 테스트는 기본적으로 `-short` 플래그로 스킵됩니다. 전체 테스트를 실행하려면:

```bash
go test -tags=performance ./tests/performance/... -v
```

## 테스트 구조

### topology_test.go
- 토폴로지 조회 성능 테스트 (SC-001, SC-002)
- 경로 탐색 성능 테스트
- 벤치마크 테스트

### metrics_test.go
- 메트릭 조회 성능 테스트 (SC-007)
- 대용량 데이터셋 테스트
- 벤치마크 테스트

### load_test.go
- 대규모 환경 시뮬레이션 (10,000 VM)
- 동시 쿼리 테스트
- 데이터베이스 쿼리 성능 테스트

## 성능 목표

| 테스트 | 목표 | 측정 항목 |
|--------|------|-----------|
| SC-001 | 5초 이내 | VM 토폴로지 조회 시간 |
| SC-002 | 10초 이내 | 10,000 VM 환경에서 토폴로지 조회 시간 |
| SC-007 | 3초 이내 | 과거 메트릭 조회 시간 (7일 범위) |

## 주의사항

1. **데이터베이스 용량**: 대규모 테스트는 상당한 데이터베이스 용량을 사용합니다. 충분한 디스크 공간을 확보하세요.

2. **실행 시간**: 대규모 테스트는 실행에 시간이 걸릴 수 있습니다 (수 분 ~ 수십 분).

3. **독립 실행**: 각 테스트는 독립적으로 실행되며, 테스트 간 데이터를 공유하지 않습니다.

4. **환경 변수**: 테스트 실행 전에 필요한 환경 변수를 설정하세요.

## 트러블슈팅

### 데이터베이스 연결 실패

```bash
# 연결 확인
mysql -h $PERF_DB_HOST -P $PERF_DB_PORT -u $PERF_DB_USER -p$PERF_DB_PASSWORD -e "SELECT 1"
```

### 메모리 부족

대규모 테스트 실행 시 메모리 부족이 발생할 수 있습니다. 테스트 데이터베이스의 메모리 설정을 확인하세요.

### 타임아웃

테스트가 타임아웃되면 데이터베이스 인덱스가 올바르게 생성되었는지 확인하세요.

