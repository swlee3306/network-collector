# 메트릭 수집 기능 확인 가이드

이 문서는 인스턴스, 네트워크, 하이퍼바이저 메트릭 수집 기능이 정상적으로 동작하는지 확인하는 방법을 설명합니다.

## 메트릭 수집 개요

시스템은 다음 메트릭을 수집합니다:

1. **인스턴스 메트릭** (`instance_metrics`)
   - CPU 사용률
   - 메모리 사용량/총량
   - 디스크 읽기/쓰기 바이트
   - 네트워크 송수신 바이트

2. **네트워크 메트릭** (`network_metrics`)
   - 총 바이트
   - 드롭된 패킷 수
   - 연결된 VM 수

3. **하이퍼바이저 메트릭** (`hypervisor_metrics`)
   - 사용된/총 VCPU 수
   - 사용된/총 메모리
   - 실행 중인 VM 수

## 메트릭 수집 주기

- **리소스 수집**: 1분마다 (`main.go`의 ticker)
- **메트릭 수집**: 리소스 수집 후 자동 실행 (`openstack.go`의 `CollectAll()`)

## 확인 방법

### 1. 자동 확인 스크립트 사용

```bash
# 메트릭 수집 상태 확인
./scripts/check-metrics.sh
```

이 스크립트는 다음을 확인합니다:
- 메트릭 테이블 존재 여부
- 메트릭 데이터 존재 여부
- 최근 메트릭 상세 정보
- Collector Pod 로그
- API 엔드포인트 동작

### 2. 수동 확인

#### 2.1 데이터베이스에서 직접 확인

```bash
# Kubernetes Pod를 통해 데이터베이스 접속
kubectl run mysql-client --rm -i --restart=Never --image=mysql:8.0 -- \
  mysql -h mariadb -u openstack_monitor -p<password> openstack_monitor

# 인스턴스 메트릭 확인
SELECT COUNT(*) FROM instance_metrics;
SELECT * FROM instance_metrics ORDER BY timestamp DESC LIMIT 5;

# 네트워크 메트릭 확인
SELECT COUNT(*) FROM network_metrics;
SELECT * FROM network_metrics ORDER BY timestamp DESC LIMIT 5;

# 하이퍼바이저 메트릭 확인
SELECT COUNT(*) FROM hypervisor_metrics;
SELECT * FROM hypervisor_metrics ORDER BY timestamp DESC LIMIT 5;
```

#### 2.2 Collector Pod 로그 확인

```bash
# Collector Pod 로그 확인
kubectl logs -l app=network-collector --tail=100 | grep -i metric

# 메트릭 수집 관련 로그 확인
kubectl logs -l app=network-collector | grep "Collecting metrics"
```

예상되는 로그:
```
Collecting metrics...
Collecting instance metrics...
Collecting network metrics...
Collecting hypervisor metrics...
Metrics collection completed successfully
```

#### 2.3 API 엔드포인트 확인

```bash
# API Service 확인
kubectl get svc network-collector-api

# 인스턴스 목록 가져오기
kubectl port-forward svc/network-collector-api 8080:8080 &
INSTANCE_ID=$(curl -s http://localhost:8080/api/v1/instances | jq -r '.data[0].id')

# 인스턴스 메트릭 조회
curl "http://localhost:8080/api/v1/instances/$INSTANCE_ID/metrics"

# 네트워크 목록 가져오기
NETWORK_ID=$(curl -s http://localhost:8080/api/v1/networks | jq -r '.data[0].id')

# 네트워크 메트릭 조회
curl "http://localhost:8080/api/v1/networks/$NETWORK_ID/metrics"

# 하이퍼바이저 목록 가져오기
HYPERVISOR_ID=$(curl -s http://localhost:8080/api/v1/hypervisors | jq -r '.data[0].id')

# 하이퍼바이저 메트릭 조회
curl "http://localhost:8080/api/v1/hypervisors/$HYPERVISOR_ID/metrics"
```

