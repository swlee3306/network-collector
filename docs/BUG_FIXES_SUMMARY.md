# 버그 수정 요약 문서

이 문서는 OpenStack Monitoring System 개발 중 발견되고 수정된 모든 버그들을 정리한 것입니다.

## 수정 일자: 2025-01-XX

---

## 1. 데이터베이스 연결 관리 관련 버그

### Bug 1-1: `database.Close()`의 잘못된 호출 시점 (API 서버)
**파일**: `backend/cmd/api/main.go`

**문제점**:
- `database.Close()`가 `defer`로 즉시 호출되어 서버 실행 중에 데이터베이스 연결이 닫힘
- 서버가 goroutine에서 계속 실행되는 동안 연결이 닫혀 후속 데이터베이스 작업 실패

**수정 내용**:
- `defer database.Close()` 제거
- graceful shutdown 핸들러에서 `server.Stop()` 후에 `database.Close()` 호출
- 서버 종료 후에만 데이터베이스 연결이 닫히도록 변경

**코드 위치**: `backend/cmd/api/main.go:77-80`

---

### Bug 1-2: `database.Close()`의 잘못된 호출 시점 (Collector 서버)
**파일**: `backend/cmd/collector/main.go`

**문제점**:
- `database.Close()`가 `defer`로 즉시 호출되어 cleanup service goroutine이 여전히 데이터베이스를 사용하는 동안 연결이 닫힘
- `sync.Once`로 인해 한 번만 실행되어 cleanup service가 실행 중일 때 연결이 닫힐 수 있음
- Race condition으로 graceful shutdown 중 데이터베이스 쿼리 실패 가능

**수정 내용**:
- `defer database.Close()` 제거
- graceful shutdown 핸들러에서 모든 goroutine이 완료된 후 `database.Close()` 호출
- cleanup service와 collection goroutine이 모두 종료된 후에만 데이터베이스 연결 종료

**코드 위치**: `backend/cmd/collector/main.go:137-140`

---

### Bug 1-3: 데이터베이스 연결의 동시성 문제
**파일**: `backend/internal/database/connection.go`

**문제점**:
- `database.Close()`가 전역 `DB` 변수를 닫을 때 여러 `Connect()` 호출이 있었는지 확인하지 않음
- 여러 서비스 인스턴스가 있으면 첫 번째 `Close()`가 연결 풀을 닫아 후속 쿼리 실패 가능

**수정 내용**:
- `sync.Once`를 사용해 `Close()`가 한 번만 실행되도록 보장
- `Connect()`에서 이미 연결되어 있으면 기존 연결을 반환하고, 연결이 끊어진 경우에만 재연결
- `sync.Mutex`로 `Connect()`와 `Close()` 동시 접근 보호
- 연결 상태 확인을 위해 `Ping()` 사용

**코드 위치**: `backend/internal/database/connection.go:29-86`

---

## 2. SSE (Server-Sent Events) 관련 버그

### Bug 2-1: StreamEvents의 동시 쓰기 문제
**파일**: `backend/internal/api/handlers/events.go`

**문제점**:
- Heartbeat goroutine과 event streaming goroutine이 동시에 `c.Writer`에 쓰기 작업 수행
- HTTP ResponseWriter는 thread-safe하지 않아 동시 쓰기가 SSE 스트림 손상 가능

**수정 내용**:
- `sync.Mutex`로 `c.Writer` 쓰기 동기화
- `writeSSE()` 헬퍼 함수로 모든 쓰기 작업을 mutex로 보호
- 두 goroutine이 동일한 mutex를 통해 순차적으로 쓰기 수행

**코드 위치**: `backend/internal/api/handlers/events.go:62-76`

---

### Bug 2-2: WaitGroup의 잘못된 카운트
**파일**: `backend/internal/api/handlers/events.go`

**문제점**:
- WaitGroup에 하나의 goroutine만 추가되었지만 두 개의 goroutine이 실행됨
- 이벤트 스트리밍 goroutine이 WaitGroup에 신호를 보내지 않아 `wg.Wait()`가 무한 대기

