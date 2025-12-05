# Integration Tests

이 디렉토리는 통합 테스트를 포함합니다.

## 실행 방법

통합 테스트를 실행하려면 `integration` 빌드 태그를 사용해야 합니다:

```bash
go test -tags=integration ./tests/integration/...
```

## 전제 조건

통합 테스트를 실행하기 전에 다음이 필요합니다:

1. **테스트 데이터베이스**: MariaDB 인스턴스가 실행 중이어야 합니다.
2. **환경 변수 설정**:
   - `TEST_DB_HOST`: 데이터베이스 호스트 (기본값: localhost)
   - `TEST_DB_PORT`: 데이터베이스 포트 (기본값: 3306)
   - `TEST_DB_USER`: 데이터베이스 사용자 (기본값: root)
   - `TEST_DB_PASSWORD`: 데이터베이스 비밀번호
   - `TEST_DB_NAME`: 테스트 데이터베이스 이름 (기본값: openstack_monitor_test)

## 테스트 구성

### 데이터베이스 통합 테스트 (`database_test.go`)

- 데이터베이스 연결 테스트
- Repository 작업 테스트 (Instance, Network, Topology)

### API 통합 테스트 (`api_test.go`)

- Health Check 엔드포인트 테스트
- Login 엔드포인트 테스트
- 인증이 필요한 엔드포인트 테스트
- 인증 실패 테스트

## 주의사항

- 통합 테스트는 실제 데이터베이스에 연결하므로, 테스트 데이터베이스를 사용하는 것이 좋습니다.
- 테스트는 자동으로 데이터를 정리하지만, 테스트 실패 시 수동 정리가 필요할 수 있습니다.
- `-short` 플래그를 사용하면 통합 테스트가 스킵됩니다: `go test -short ./...`

