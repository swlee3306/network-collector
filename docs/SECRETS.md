# Secret 값 생성 가이드

이 문서는 OpenStack 모니터링 시스템 배포 시 필요한 Secret 값들을 생성하는 방법을 설명합니다.

## JWT_SECRET

JWT_SECRET은 JWT 토큰을 서명하는 데 사용되는 비밀 키입니다. 충분히 길고 예측 불가능한 랜덤 문자열을 사용해야 합니다.

### 생성 방법

#### 방법 1: OpenSSL 사용 (권장)

```bash
openssl rand -base64 32
```

출력 예시:
```
acpJvW2XQF/OaOm3iEI3xzGLpT3o1yizTiv3JEQOc+8=
```

#### 방법 2: OpenSSL (16진수)

```bash
openssl rand -hex 32
```

출력 예시:
```
a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
```

#### 방법 3: Python 사용

```bash
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

#### 방법 4: Node.js 사용

```bash
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"
```

### 요구사항

- **최소 길이**: 32바이트 (256비트) 이상 권장
- **형식**: Base64 인코딩된 문자열 또는 16진수 문자열
- **보안**: 예측 불가능한 암호학적으로 안전한 난수 생성기 사용
- **프로덕션**: 각 환경마다 고유한 값 사용

### 사용 예시

```bash
# Secret 생성 시
kubectl create secret generic network-collector-secrets \
  --from-literal=JWT_SECRET='acpJvW2XQF/OaOm3iEI3xzGLpT3o1yizTiv3JEQOc+8=' \
  ...

# Helm 배포 시
helm install network-collector backend/deployments/helm \
  --set secrets.jwtSecret='acpJvW2XQF/OaOm3iEI3xzGLpT3o1yizTiv3JEQOc+8=' \
  ...
```

## AUTH_TOKEN

AUTH_TOKEN은 API 인증에 사용되는 토큰입니다. 현재 구현에서는 이 토큰을 직접 사용합니다.

### 생성 방법

JWT_SECRET과 동일한 방법으로 생성할 수 있습니다:

```bash
openssl rand -base64 32
```

### 사용 예시

```bash
# 로그인 후 받은 토큰을 Authorization 헤더에 사용
curl -H "Authorization: Bearer your-auth-token" \
  http://localhost:8080/api/v1/instances
```

## DB_PASSWORD

데이터베이스 비밀번호입니다.

### 생성 방법

```bash
openssl rand -base64 24
```

또는 MariaDB가 요구하는 복잡도 요구사항을 만족하는 비밀번호:

```bash
# 영문 대소문자, 숫자, 특수문자 포함
openssl rand -base64 24 | tr -d "=+/" | cut -c1-24
```

### 요구사항

- **최소 길이**: 12자 이상 권장
- **복잡도**: 영문 대소문자, 숫자, 특수문자 포함 권장

## OPENSTACK_PASSWORD

OpenStack 사용자 비밀번호입니다. OpenStack 환경에서 설정한 실제 비밀번호를 사용합니다.

## 모든 Secret 값 한 번에 생성

다음 스크립트를 사용하여 모든 Secret 값을 한 번에 생성할 수 있습니다:

```bash
#!/bin/bash

echo "=== Secret 값 생성 ==="
echo ""
echo "JWT_SECRET:"
openssl rand -base64 32
echo ""
echo "AUTH_TOKEN:"
openssl rand -base64 32
echo ""
echo "DB_PASSWORD:"
openssl rand -base64 24 | tr -d "=+/" | cut -c1-24
echo ""
echo "=== 생성 완료 ==="
```

## 보안 권장사항

1. **프로덕션 환경**
   - 각 환경(개발, 스테이징, 프로덕션)마다 고유한 Secret 값 사용
   - Secret 값을 버전 관리 시스템에 커밋하지 않음
   - Secret 관리 도구 사용 권장 (예: Sealed Secrets, External Secrets Operator)

2. **Secret 저장**
   - Kubernetes Secret으로 저장
   - 환경 변수로 직접 노출하지 않음
   - 로그에 출력하지 않음

3. **정기적 교체**
   - 정기적으로 Secret 값 교체 (예: 분기별)
   - 교체 시 서비스 재시작 필요

4. **접근 제어**
   - Secret에 접근할 수 있는 사용자 최소화
   - RBAC를 통한 접근 제어 설정

## Secret 값 확인

### Kubernetes Secret 확인

```bash
# Secret 목록 확인
kubectl get secrets

# Secret 값 확인 (Base64 디코딩 필요)
kubectl get secret network-collector-secrets -o jsonpath='{.data.JWT_SECRET}' | base64 -d
kubectl get secret network-collector-secrets -o jsonpath='{.data.AUTH_TOKEN}' | base64 -d
```

### Helm Secret 확인

```bash
# Helm values 확인
helm get values network-collector
```

## 트러블슈팅

### Secret 값이 적용되지 않음

1. Secret이 올바르게 생성되었는지 확인:
   ```bash
   kubectl get secret network-collector-secrets -o yaml
   ```

2. Pod가 Secret을 참조하는지 확인:
   ```bash
   kubectl describe pod <pod-name>
   ```

3. Pod 재시작:
   ```bash
   kubectl rollout restart deployment/network-collector-api
   ```

### 인증 실패

1. AUTH_TOKEN이 올바른지 확인:
   ```bash
   kubectl get secret network-collector-secrets -o jsonpath='{.data.AUTH_TOKEN}' | base64 -d
   ```

2. API 로그 확인:
   ```bash
   kubectl logs deployment/network-collector-api | grep -i auth
   ```