**수정 내용**:
- `wg.Add(2)`로 변경하여 두 goroutine 모두 WaitGroup에 추가
- 이벤트 스트리밍 goroutine에 `defer wg.Done()` 추가
- `done` 채널은 이벤트 루프 완료를 알리기 위해 유지
- 두 goroutine이 모두 완료될 때까지 대기하도록 수정

**코드 위치**: `backend/internal/api/handlers/events.go:78-160`

---

### Bug 2-3: Broadcast 메서드의 동시성 문제
**파일**: `backend/internal/services/events/broadcaster.go`

**문제점**:
- `Broadcast` 메서드에서 `RLock`을 사용하면서 이벤트를 직접 수정
- 여러 goroutine이 동시에 호출하면 같은 이벤트 구조체를 수정하여 race condition 발생
- 여러 클라이언트에 보낼 때 타임스탬프가 다를 수 있음

**수정 내용**:
- 클라이언트 목록을 복사한 후 락 해제 (락을 오래 유지하지 않음)
- 각 클라이언트에 보낼 때 새로운 이벤트 인스턴스 생성
- Data 맵도 복사하여 mutable state 공유 방지
- 타임스탬프를 한 번만 설정하고 모든 클라이언트에 동일한 값 사용

**코드 위치**: `backend/internal/services/events/broadcaster.go:65-100`

---

### Bug 2-4: globalBroadcaster의 동기화 문제
**파일**: `backend/internal/api/handlers/events.go`, `backend/internal/api/server.go`

**문제점**:
- 전역 변수 `globalBroadcaster`가 동기화 없이 접근됨
- 데이터 레이스 가능성

**수정 내용**:
- 전역 변수 제거
- `Server` 구조체에 `broadcaster` 필드 추가
- `StreamEvents` 함수가 broadcaster를 파라미터로 받도록 변경
- Dependency injection 패턴 적용

**코드 위치**: 
- `backend/internal/api/server.go:25, 30, 126`
- `backend/internal/api/handlers/events.go:16`

---

## 3. 데이터 수집 관련 버그

### Bug 3-1: Foreign Key 관계의 ID 변환 문제 (Port)
**파일**: `backend/internal/services/collector/network.go`

**문제점**:
- `savePort()`에서 OpenStack `port.DeviceID`를 직접 저장하지만, 모델은 내부 Instance ID를 기대
- Foreign key 관계가 깨져 GORM 조인 실패

**수정 내용**:
- DeviceOwner가 instance인 경우 OpenStack DeviceID를 내부 Instance ID로 변환
- Instance를 찾지 못한 경우 빈 문자열 저장 (foreign key 무결성 유지)
- Non-instance device(router 등)는 OpenStack ID 직접 저장

**코드 위치**: `backend/internal/services/collector/network.go:124-177`

---

### Bug 3-2: Foreign Key 관계의 ID 변환 문제 (Volume)
**파일**: `backend/internal/services/collector/volume.go`

**문제점**:
- `saveVolume()`에서 OpenStack ServerID를 저장하지만 내부 Instance ID를 저장해야 함
- Foreign key 관계가 깨져 GORM 조인 실패

**수정 내용**:
- OpenStack ServerID를 내부 Instance ID로 변환
- Instance를 찾지 못한 경우 빈 문자열 저장 (foreign key 무결성 유지)
- 주석 추가로 의도 명확화

**코드 위치**: `backend/internal/services/collector/volume.go:55-95`

---

### Bug 3-3: JSON marshaling 에러 무시
**파일**: `backend/internal/services/collector/network.go`

**문제점**:
- `network.go`의 141번과 188번 라인에서 JSON marshaling 에러를 무시
- marshaling 실패 시 빈 JSON 문자열이 저장되어 데이터 손상

**수정 내용**:
- `savePort` 함수에서 `fixedIPsJSON` marshaling 에러 처리 추가
- `saveRouter` 함수에서 `externalGatewayJSON` marshaling 에러 처리 추가
- 에러 발생 시 호출자에게 전파

