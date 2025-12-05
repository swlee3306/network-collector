# 기존 Docker 이미지 사용 가이드

이미 빌드되어 있는 Docker 이미지를 Kubernetes에서 사용하는 방법을 설명합니다.

## 현재 이미지 확인

먼저 현재 Docker에 있는 이미지를 확인합니다:

```bash
docker images | grep -E "network-collector|IMAGE"
```

출력 예시:
```
REPOSITORY                    TAG       IMAGE ID       CREATED         SIZE
network-collector             latest    abc123def456   2 hours ago     50MB
network-collector-api         latest    def456ghi789   2 hours ago     45MB
network-collector-frontend    latest    ghi789jkl012   2 hours ago     120MB
my-registry/network-collector v1.0.0    xyz789abc123   1 day ago      50MB
```

## 방법 1: 이미지 태그 변경

이미지 이름이 다르면 올바른 태그로 변경합니다:

```bash
# 예: 다른 이름의 이미지를 network-collector:latest로 태그
docker tag my-existing-image:tag network-collector:latest
docker tag my-existing-image-api:tag network-collector-api:latest
docker tag my-existing-image-frontend:tag network-collector-frontend:latest

# 확인
docker images | grep network-collector
```

## 방법 2: Deployment에서 이미지 이름 수정

Deployment 매니페스트에서 실제 이미지 이름을 사용하도록 수정합니다:

```bash
# 현재 이미지 이름 확인
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}"

# 예: 이미지가 my-collector:1.0.0으로 되어 있다면
kubectl set image deployment/network-collector collector=my-collector:1.0.0
kubectl set image deployment/network-collector-api api=my-api:1.0.0
kubectl set image deployment/network-collector-frontend frontend=my-frontend:1.0.0
```

또는 매니페스트 파일 직접 수정:

```yaml
# backend/deployments/k8s/collector/deployment.yaml
image: my-collector:1.0.0  # 실제 이미지 이름으로 변경
```

## 방법 3: 이미지를 다른 노드로 복사

한 노드에 있는 이미지를 다른 노드로 복사합니다:

### 이미지 저장 및 로드

```bash
# 이미지가 있는 노드에서
docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o network-collector-images.tar

# 다른 노드로 복사
scp network-collector-images.tar user@k8s-worker-01:/tmp/

# 다른 노드에서 로드
ssh user@k8s-worker-01 "docker load -i /tmp/network-collector-images.tar"

# 확인
ssh user@k8s-worker-01 "docker images | grep network-collector"
```

### 이미지 이름 확인 및 태그

```bash
# 로드된 이미지 확인
docker images

# 이미지 ID로 태그 변경 (필요한 경우)
docker tag <IMAGE_ID> network-collector:latest
docker tag <IMAGE_ID> network-collector-api:latest
docker tag <IMAGE_ID> network-collector-frontend:latest
```

## 방법 4: 스크립트를 사용한 자동 처리

기존 이미지를 자동으로 처리하는 스크립트:

```bash
#!/bin/bash
# use-existing-images.sh

# 현재 이미지 확인
echo "=== 현재 Docker 이미지 ==="
docker images | grep -E "network-collector|collector|api|frontend"

# 이미지 이름 입력
read -p "Collector 이미지 이름 (예: network-collector:latest): " COLLECTOR_IMAGE
read -p "API 이미지 이름 (예: network-collector-api:latest): " API_IMAGE
read -p "Frontend 이미지 이름 (예: network-collector-frontend:latest): " FRONTEND_IMAGE

# 표준 이름으로 태그
if [ "$COLLECTOR_IMAGE" != "network-collector:latest" ]; then
    docker tag "$COLLECTOR_IMAGE" network-collector:latest
    echo "Tagged $COLLECTOR_IMAGE as network-collector:latest"
fi

if [ "$API_IMAGE" != "network-collector-api:latest" ]; then
    docker tag "$API_IMAGE" network-collector-api:latest
    echo "Tagged $API_IMAGE as network-collector-api:latest"
fi

if [ "$FRONTEND_IMAGE" != "network-collector-frontend:latest" ]; then
    docker tag "$FRONTEND_IMAGE" network-collector-frontend:latest
    echo "Tagged $FRONTEND_IMAGE as network-collector-frontend:latest"
fi

echo "=== 태그 완료 ==="
docker images | grep network-collector
```

