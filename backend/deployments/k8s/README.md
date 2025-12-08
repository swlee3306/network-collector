# Kubernetes Deployment Manifests

이 디렉토리는 OpenStack 모니터링 시스템의 Kubernetes 배포 매니페스트를 포함합니다.

## 파일 구조

```
k8s/
├── configmap.yaml          # ConfigMap (비밀 정보 제외 설정)
├── secret.yaml             # Secret (비밀 정보)
├── mariadb/
│   └── statefulset.yaml    # MariaDB StatefulSet 및 Service
├── collector/
│   └── deployment.yaml     # Collector Deployment
├── api/
│   └── deployment.yaml     # API Service Deployment 및 Service
├── frontend/
│   └── deployment.yaml     # Frontend Deployment 및 Service
├── ingress.yaml            # Ingress 리소스 (선택적)
└── README.md              # 이 파일
```

## 배포 순서

1. **ConfigMap 및 Secret 생성**:
   ```bash
   kubectl apply -f configmap.yaml
   kubectl apply -f secret.yaml
   ```

2. **MariaDB 배포**:
   ```bash
   kubectl apply -f mariadb/statefulset.yaml
   ```

3. **Collector 배포**:
   ```bash
   kubectl apply -f collector/deployment.yaml
   ```

4. **API Service 배포**:
   ```bash
   kubectl apply -f api/deployment.yaml
   ```

5. **Frontend 배포**:
   ```bash
   kubectl apply -f frontend/deployment.yaml
   ```

6. **Ingress 배포** (선택적):
   ```bash
   kubectl apply -f ingress.yaml
   ```

## Secret 설정

배포 전에 `secret.yaml` 파일의 다음 값들을 실제 값으로 변경해야 합니다:

- `DB_PASSWORD`: MariaDB 비밀번호
- `OPENSTACK_AUTH_URL`: OpenStack Keystone URL
- `OPENSTACK_USERNAME`: OpenStack 사용자명
- `OPENSTACK_PASSWORD`: OpenStack 비밀번호
- `OPENSTACK_PROJECT_ID`: OpenStack 프로젝트 ID
- `JWT_SECRET`: JWT 토큰 서명 키
- `AUTH_TOKEN`: API 인증 토큰

## 이미지 빌드

배포 전에 다음 이미지들을 빌드하고 레지스트리에 푸시해야 합니다:

```bash
# Collector 이미지
docker build -t network-collector:latest -f backend/Dockerfile.collector backend/

# API 이미지
docker build -t network-collector-api:latest -f backend/Dockerfile.api backend/

# Frontend 이미지
docker build -t network-collector-frontend:latest -f frontend/Dockerfile frontend/
```

## 확인

배포 상태 확인:

```bash
# Pod 상태 확인
kubectl get pods

# 서비스 확인
kubectl get services

# 로그 확인
kubectl logs -f deployment/network-collector
kubectl logs -f deployment/network-collector-api
kubectl logs -f deployment/network-collector-frontend
```

## 트러블슈팅

### Pod가 시작되지 않음

1. Secret과 ConfigMap이 올바르게 생성되었는지 확인:
   ```bash
   kubectl get configmap network-collector-config
   kubectl get secret network-collector-secrets
   ```

2. 이미지가 레지스트리에 있는지 확인

3. Pod 이벤트 확인:
   ```bash
   kubectl describe pod <pod-name>
   ```

### 데이터베이스 연결 실패

1. MariaDB Pod가 실행 중인지 확인:
   ```bash
   kubectl get pods -l app=mariadb
   ```

2. MariaDB 서비스 확인:
   ```bash
   kubectl get svc mariadb
   kubectl get svc mariadb-nodeport  # NodePort Service 확인
   ```

3. 데이터베이스 비밀번호가 올바른지 확인

### MariaDB 외부 접속

MariaDB는 NodePort Service를 통해 외부에서 접속할 수 있습니다:

1. **NodePort Service 확인**:
   ```bash
   kubectl get svc mariadb-nodeport
   ```

2. **외부 접속 정보**:
   - **포트**: 30306 (기본값)
   - **호스트**: Kubernetes 노드 IP 주소
   - **접속 예시**:
     ```bash
     # 노드 IP 확인
     kubectl get nodes -o wide
     
     # MySQL 클라이언트로 접속
     mysql -h <node-ip> -P 30306 -u openstack_monitor -p
     ```

3. **보안 주의사항**:
   - NodePort는 모든 노드의 IP에서 접속 가능합니다
   - 프로덕션 환경에서는 방화벽 규칙을 설정하여 접속을 제한하는 것이 좋습니다
   - 또는 NodePort를 비활성화하고 `kubectl port-forward`를 사용할 수 있습니다:
     ```bash
     kubectl port-forward svc/mariadb 3306:3306
     ```

### API가 응답하지 않음

1. API Pod 상태 확인:
   ```bash
   kubectl get pods -l app=network-collector-api
   ```

2. 헬스 체크:
   ```bash
   kubectl port-forward svc/network-collector-api 8080:8080
   curl http://localhost:8080/health
   ```