**코드 위치**: 
- `backend/internal/services/collector/network.go:159-163`
- `backend/internal/services/collector/network.go:210-213`

---

## 4. 에러 처리 관련 버그

### Bug 4-1: containsAny 함수의 비효율적인 구현
**파일**: `backend/internal/services/collector/error_handler.go`

**문제점**:
- 수동 substring 검색으로 O(n*m) 복잡도
- Go 표준 라이브러리의 최적화된 함수를 사용하지 않음

**수정 내용**:
- `strings.Contains`를 사용하도록 변경
- `strings` 패키지 import 추가

**코드 위치**: `backend/internal/services/collector/error_handler.go:118-126`

---

### Bug 4-2: ClassifyOpenStackError 재시도 가능성 로직 문제
**파일**: `backend/internal/services/collector/error_handler.go`

**문제점**:
- retryable 패턴 체크 후 non-retryable 체크가 무조건 덮어씀
- 주석은 401, 403을 제외한다고 하지만 실제로는 모든 4xx에 대해 false 설정
- "timeout 401" 같은 경우에도 false로 설정됨

**수정 내용**:
- 로직을 단계적으로 처리하도록 변경:
  1. retryable 패턴이 있으면 `retryable = true` (우선)
  2. retryable 패턴이 없고 non-retryable 4xx (400, 404, 409)가 있으면 `retryable = false`
  3. 401/403은 retryable 패턴과 함께 있으면 `retryable = true`, 아니면 `retryable = false`
- retryable 패턴이 이미 true인 경우 non-retryable 체크가 덮어쓰지 않도록 보호

**코드 위치**: `backend/internal/services/collector/error_handler.go:78-116`

---

## 5. API 라우팅 관련 버그

### Bug 5-1: 라우트 등록 순서 문제
**파일**: `backend/internal/api/server.go`

**문제점**:
- `/projects/:id` 라우트가 `/projects/compare` 라우트보다 먼저 등록됨
- Gin은 등록 순서대로 라우트를 매칭하므로, `/projects/compare` 요청이 `:id` 패턴에 "compare"로 매칭됨
- 결과적으로 비교 엔드포인트에 접근할 수 없음

**수정 내용**:
- `/projects/compare` 라우트를 `/projects/:id` 라우트보다 먼저 등록하도록 순서 변경
- `/projects/:id/summary`도 `/projects/:id`보다 먼저 등록하여 구체적인 경로가 우선 매칭되도록 함
- 주석 추가로 라우트 등록 순서의 중요성 명시

**수정 후 라우트 등록 순서**:
1. `/projects/compare` (구체적인 경로)
2. `/projects/:id/summary` (구체적인 경로)
3. `/projects/:id` (파라미터 경로)

**코드 위치**: `backend/internal/api/server.go:102-105`

---

## 6. 데이터베이스 스키마 관련 버그

### Bug 6-1: 인덱스 이름 중복 문제
**파일**: `backend/internal/database/migrations/migrate.go`

**문제점**:
- `createIndexes` 함수가 세 개의 다른 메트릭 테이블에 동일한 이름 `idx_timestamp`로 인덱스를 생성하려고 시도
- MySQL에서 인덱스 이름은 데이터베이스 전체에서 고유해야 함
- `IF NOT EXISTS` 절로 인해 첫 번째 인덱스만 생성되고 나머지는 조용히 실패
- `network_metrics`와 `hypervisor_metrics` 테이블에 타임스탬프 인덱스가 없어 쿼리 성능 저하

**수정 내용**:
- 각 인덱스에 고유한 이름 부여:
  - `idx_instance_metrics_timestamp`
  - `idx_network_metrics_timestamp`
  - `idx_hypervisor_metrics_timestamp`

**코드 위치**: `backend/internal/database/migrations/migrate.go:39-60`

---

### Bug 6-2: GetRoutersByNetwork 필터링 문제
**파일**: `backend/internal/services/storage/repository.go`

