# 트러블슈팅 가이드

이 문서는 OpenStack 모니터링 시스템 배포 및 운영 중 발생할 수 있는 문제와 해결 방법을 설명합니다.

## MariaDB Pod가 Pending 상태

### 증상

```bash
kubectl get pods
NAME        READY   STATUS    RESTARTS   AGE
mariadb-0   0/1     Pending  0          2m
```

```bash
kubectl describe pod mariadb-0
...
Events:
  Warning  FailedScheduling  default-scheduler  0/2 nodes are available: pod has unbound immediate PersistentVolumeClaims.
```

### 원인

PersistentVolumeClaim이 바인딩되지 않아 Pod가 스케줄링되지 않습니다. 일반적으로 StorageClass가 클러스터에 존재하지 않거나 잘못 설정된 경우 발생합니다.

### 해결 방법

#### 1. StorageClass 확인

```bash
kubectl get storageclass
```

출력 예시:
```
NAME                 PROVISIONER             RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
local-path           rancher.io/local-path   Delete          WaitForFirstConsumer   false                  18m
```

#### 2. StorageClass에 맞게 수정

**방법 A: local-path-storage 사용 (권장)**

`backend/deployments/k8s/mariadb/statefulset.yaml` 파일 수정:

```yaml
volumeClaimTemplates:
- metadata:
    name: mariadb-data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    storageClassName: local-path  # local-path-storage 사용
    resources:
      requests:
        storage: 20Gi
```

**방법 B: 기본 StorageClass 사용**

```yaml
volumeClaimTemplates:
- metadata:
    name: mariadb-data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    storageClassName: ""  # 기본 StorageClass 사용
    resources:
      requests:
        storage: 20Gi
```

#### 3. 기존 StatefulSet 업데이트

```bash
# StatefulSet 삭제 (PVC는 유지됨)
kubectl delete statefulset mariadb

# 수정된 매니페스트 적용
kubectl apply -f backend/deployments/k8s/mariadb/statefulset.yaml

# 또는 PVC를 삭제하고 다시 생성
kubectl delete pvc mariadb-data-mariadb-0
kubectl apply -f backend/deployments/k8s/mariadb/statefulset.yaml
```

#### 4. Helm Chart 사용 시

`backend/deployments/helm/values.yaml` 파일 수정:

```yaml
mariadb:
  storage:
    size: 20Gi
    storageClass: local-path  # 또는 ""
```

그리고 업그레이드:

```bash
helm upgrade network-collector backend/deployments/helm \
  --set mariadb.storage.storageClass=local-path \
  ...
```

### 확인

```bash
# PVC 상태 확인
kubectl get pvc

# Pod 상태 확인
kubectl get pods -l app=mariadb

# Pod 이벤트 확인
kubectl describe pod mariadb-0
```

## Collector가 데이터를 수집하지 않음

### 증상

- Collector Pod는 실행 중이지만 데이터가 수집되지 않음
- 로그에 OpenStack API 연결 오류

### 해결 방법

#### 1. OpenStack 인증 정보 확인

```bash
# Secret 확인
kubectl get secret network-collector-secrets -o yaml

# 개별 값 확인
kubectl get secret network-collector-secrets -o jsonpath='{.data.OPENSTACK_AUTH_URL}' | base64 -d
kubectl get secret network-collector-secrets -o jsonpath='{.data.OPENSTACK_PROJECT_ID}' | base64 -d
```

#### 2. 프로젝트 ID 형식 확인

**중요**: `OPENSTACK_PROJECT_ID`는 프로젝트 이름이 아닌 **UUID 형식**이어야 합니다.

```bash
# OpenStack CLI로 확인
openstack project list
```

출력에서 ID 열의 UUID 값을 사용해야 합니다.

#### 3. 네트워크 연결 확인

```bash
# Collector Pod에서 OpenStack API 접근 확인
kubectl exec -it deployment/network-collector -- curl -v http://192.168.219.121:5000/v3
```

#### 4. Collector 로그 확인

```bash
kubectl logs -f deployment/network-collector
```

일반적인 오류:
- `401 Unauthorized`: 인증 정보 오류
- `Connection refused`: 네트워크 연결 불가
- `Project not found`: 프로젝트 ID 오류

## API가 응답하지 않음

### 증상

- API Pod는 실행 중이지만 요청이 실패
- 500 Internal Server Error

### 해결 방법

#### 1. 데이터베이스 연결 확인

```bash
# MariaDB Pod 상태 확인
kubectl get pods -l app=mariadb

# MariaDB 로그 확인
kubectl logs -l app=mariadb

# API Pod에서 데이터베이스 연결 테스트
kubectl exec -it deployment/network-collector-api -- ping mariadb
```

#### 2. 데이터베이스 비밀번호 확인

```bash
kubectl get secret network-collector-secrets -o jsonpath='{.data.DB_PASSWORD}' | base64 -d
```

