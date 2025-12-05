# 워커 노드 배포 가이드

워커 노드에서도 Pod가 실행되도록 설정하는 방법입니다.

## 변경 사항

1. **`nodeSelector` 제거**: 모든 노드에서 실행 가능
2. **`imagePullPolicy: IfNotPresent`**: 로컬에 이미지가 없으면 pull 시도
3. **`tolerations` 유지**: 마스터 노드에서도 실행 가능

## 이미지 배포 방법

워커 노드에서 Pod가 실행되려면 워커 노드에도 이미지가 있어야 합니다.

### 방법 1: 자동 배포 스크립트 사용 (권장)

```bash
# 마스터 노드에서 실행
cd ~/network-collector
./scripts/deploy-images-to-nodes.sh
```

이 스크립트는:
- 모든 Kubernetes 노드를 자동 감지
- 마스터 노드의 이미지를 tar로 저장
- 각 워커 노드에 SSH로 이미지 전송 및 로드

### 방법 2: 수동 배포

#### 2-1. 마스터 노드에서 이미지 저장

```bash
# 마스터 노드에서 실행
docker save network-collector:latest network-collector-api:latest network-collector-frontend:latest -o network-collector-images.tar
```

#### 2-2. 워커 노드로 전송 및 로드

```bash
# 각 워커 노드에 대해 실행
scp network-collector-images.tar user@k8s-worker-01:/tmp/
ssh user@k8s-worker-01 "docker load -i /tmp/network-collector-images.tar"
```

### 방법 3: 워커 노드에서 직접 빌드

워커 노드에 Docker가 있다면:

```bash
# 각 워커 노드에서 실행
cd ~/network-collector
./scripts/build-images.sh
```

## 배포 후 확인

### 1. Pod가 워커 노드에 스케줄되었는지 확인

```bash
kubectl get pods -o wide
```

예상 결과:
```
NAME                                  READY   STATUS    NODE
network-collector-xxx                 1/1     Running   k8s-worker-01
network-collector-api-xxx             1/1     Running   k8s-master-01
network-collector-api-xxx             1/1     Running   k8s-worker-01
network-collector-frontend-xxx        1/1     Running   k8s-worker-01
network-collector-frontend-xxx        1/1     Running   k8s-master-01
```

### 2. 워커 노드에 이미지가 있는지 확인

```bash
# 워커 노드에서 실행
docker images | grep network-collector
```

## 문제 해결

### Pod가 워커 노드에서 ImagePullBackOff

**원인**: 워커 노드에 이미지가 없고 레지스트리에서 pull할 수 없음

**해결**:
1. 워커 노드에 이미지 배포 (위의 방법 1 또는 2 사용)
2. 또는 `imagePullPolicy: Never`로 변경하고 마스터 노드에만 이미지 배포

### Pod가 마스터 노드에만 스케줄됨

**원인**: 워커 노드에 리소스 부족 또는 taint

**해결**:
```bash
# 워커 노드 리소스 확인
kubectl describe node k8s-worker-01

# 워커 노드 taint 확인
kubectl describe node k8s-worker-01 | grep Taints
```

## 마스터 노드 전용으로 되돌리기

만약 다시 마스터 노드에서만 실행하고 싶다면:

```yaml
spec:
  template:
    spec:
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
      imagePullPolicy: Never
```

또는 Deployment 파일을 이전 버전으로 되돌리기:

```bash
git checkout HEAD -- backend/deployments/k8s/*/deployment.yaml
```

