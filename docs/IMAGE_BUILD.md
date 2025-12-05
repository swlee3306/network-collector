# Docker 이미지 빌드 및 배포 가이드

이 문서는 OpenStack 모니터링 시스템의 Docker 이미지를 빌드하고 Kubernetes 클러스터에 배포하는 방법을 설명합니다.

## 문제: ImagePullBackOff 오류

원격 Kubernetes 클러스터에서 배포할 때 `ImagePullBackOff` 오류가 발생하는 이유:

1. **이미지가 레지스트리에 없음**: 로컬에서만 빌드되어 원격 클러스터에서 접근 불가
2. **이미지 이름에 레지스트리 경로 없음**: `network-collector:latest`는 로컬 이미지로 인식
3. **Dockerfile이 없음**: 이미지를 빌드할 수 없음

## 해결 방법

### 방법 1: 원격 서버에서 직접 빌드 (권장)

원격 Kubernetes 클러스터의 노드에서 직접 이미지를 빌드하는 방법입니다.

#### 1. 원격 서버에 프로젝트 복사

```bash
# 로컬에서
scp -r network-collector user@remote-server:/path/to/

# 원격 서버에서
cd /path/to/network-collector
```

#### 2. 이미지 빌드

```bash
# 모든 이미지 빌드
./scripts/build-images.sh

# 또는 개별 빌드
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
docker build -t network-collector-api:latest -f Dockerfile.api .
cd ../frontend
docker build -t network-collector-frontend:latest -f Dockerfile .
```

#### 3. 이미지 로드 (Kubernetes 노드가 다른 경우)

Kubernetes 클러스터의 모든 노드에서 이미지를 사용할 수 있도록 해야 합니다:

**옵션 A: Docker 이미지를 tar로 저장하고 로드**

```bash
# 이미지 저장
docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o network-collector-images.tar

# 각 노드에 복사 및 로드
scp network-collector-images.tar user@node1:/tmp/
ssh user@node1 "docker load -i /tmp/network-collector-images.tar"
```

**옵션 B: imagePullPolicy를 Never로 변경**

Deployment 매니페스트에서 `imagePullPolicy: Never`로 설정하면 로컬 이미지를 사용합니다.

### 방법 2: 레지스트리에 푸시 (프로덕션 권장)

Docker 레지스트리(Docker Hub, Harbor, ECR 등)에 이미지를 푸시하는 방법입니다.

#### 1. 이미지 빌드 및 태깅

```bash
# 레지스트리 주소 설정
REGISTRY="your-registry.io"  # 예: docker.io/your-username 또는 harbor.example.com/project

# 이미지 빌드 및 태깅
./scripts/build-images.sh $REGISTRY latest

# 또는 수동으로
docker build -t $REGISTRY/network-collector:latest -f backend/Dockerfile.collector backend/
docker build -t $REGISTRY/network-collector-api:latest -f backend/Dockerfile.api backend/
docker build -t $REGISTRY/network-collector-frontend:latest -f frontend/Dockerfile frontend/
```

#### 2. 레지스트리에 로그인

```bash
# Docker Hub
docker login

# Harbor 또는 기타 레지스트리
docker login your-registry.io
```

#### 3. 이미지 푸시

```bash
docker push $REGISTRY/network-collector:latest
docker push $REGISTRY/network-collector-api:latest
docker push $REGISTRY/network-collector-frontend:latest
```

#### 4. Deployment 매니페스트 수정

이미지 이름에 레지스트리 경로를 포함하도록 수정:

```yaml
# backend/deployments/k8s/collector/deployment.yaml
image: your-registry.io/network-collector:latest
imagePullPolicy: Always  # 또는 IfNotPresent
```

또는 Helm 사용 시:

```bash
helm install network-collector backend/deployments/helm \
  --set collector.image.repository=your-registry.io/network-collector \
  --set collector.image.tag=latest \
  --set api.image.repository=your-registry.io/network-collector-api \
  --set api.image.tag=latest \
  --set frontend.image.repository=your-registry.io/network-collector-frontend \
  --set frontend.image.tag=latest \
  ...
```

### 방법 3: imagePullPolicy를 Never로 변경 (개발/테스트용)

로컬에서 빌드한 이미지를 사용하는 경우, Deployment에서 `imagePullPolicy: Never`로 설정합니다.

#### Kubernetes 매니페스트 수정

```yaml
# backend/deployments/k8s/collector/deployment.yaml
image: network-collector:latest
imagePullPolicy: Never  # 로컬 이미지 사용
```

#### Helm Chart 수정

```yaml
# backend/deployments/helm/values.yaml
collector:
  image:
    pullPolicy: Never
```

## 빠른 해결 (원격 서버에서)

### 방법 A: 모든 노드에 이미지 배포 스크립트 사용

```bash
# 모든 노드에 자동으로 이미지 배포
./scripts/deploy-images-to-nodes.sh k8s-master-01 k8s-worker-01

# 또는 kubectl로 노드 목록 자동 감지
./scripts/deploy-images-to-nodes.sh
```

### 방법 B: 수동으로 각 노드에 배포

원격 서버에서 즉시 해결하려면:

### 1. Deployment 수정 (imagePullPolicy: Never)

```bash
# 원격 서버에서
kubectl patch deployment network-collector -p '{"spec":{"template":{"spec":{"containers":[{"name":"collector","imagePullPolicy":"Never"}]}}}}'
kubectl patch deployment network-collector-api -p '{"spec":{"template":{"spec":{"containers":[{"name":"api","imagePullPolicy":"Never"}]}}}}'
kubectl patch deployment network-collector-frontend -p '{"spec":{"template":{"spec":{"containers":[{"name":"frontend","imagePullPolicy":"Never"}]}}}}'
```

### 2. 이미지 빌드

```bash
# 원격 서버에서 프로젝트 디렉토리로 이동
cd /path/to/network-collector

# 이미지 빌드
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
docker build -t network-collector-api:latest -f Dockerfile.api .
cd ../frontend
docker build -t network-collector-frontend:latest -f Dockerfile .
```

### 3. Pod 재시작

```bash
kubectl rollout restart deployment/network-collector
kubectl rollout restart deployment/network-collector-api
kubectl rollout restart deployment/network-collector-frontend
```

## 확인

```bash
# Pod 상태 확인
kubectl get pods

# Pod 이벤트 확인
kubectl describe pod <pod-name>

# 이미지 확인
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[0].image}'
```

## 권장 사항

### 개발 환경
- `imagePullPolicy: Never` 사용
- 로컬에서 이미지 빌드

### 프로덕션 환경
- Docker 레지스트리 사용
- `imagePullPolicy: Always` 또는 `IfNotPresent`
- 이미지 태그에 버전 번호 사용 (예: `v1.0.0`)

## 트러블슈팅

### 이미지가 여전히 Pull되지 않음

1. **이미지 이름 확인**
   ```bash
   kubectl get deployment network-collector -o jsonpath='{.spec.template.spec.containers[0].image}'
   ```

2. **레지스트리 접근 확인**
   ```bash
   kubectl describe pod <pod-name> | grep -i "pull\|image"
   ```

3. **imagePullSecrets 확인** (프라이빗 레지스트리인 경우)
   ```bash
   kubectl get secret <registry-secret>
   ```

### 빌드 실패

1. **Dockerfile 경로 확인**
   ```bash
   ls -la backend/Dockerfile.*
   ls -la frontend/Dockerfile
   ```

2. **빌드 로그 확인**
   ```bash
   docker build -t network-collector:latest -f backend/Dockerfile.collector backend/ 2>&1 | tee build.log
   ```

