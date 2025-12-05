# Image Pull Policy 가이드

## 문제 상황

Kubernetes는 containerd나 CRI-O 같은 컨테이너 런타임을 사용하는데, Docker에 있는 이미지를 Kubernetes가 직접 인식하지 못할 수 있습니다.

### 증상

- `docker images`로는 이미지가 보이지만
- Pod는 `ErrImagePull` 오류 발생
- `imagePullPolicy: IfNotPresent`로 설정했는데도 레지스트리에서 pull 시도

### 원인

1. **컨테이너 런타임 차이**: Kubernetes는 Docker가 아닌 containerd/CRI-O를 사용
2. **이미지 저장소 분리**: Docker 이미지와 containerd 이미지는 별도로 관리됨
3. **Pull Policy 동작**: `IfNotPresent`는 컨테이너 런타임의 이미지 저장소를 확인

## 해결 방법

### 방법 1: imagePullPolicy: Never + 워커 노드 배포 (권장)

**장점:**
- 가장 확실한 방법
- 레지스트리 불필요
- 네트워크 트래픽 없음

**단점:**
- 워커 노드에 이미지 배포 필요

**절차:**

1. `imagePullPolicy: Never` 설정 (이미 완료됨)
2. 마스터 노드에서 이미지 빌드:
```bash
./scripts/build-images.sh
```

3. 워커 노드에 이미지 배포:
```bash
./scripts/deploy-images-to-nodes.sh
```

4. Deployment 재적용:
```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
kubectl apply -f backend/deployments/k8s/api/deployment.yaml
kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml
```

5. Pod 재시작:
```bash
kubectl delete pods -l 'app in (network-collector,network-collector-api,network-collector-frontend)'
```

### 방법 2: containerd로 이미지 import

**장점:**
- Docker 이미지를 containerd로 변환
- `IfNotPresent` 정책 사용 가능

**단점:**
- 각 노드에서 수동 작업 필요
- containerd가 설치되어 있어야 함

**절차:**

1. 마스터 노드에서 이미지 빌드:
```bash
./scripts/build-images.sh
```

2. containerd로 import:
```bash
./scripts/import-images-to-containerd.sh
```

3. 워커 노드에도 동일하게 import:
```bash
# 각 워커 노드에서
./scripts/import-images-to-containerd.sh
```

### 방법 3: 로컬 레지스트리 사용

**장점:**
- 표준적인 방법
- 모든 노드에서 동일하게 작동

**단점:**
- 레지스트리 설정 필요
- 추가 인프라 필요

## 현재 설정

현재 Deployment는 `imagePullPolicy: Never`로 설정되어 있습니다.

이 설정은:
- ✅ 로컬에 이미지가 있으면 사용
- ❌ 로컬에 이미지가 없으면 에러 (pull 시도 안 함)

따라서 **워커 노드에도 이미지를 배포해야 합니다**.

## 이미지 확인 방법

### Docker 사용하는 경우
```bash
docker images | grep network-collector
```

### containerd 사용하는 경우
```bash
crictl images | grep network-collector
# 또는
ctr -n k8s.io images list | grep network-collector
```

## 문제 해결 체크리스트

- [ ] 마스터 노드에 이미지 빌드 완료
- [ ] 워커 노드에 이미지 배포 완료
- [ ] `imagePullPolicy: Never` 설정 확인
- [ ] Pod가 올바른 노드에 스케줄됨
- [ ] 컨테이너 런타임에서 이미지 확인

## 추가 정보

- Kubernetes는 기본적으로 containerd를 사용합니다
- Docker Desktop이나 일부 환경에서는 Docker를 사용할 수 있지만, 프로덕션에서는 보통 containerd를 사용합니다
- `docker images`와 `crictl images`는 서로 다른 이미지 저장소를 보여줍니다