**문제점**:
- `GetRoutersByNetwork` 함수가 `networkOpenStackID` 파라미터를 완전히 무시하고 모든 라우터를 반환
- WHERE 절 없이 `r.db.Find(&routers)`를 반환하여 잘못된 토폴로지 데이터 사용

**수정 내용**:
- 네트워크로 라우터를 필터링하도록 수정
- 네트워크를 `networkOpenStackID`로 조회
- 해당 네트워크와 연결된 포트를 찾고, 라우터가 소유한 포트를 필터링
- 최종적으로 해당 포트와 연결된 고유한 라우터들을 반환

**코드 위치**: `backend/internal/services/storage/repository.go` (GetRoutersByNetwork 메서드)

---

## 7. 입력 파싱 관련 버그

### Bug 7-1: splitCommaSeparated 공백 처리 문제
**파일**: `backend/internal/api/handlers/projects_comparison.go`

**문제점**:
- `splitCommaSeparated()` 함수가 공백을 제거하지 않음
- "proj-1, proj-2" 입력 시 " proj-2"처럼 앞에 공백이 포함된 ID 생성
- 공백이 포함된 ID로 인해 데이터베이스 조회 실패
- 공백만 있는 문자열("  ")이 트림 후 빈 문자열("")이 되어도 결과에 추가됨

**수정 내용**:
- `strings.TrimSpace()`를 사용하여 각 토큰의 앞뒤 공백 제거
- 트림 후 빈 문자열인지 확인하고, 비어있지 않을 때만 결과에 추가
- 공백만 있는 토큰은 자동으로 필터링됨

**코드 위치**: `backend/internal/api/handlers/projects_comparison.go:149-171`

---

## 수정 사항 요약

### 수정된 파일 목록
1. `backend/cmd/api/main.go` - 데이터베이스 연결 종료 시점 수정
2. `backend/cmd/collector/main.go` - 데이터베이스 연결 종료 시점 수정
3. `backend/internal/database/connection.go` - 연결 관리 동시성 개선
4. `backend/internal/api/handlers/events.go` - SSE 동시성 및 WaitGroup 수정
5. `backend/internal/services/events/broadcaster.go` - 이벤트 브로드캐스트 동시성 수정
6. `backend/internal/api/server.go` - 라우트 등록 순서 수정, Dependency injection 적용
7. `backend/internal/services/collector/network.go` - Foreign key ID 변환, JSON marshaling 에러 처리
8. `backend/internal/services/collector/volume.go` - Foreign key ID 변환
9. `backend/internal/services/collector/error_handler.go` - 에러 분류 로직 개선
10. `backend/internal/database/migrations/migrate.go` - 인덱스 이름 고유성 보장
11. `backend/internal/services/storage/repository.go` - GetRoutersByNetwork 필터링 수정
12. `backend/internal/api/handlers/projects_comparison.go` - 입력 파싱 공백 처리 개선

### 주요 개선 사항
- **Graceful Shutdown**: 모든 서비스에서 데이터베이스 연결이 goroutine 완료 후에만 닫히도록 수정
- **동시성 안전성**: SSE 핸들러와 이벤트 브로드캐스터의 race condition 해결
- **데이터 무결성**: Foreign key 관계를 올바르게 유지하도록 ID 변환 로직 개선
- **에러 처리**: JSON marshaling 에러와 재시도 로직 개선
- **라우팅**: Gin 라우트 등록 순서 문제 해결
- **입력 검증**: 공백 처리 및 빈 문자열 필터링 개선

### 테스트 상태
- 모든 수정 사항은 린터 검사를 통과했습니다.
- 빌드 오류 없이 컴파일됩니다.

---

## 향후 개선 사항

1. **통합 테스트**: 수정된 버그들에 대한 통합 테스트 추가
2. **성능 테스트**: 동시성 수정 사항의 성능 영향 평가
3. **모니터링**: 데이터베이스 연결 상태 및 SSE 연결 수 모니터링 추가
4. **문서화**: API 엔드포인트 및 에러 처리 가이드 문서화

