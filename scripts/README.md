# 배포 스크립트

이 디렉토리에는 OpenStack 모니터링 시스템의 배포 및 관리를 위한 스크립트가 포함되어 있습니다.

## 스크립트 목록

### build-images.sh

Docker 이미지를 빌드하는 스크립트입니다.

**사용법:**
```bash
# 로컬 이미지 빌드
./scripts/build-images.sh

# 레지스트리에 푸시할 이미지 빌드
./scripts/build-images.sh your-registry.io latest
```

**예시:**
```bash
# 로컬 이미지 빌드
./scripts/build-images.sh

# Docker Hub에 푸시할 이미지 빌드
./scripts/build-images.sh your-username latest

# 프라이빗 레지스트리에 푸시할 이미지 빌드
./scripts/build-images.sh registry.example.com v1.0.0
```

### deploy.sh

Kubernetes 또는 Helm을 사용하여 시스템을 배포하는 스크립트입니다.

**사용법:**
```bash
# Kubernetes 매니페스트로 배포
./scripts/deploy.sh k8s [namespace]

# Helm Chart로 배포
./scripts/deploy.sh helm [namespace]
```

**예시:**
```bash
# 기본 네임스페이스에 Kubernetes로 배포
./scripts/deploy.sh k8s

# 특정 네임스페이스에 Helm으로 배포
./scripts/deploy.sh helm monitoring

# 환경 변수로 이미지 레지스트리 설정
IMAGE_REGISTRY=your-registry.io/ IMAGE_TAG=v1.0.0 ./scripts/deploy.sh helm
```

**환경 변수:**
- `IMAGE_REGISTRY`: 이미지 레지스트리 URL (예: `your-registry.io/`)
- `IMAGE_TAG`: 이미지 태그 (기본값: `latest`)

### setup-ssh-keys.sh

Kubernetes 노드들에 SSH 키를 배포하는 스크립트입니다. `deploy-images-to-nodes.sh`를 사용하기 전에 먼저 실행하는 것을 권장합니다.

**사용법:**
```bash
# 자동으로 노드 감지하여 SSH 키 배포
./scripts/setup-ssh-keys.sh

# 특정 노드에 SSH 키 배포
./scripts/setup-ssh-keys.sh k8s-master-01 k8s-worker-01 k8s-worker-02
```

**예시:**
```bash
# 모든 노드에 SSH 키 배포
./scripts/setup-ssh-keys.sh

# 특정 노드에 SSH 키 배포 (user@host 형식)
./scripts/setup-ssh-keys.sh user@k8s-master-01 user@k8s-worker-01
```

**기능:**
- SSH 키 자동 생성 (없는 경우)
- 각 노드의 `~/.ssh/authorized_keys`에 공개키 추가
- 비밀번호 없이 접속 가능하도록 설정
- SSH 키 위치: `~/.ssh/id_rsa_network_collector`

### deploy-images-to-nodes.sh

모든 Kubernetes 노드에 Docker 이미지를 배포하는 스크립트입니다.

**사용법:**
```bash
# 자동으로 노드 감지하여 이미지 배포
./scripts/deploy-images-to-nodes.sh

# 특정 노드에 이미지 배포
./scripts/deploy-images-to-nodes.sh k8s-master-01 k8s-worker-01
```

**예시:**
```bash
# 모든 노드에 이미지 배포
./scripts/deploy-images-to-nodes.sh

# 특정 노드에 이미지 배포
./scripts/deploy-images-to-nodes.sh k8s-master-01 k8s-worker-01 k8s-worker-02
```

**주의사항:**
- `setup-ssh-keys.sh`를 먼저 실행하여 SSH 키를 설정하는 것을 권장합니다.
- SSH 키가 설정되어 있으면 자동으로 사용하며, 없으면 기본 SSH 키를 사용합니다.
- 첫 번째 노드에서 이미지를 빌드하고, 나머지 노드에 배포합니다.

### undeploy.sh

배포된 시스템을 제거하는 스크립트입니다.

**사용법:**
```bash
# Kubernetes 배포 제거
./scripts/undeploy.sh k8s [namespace]

# Helm 배포 제거
./scripts/undeploy.sh helm [namespace]
```

