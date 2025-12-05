# 배포 가이드

이 문서는 OpenStack 모니터링 시스템을 Kubernetes에 배포하는 방법을 설명합니다.

## 목차

1. [전제 조건](#전제-조건)
2. [이미지 빌드](#이미지-빌드)
3. [Kubernetes 배포](#kubernetes-배포)
4. [Helm을 사용한 배포](#helm을-사용한-배포)
5. [환경 변수 설정](#환경-변수-설정)
6. [검증](#검증)
7. [트러블슈팅](#트러블슈팅)

## 전제 조건

### 필수 요구사항

- Kubernetes 클러스터 (v1.19+)
- kubectl 설치 및 클러스터 접근 권한
- Docker 또는 컨테이너 레지스트리 접근 권한
- Helm 3.0+ (Helm 배포 시)

### OpenStack 접근 권한

다음 OpenStack 서비스에 대한 접근 권한이 필요합니다:

- Keystone (인증)
- Nova (컴퓨트)
- Neutron (네트워킹)
- Cinder (볼륨)
- Glance (이미지, 선택적)

### 데이터베이스

- MariaDB 10.6+ 또는 호환 가능한 MySQL
- 데이터베이스 생성 권한

## 이미지 빌드

### Docker 이미지 빌드

#### Collector 이미지

```bash
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
```

#### API 이미지

```bash
cd backend
docker build -t network-collector-api:latest -f Dockerfile.api .
```

#### Frontend 이미지

```bash
cd frontend
docker build -t network-collector-frontend:latest -f Dockerfile .
```

### 이미지 레지스트리에 푸시

```bash
# 예: Docker Hub
docker tag network-collector:latest your-registry/network-collector:latest
docker push your-registry/network-collector:latest

docker tag network-collector-api:latest your-registry/network-collector-api:latest
docker push your-registry/network-collector-api:latest

docker tag network-collector-frontend:latest your-registry/network-collector-frontend:latest
docker push your-registry/network-collector-frontend:latest
```

## Kubernetes 배포

### 1. Secret 생성

```bash
kubectl create secret generic network-collector-secrets \
  --from-literal=DB_PASSWORD='your-db-password' \
  --from-literal=OPENSTACK_AUTH_URL='http://keystone:5000/v3' \
  --from-literal=OPENSTACK_USERNAME='admin' \
  --from-literal=OPENSTACK_PASSWORD='your-openstack-password' \
  --from-literal=OPENSTACK_PROJECT_ID='your-project-id' \
  --from-literal=JWT_SECRET='your-jwt-secret' \
  --from-literal=AUTH_TOKEN='your-auth-token'
```

### 2. ConfigMap 생성

```bash
kubectl apply -f backend/deployments/k8s/configmap.yaml
```

### 3. MariaDB 배포

```bash
kubectl apply -f backend/deployments/k8s/mariadb/statefulset.yaml
```

MariaDB가 준비될 때까지 대기:

```bash
kubectl wait --for=condition=ready pod -l app=mariadb --timeout=300s
```

### 4. Collector 배포

```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
```

### 5. API Service 배포

```bash
kubectl apply -f backend/deployments/k8s/api/deployment.yaml
```

### 6. Frontend 배포

```bash
kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml
```

### 7. Ingress 배포 (선택적)

```bash
kubectl apply -f backend/deployments/k8s/ingress.yaml
```

## Helm을 사용한 배포

### 1. Helm Chart 설치

```bash
cd backend/deployments/helm

helm install network-collector . \
  --set secrets.dbPassword='your-db-password' \
  --set secrets.openstackAuthURL='http://keystone:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='your-openstack-password' \
  --set secrets.openstackProjectID='your-project-id' \
  --set secrets.jwtSecret='your-jwt-secret' \
  --set secrets.authToken='your-auth-token'
```

### 2. values.yaml 파일 사용

```bash
# values.yaml 파일 편집
vim values.yaml

# 설치
helm install network-collector . -f values.yaml
```

### 3. 업그레이드

```bash
helm upgrade network-collector . \
  --set secrets.dbPassword='your-db-password' \
  --set secrets.openstackAuthURL='http://keystone:5000/v3' \
  --set secrets.openstackUsername='admin' \
  --set secrets.openstackPassword='your-openstack-password' \
  --set secrets.openstackProjectID='your-project-id' \
  --set secrets.jwtSecret='your-jwt-secret' \
  --set secrets.authToken='your-auth-token'
```

### 4. 제거

```bash
helm uninstall network-collector
```

## 환경 변수 설정

### Collector 환경 변수

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
| `OPENSTACK_PROJECT_ID` | OpenStack 프로젝트 ID | 예 | - |
| `OPENSTACK_DOMAIN_NAME` | OpenStack 도메인 이름 | 아니오 | default |
| `LOG_LEVEL` | 로그 레벨 | 아니오 | info |
| `ENVIRONMENT` | 환경 (development/production) | 아니오 | development |

### API Service 환경 변수

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

## 검증

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

### 헬스 체크

```bash
# API 헬스 체크
kubectl port-forward svc/network-collector-api 8080:8080
curl http://localhost:8080/health
```

### 데이터 수집 확인

```bash
# Collector가 데이터를 수집하는지 확인
kubectl logs deployment/network-collector | grep "collection"
```

## 트러블슈팅

### Pod가 시작되지 않음

1. **이미지 Pull 실패**
   ```bash
   kubectl describe pod <pod-name>
   # imagePullSecrets 설정 확인
   ```

2. **Secret/ConfigMap 누락**
   ```bash
   kubectl get secret network-collector-secrets
   kubectl get configmap network-collector-config
   ```

3. **리소스 부족**
   ```bash
   kubectl describe node
   # 리소스 제한 확인
   ```

### 데이터베이스 연결 실패

1. **MariaDB 상태 확인**
   ```bash
   kubectl get pods -l app=mariadb
   kubectl logs -l app=mariadb
   ```

2. **네트워크 연결 확인**
   ```bash
   kubectl exec -it <collector-pod> -- ping mariadb
   ```

3. **비밀번호 확인**
   ```bash
   kubectl get secret network-collector-secrets -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
   ```

### OpenStack API 연결 실패

1. **인증 정보 확인**
   ```bash
   kubectl get secret network-collector-secrets -o yaml
   ```

2. **네트워크 연결 확인**
   ```bash
   kubectl exec -it <collector-pod> -- curl <openstack-auth-url>
   ```

3. **Collector 로그 확인**
   ```bash
   kubectl logs -f deployment/network-collector | grep -i error
   ```

### API가 응답하지 않음

1. **Pod 상태 확인**
   ```bash
   kubectl get pods -l app=network-collector-api
   kubectl describe pod <api-pod>
   ```

2. **헬스 체크**
   ```bash
   kubectl exec -it <api-pod> -- curl http://localhost:8080/health
   ```

3. **서비스 엔드포인트 확인**
   ```bash
   kubectl get endpoints network-collector-api
   ```

### Frontend가 로드되지 않음

1. **API URL 확인**
   ```bash
   kubectl get configmap network-collector-frontend-config -o yaml
   ```

2. **브라우저 콘솔 확인**
   - 브라우저 개발자 도구에서 네트워크 오류 확인

3. **CORS 설정 확인**
   - API 서비스의 CORS 설정 확인

## 추가 리소스

- [Kubernetes 매니페스트 가이드](backend/deployments/k8s/README.md)
- [Helm Chart 가이드](backend/deployments/helm/README.md)
- [API 문서](specs/001-openstack-monitoring/contracts/api.yaml)
- [빠른 시작 가이드](specs/001-openstack-monitoring/quickstart.md)