## 방법 5: kubectl set image 사용

실행 중인 Deployment의 이미지를 직접 변경:

```bash
# 현재 이미지 확인
kubectl get deployment network-collector -o jsonpath='{.spec.template.spec.containers[0].image}'

# 이미지 변경
kubectl set image deployment/network-collector collector=network-collector:latest
kubectl set image deployment/network-collector-api api=network-collector-api:latest
kubectl set image deployment/network-collector-frontend frontend=network-collector-frontend:latest

# 롤아웃 확인
kubectl rollout status deployment/network-collector
```

## 빠른 해결 (이미지가 한 노드에 있는 경우)

### 1. 이미지 이름 확인

```bash
docker images
```

### 2. 표준 이름으로 태그 (필요한 경우)

```bash
# 예: 이미지가 다른 이름으로 되어 있다면
docker tag existing-collector:tag network-collector:latest
docker tag existing-api:tag network-collector-api:latest
docker tag existing-frontend:tag network-collector-frontend:latest
```

### 3. 다른 노드로 복사

```bash
# 이미지 저장
docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o images.tar

# 각 노드에 복사 및 로드
for node in k8s-master-01 k8s-worker-01; do
    scp images.tar user@$node:/tmp/
    ssh user@$node "docker load -i /tmp/images.tar && rm /tmp/images.tar"
done
```

### 4. Deployment 재적용

```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
kubectl apply -f backend/deployments/k8s/api/deployment.yaml
kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml
```

## 확인

```bash
# 각 노드에서 이미지 확인
kubectl get nodes -o wide
for node in $(kubectl get nodes -o name | cut -d/ -f2); do
    echo "=== $node ==="
    ssh user@$node "docker images | grep network-collector" || echo "이미지 없음"
done

# Pod 상태 확인
kubectl get pods -o wide

# Pod 이벤트 확인
kubectl describe pod <pod-name>
```

## 주의사항

1. **이미지 이름 일치**: Deployment의 `image` 필드와 Docker 이미지 이름이 일치해야 합니다.
2. **모든 노드에 이미지**: Pod가 스케줄될 수 있는 모든 노드에 이미지가 있어야 합니다.
3. **이미지 태그**: `latest` 태그 대신 버전 태그 사용을 권장합니다 (프로덕션 환경).

## 예시 시나리오

### 시나리오 1: 이미지가 다른 이름으로 되어 있음

```bash
# 현재 이미지
docker images
# REPOSITORY          TAG       IMAGE ID
# my-collector        v1.0       abc123
# my-api              v1.0       def456
# my-frontend         v1.0       ghi789

# 표준 이름으로 태그
docker tag my-collector:v1.0 network-collector:latest
docker tag my-api:v1.0 network-collector-api:latest
docker tag my-frontend:v1.0 network-collector-frontend:latest
```

### 시나리오 2: 이미지가 레지스트리에 있음

```bash
# 레지스트리에서 Pull
docker pull my-registry.io/network-collector:latest
docker pull my-registry.io/network-collector-api:latest
docker pull my-registry.io/network-collector-frontend:latest

# 로컬 태그로 변경 (선택적)
docker tag my-registry.io/network-collector:latest network-collector:latest
docker tag my-registry.io/network-collector-api:latest network-collector-api:latest
docker tag my-registry.io/network-collector-frontend:latest network-collector-frontend:latest
```

### 시나리오 3: 이미지가 tar 파일로 있음

```bash
# 이미지 로드
docker load -i network-collector-images.tar

# 이미지 확인
docker images

# 필요시 태그 변경
docker tag <loaded-image-id> network-collector:latest
```

