# 원격 서버 배포 가이드

이 문서는 원격 Kubernetes 클러스터에 배포할 때 발생하는 문제와 해결 방법을 설명합니다.

## ErrImageNeverPull 오류 해결

### 증상

```bash
kubectl describe pod network-collector-xxx
...
Warning  ErrImageNeverPull  Container image "network-collector:latest" is not present with pull policy of Never
```

### 원인

`imagePullPolicy: Never`로 설정되어 있지만, Pod가 스케줄된 노드에 이미지가 없습니다.

### 해결 방법

#### 방법 1: 모든 노드에 이미지 빌드 (권장)

Kubernetes 클러스터의 모든 노드에서 이미지를 빌드합니다.

```bash
# 각 노드에서 실행
# Master 노드
ssh user@k8s-master-01
cd /path/to/network-collector
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
docker build -t network-collector-api:latest -f Dockerfile.api .
cd ../frontend
docker build -t network-collector-frontend:latest -f Dockerfile .

# Worker 노드
ssh user@k8s-worker-01
cd /path/to/network-collector
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
docker build -t network-collector-api:latest -f Dockerfile.api .
cd ../frontend
docker build -t network-collector-frontend:latest -f Dockerfile .
```

#### 방법 2: 이미지를 tar로 저장하고 로드

한 노드에서 빌드한 이미지를 다른 노드로 복사합니다.

```bash
# Master 노드에서 이미지 빌드 및 저장
cd /path/to/network-collector
cd backend
docker build -t network-collector:latest -f Dockerfile.collector .
docker build -t network-collector-api:latest -f Dockerfile.api .
cd ../frontend
docker build -t network-collector-frontend:latest -f Dockerfile .

# 이미지 저장
docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o network-collector-images.tar

# 각 Worker 노드에 복사 및 로드
scp network-collector-images.tar user@k8s-worker-01:/tmp/
ssh user@k8s-worker-01 "docker load -i /tmp/network-collector-images.tar"

# Master 노드에도 로드 (필요한 경우)
docker load -i network-collector-images.tar
```

#### 방법 3: imagePullPolicy를 IfNotPresent로 변경

레지스트리를 사용하거나, 이미지를 빌드한 후 `IfNotPresent`로 변경합니다.

```bash
# Deployment 수정
kubectl patch deployment network-collector -p '{"spec":{"template":{"spec":{"containers":[{"name":"collector","imagePullPolicy":"IfNotPresent"}]}}}}'
kubectl patch deployment network-collector-api -p '{"spec":{"template":{"spec":{"containers":[{"name":"api","imagePullPolicy":"IfNotPresent"}]}}}}'
kubectl patch deployment network-collector-frontend -p '{"spec":{"template":{"spec":{"containers":[{"name":"frontend","imagePullPolicy":"IfNotPresent"}]}}}}'
```

또는 매니페스트 파일을 수정:

```yaml
# backend/deployments/k8s/collector/deployment.yaml
imagePullPolicy: IfNotPresent  # 또는 Always (레지스트리 사용 시)
```

#### 방법 4: NodeSelector를 사용하여 특정 노드에만 스케줄

이미지가 있는 노드에만 Pod를 스케줄합니다.

```yaml
# Deployment에 추가
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/hostname: k8s-worker-01  # 이미지가 있는 노드
```

## 빠른 해결 스크립트

다음 스크립트를 사용하여 모든 노드에 이미지를 배포할 수 있습니다:

```bash
#!/bin/bash
# deploy-images-to-nodes.sh

NODES=("k8s-master-01" "k8s-worker-01")
PROJECT_PATH="/path/to/network-collector"

for node in "${NODES[@]}"; do
    echo "Deploying images to $node..."
    
    # 이미지 빌드 (한 노드에서만)
    if [ "$node" == "k8s-master-01" ]; then
        ssh user@$node "cd $PROJECT_PATH/backend && \
            docker build -t network-collector:latest -f Dockerfile.collector . && \
            docker build -t network-collector-api:latest -f Dockerfile.api . && \
            cd ../frontend && \
            docker build -t network-collector-frontend:latest -f Dockerfile ."
        
        # 이미지 저장
        ssh user@$node "docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o /tmp/network-collector-images.tar"
    fi
    
    # 다른 노드에 복사 및 로드
    if [ "$node" != "k8s-master-01" ]; then
        scp user@k8s-master-01:/tmp/network-collector-images.tar /tmp/
        ssh user@$node "docker load -i /tmp/network-collector-images.tar"
    fi
done

echo "Image deployment completed!"
```

## 확인

```bash
# 각 노드에서 이미지 확인
ssh user@k8s-worker-01 "docker images | grep network-collector"

# Pod 재시작
kubectl rollout restart deployment/network-collector
kubectl rollout restart deployment/network-collector-api
kubectl rollout restart deployment/network-collector-frontend

# Pod 상태 확인
kubectl get pods -o wide
```

## 프로덕션 권장 사항

프로덕션 환경에서는 다음을 권장합니다:

1. **Docker 레지스트리 사용**: 이미지를 중앙 레지스트리에 푸시
2. **imagePullPolicy: Always 또는 IfNotPresent**: 레지스트리에서 자동으로 Pull
3. **이미지 태그 버전 관리**: `latest` 대신 버전 태그 사용 (예: `v1.0.0`)