**예시:**
```bash
# 기본 네임스페이스에서 Kubernetes 배포 제거
./scripts/undeploy.sh k8s

# 특정 네임스페이스에서 Helm 배포 제거
./scripts/undeploy.sh helm monitoring
```

### check-metrics.sh

메트릭 수집 기능이 정상적으로 동작하는지 확인하는 스크립트입니다.

**사용법:**
```bash
# 메트릭 수집 상태 확인
./scripts/check-metrics.sh
```

**기능:**
- 메트릭 테이블 존재 여부 확인
- 메트릭 데이터 존재 여부 확인
- 최근 메트릭 상세 정보 조회
- Collector Pod 로그 확인
- API 엔드포인트 동작 확인

**예시:**
```bash
# 메트릭 수집 상태 확인
./scripts/check-metrics.sh

# 환경 변수로 데이터베이스 정보 설정
DB_HOST=mariadb DB_PORT=3306 DB_USER=openstack_monitor DB_PASSWORD=your-password ./scripts/check-metrics.sh
```

**확인 항목:**
1. `instance_metrics` 테이블 및 데이터
2. `network_metrics` 테이블 및 데이터
3. `hypervisor_metrics` 테이블 및 데이터
4. Collector Pod의 메트릭 수집 로그
5. 메트릭 API 엔드포인트 동작

**참고:** 자세한 내용은 `docs/METRICS_COLLECTION.md`를 참조하세요.

## 전체 배포 워크플로우

### 로컬 이미지 사용 (imagePullPolicy: Never)

#### 1. SSH 키 설정 (권장)

```bash
# 모든 노드에 SSH 키 배포
./scripts/setup-ssh-keys.sh
```

#### 2. 이미지 빌드 및 배포

```bash
# 모든 노드에 이미지 빌드 및 배포
./scripts/deploy-images-to-nodes.sh

# 또는 특정 노드에만 배포
./scripts/deploy-images-to-nodes.sh k8s-master-01 k8s-worker-01
```

#### 3. 배포

```bash
# Kubernetes로 배포
./scripts/deploy.sh k8s
```

#### 4. 상태 확인

```bash
kubectl get pods
kubectl get services
kubectl logs -f deployment/network-collector
```

### 레지스트리 사용 (imagePullPolicy: Always/IfNotPresent)

#### 1. 이미지 빌드 및 푸시

```bash
# 이미지 빌드
./scripts/build-images.sh your-registry.io latest

# 이미지 푸시 (Docker Hub 예시)
docker push your-registry.io/network-collector:latest
docker push your-registry.io/network-collector-api:latest
docker push your-registry.io/network-collector-frontend:latest
```

#### 2. 배포

```bash
# Helm으로 배포
IMAGE_REGISTRY=your-registry.io/ IMAGE_TAG=latest ./scripts/deploy.sh helm
```

#### 3. 상태 확인

```bash
kubectl get pods
kubectl get services
kubectl logs -f deployment/network-collector
```

### 4. 제거

```bash
./scripts/undeploy.sh helm
```

## 주의사항

1. **Secret 관리**: 배포 스크립트는 Secret 값을 대화형으로 입력받습니다. 프로덕션 환경에서는 Secret 관리 도구(예: Sealed Secrets, External Secrets)를 사용하는 것이 좋습니다.

2. **이미지 레지스트리**: 이미지를 빌드한 후 반드시 레지스트리에 푸시해야 Kubernetes에서 Pull할 수 있습니다.

3. **네임스페이스**: 기본 네임스페이스는 `default`입니다. 프로덕션 환경에서는 별도의 네임스페이스를 사용하는 것이 좋습니다.

4. **데이터 백업**: 제거 전에 데이터베이스 데이터를 백업하는 것을 권장합니다.

## 트러블슈팅

### 스크립트 실행 권한 오류

```bash
chmod +x scripts/*.sh
```

### kubectl 연결 오류

```bash
kubectl cluster-info
# 클러스터 연결 확인
```

### 이미지 Pull 실패

1. 이미지가 레지스트리에 푸시되었는지 확인
2. Kubernetes 노드에서 레지스트리 접근 가능한지 확인
3. imagePullSecrets 설정 확인