## 메트릭 수집 로직

### 인스턴스 메트릭

```go
// backend/internal/services/collector/metrics.go
func (c *MetricsCollector) collectInstanceMetric(instance models.Instance) error {
    // Flavor에서 메모리 총량 가져오기
    // 실제 메트릭은 Ceilometer/Gnocchi에서 수집해야 함
    // 현재는 기본값(0)으로 저장
}
```

**참고**: 현재 구현은 기본 메트릭만 수집합니다. 실제 CPU, 메모리 사용률 등은 OpenStack Ceilometer 또는 Gnocchi를 통해 수집해야 합니다.

### 네트워크 메트릭

```go
// backend/internal/services/collector/metrics.go
func (c *MetricsCollector) collectNetworkMetric(network models.Network) error {
    // 네트워크에 연결된 포트를 확인하여 VM 수 계산
    // 실제 트래픽 메트릭은 모니터링 도구에서 수집해야 함
}
```

**참고**: 현재 구현은 연결된 VM 수만 계산합니다. 실제 트래픽 메트릭은 Neutron agents나 모니터링 도구를 통해 수집해야 합니다.

### 하이퍼바이저 메트릭

```go
// backend/internal/services/collector/metrics.go
func (c *MetricsCollector) collectHypervisorMetric(hypervisor models.Hypervisor) error {
    // 하이퍼바이저 리소스에서 직접 메트릭 가져오기
    // VCPUs, Memory, RunningVMs 등
}
```

**참고**: 하이퍼바이저 메트릭은 Nova API에서 직접 수집됩니다.

## 문제 해결

### 메트릭이 수집되지 않는 경우

1. **Collector Pod 상태 확인**
   ```bash
   kubectl get pods -l app=network-collector
   kubectl describe pod <pod-name>
   ```

2. **Collector 로그 확인**
   ```bash
   kubectl logs -l app=network-collector --tail=200
   ```

3. **데이터베이스 연결 확인**
   ```bash
   kubectl logs -l app=network-collector | grep -i "database\|connection\|error"
   ```

4. **메트릭 수집 에러 확인**
   ```bash
   kubectl logs -l app=network-collector | grep -i "metric.*error\|failed.*metric"
   ```

### 메트릭이 0으로만 저장되는 경우

현재 구현은 기본 메트릭만 수집합니다:
- **인스턴스**: Flavor의 메모리 총량만 저장 (CPU, 메모리 사용률은 0)
- **네트워크**: 연결된 VM 수만 계산 (트래픽은 0)
- **하이퍼바이저**: Nova API에서 수집한 실제 값 저장

실제 메트릭을 수집하려면:
1. OpenStack Ceilometer 또는 Gnocchi 설정
2. 메트릭 수집 로직 확장 (`collectInstanceMetric`, `collectNetworkMetric`)

### API 엔드포인트가 404를 반환하는 경우

1. **라우트 등록 확인**
   ```bash
   # API 서버 로그 확인
   kubectl logs -l app=network-collector-api | grep -i "route\|endpoint"
   ```

2. **서버 설정 확인**
   ```bash
   kubectl get configmap network-collector-config -o yaml
   ```

## 메트릭 데이터 보존

메트릭 데이터는 자동으로 정리됩니다:
- **보존 기간**: 30일 (기본값)
- **정리 주기**: 24시간마다
- **설정**: `backend/cmd/collector/main.go`의 `retention.NewCleanupService(repository, 30)`

보존 기간을 변경하려면:
```go
// backend/cmd/collector/main.go
cleanupService := retention.NewCleanupService(repository, 60) // 60일로 변경
```

## 참고

- 메트릭 수집은 리소스 수집 후 자동으로 실행됩니다.
- 메트릭 수집 실패는 전체 수집을 실패시키지 않습니다 (partial failure 허용).
- 메트릭 API는 시간 범위를 지정할 수 있습니다 (기본: 최근 24시간).