#### 3. API 로그 확인

```bash
kubectl logs -f deployment/network-collector-api
```

#### 4. 헬스 체크

```bash
kubectl port-forward svc/network-collector-api 8080:8080
curl http://localhost:8080/health
```

## 토폴로지가 표시되지 않음

### 증상

- Frontend에서 토폴로지 페이지가 비어있음
- 네트워크 경로가 표시되지 않음

### 해결 방법

#### 1. 네트워크 리소스 수집 확인

```bash
# Collector 로그에서 네트워크 수집 확인
kubectl logs deployment/network-collector | grep -i network
```

#### 2. 데이터베이스에 토폴로지 데이터 확인

```bash
# MariaDB에 접속
kubectl exec -it mariadb-0 -- mysql -u root -p

# 데이터베이스 선택
USE openstack_monitor;

# 토폴로지 노드 확인
SELECT COUNT(*) FROM topology_nodes;

# 토폴로지 엣지 확인
SELECT COUNT(*) FROM topology_edges;
```

#### 3. Frontend 콘솔 오류 확인

- 브라우저 개발자 도구(F12) 열기
- Console 탭에서 JavaScript 오류 확인
- Network 탭에서 API 요청 실패 확인

## Pod가 시작되지 않음

### 증상

- Pod가 계속 `ContainerCreating` 상태
- Pod가 `CrashLoopBackOff` 상태

### 해결 방법

#### 1. Pod 이벤트 확인

```bash
kubectl describe pod <pod-name>
```

#### 2. 이미지 Pull 실패

```bash
# 이미지가 존재하는지 확인
docker images | grep network-collector

# 이미지 레지스트리 접근 확인
kubectl describe pod <pod-name> | grep -i image
```

#### 3. Secret/ConfigMap 누락

```bash
# Secret 확인
kubectl get secret network-collector-secrets

# ConfigMap 확인
kubectl get configmap network-collector-config
```

#### 4. 리소스 부족

```bash
# 노드 리소스 확인
kubectl describe node

# Pod 리소스 요청 확인
kubectl describe pod <pod-name> | grep -A 5 "Limits\|Requests"
```

## 인증 실패

### 증상

- API 요청 시 401 Unauthorized
- 로그인 실패

### 해결 방법

#### 1. AUTH_TOKEN 확인

```bash
kubectl get secret network-collector-secrets -o jsonpath='{.data.AUTH_TOKEN}' | base64 -d
```

#### 2. 로그인 테스트

```bash
# 로그인 시도
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# 받은 토큰으로 API 호출
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/instances
```

#### 3. Secret 재생성

```bash
# Secret 삭제
kubectl delete secret network-collector-secrets

# Secret 재생성
kubectl create secret generic network-collector-secrets \
  --from-literal=AUTH_TOKEN='new-token' \
  ...

# Pod 재시작
kubectl rollout restart deployment/network-collector-api
```

## 데이터베이스 마이그레이션 실패

### 증상

- API가 시작되지 않음
- 데이터베이스 스키마 오류

### 해결 방법

#### 1. 마이그레이션 수동 실행

```bash
# 마이그레이션 Pod 실행
kubectl run migrate --image=network-collector:latest \
  --restart=Never \
  --env="DB_HOST=mariadb" \
  --env="DB_USER=root" \
  --env="DB_PASSWORD=$(kubectl get secret network-collector-secrets -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)" \
  --env="DB_NAME=openstack_monitor" \
  --command -- /app/migrate

# 로그 확인
kubectl logs migrate
```

#### 2. 데이터베이스 직접 확인

```bash
# MariaDB 접속
kubectl exec -it mariadb-0 -- mysql -u root -p

# 테이블 목록 확인
SHOW TABLES;
```

## 성능 문제

### 증상

- API 응답이 느림
- 토폴로지 조회가 10초 이상 소요

### 해결 방법

#### 1. 데이터베이스 인덱스 확인

```bash
# MariaDB 접속
kubectl exec -it mariadb-0 -- mysql -u root -p openstack_monitor

# 인덱스 확인
SHOW INDEX FROM instances;
SHOW INDEX FROM topology_nodes;
```

#### 2. Pod 리소스 증가

```bash
# values.yaml 또는 deployment.yaml에서 리소스 증가
resources:
  requests:
    memory: "512Mi"
    cpu: "200m"
  limits:
    memory: "2Gi"
    cpu: "1000m"
```

#### 3. 데이터베이스 연결 풀 확인

- API 서비스의 데이터베이스 연결 수 확인
- 필요시 연결 풀 크기 조정

## 추가 리소스

- [배포 가이드](../DEPLOYMENT.md)
- [빠른 시작 가이드](../specs/001-openstack-monitoring/quickstart.md)
- [Secret 생성 가이드](./SECRETS.md)

