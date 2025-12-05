# 배포 문제 해결 가이드

## 배포 전 점검 사항

### 1. 이미지 확인

배포 전에 마스터 노드에 이미지가 있는지 확인하세요:

```bash
# 마스터 노드에서 실행
docker images | grep network-collector
```

필요한 이미지:
- `network-collector:latest`
- `network-collector-api:latest`
- `network-collector-frontend:latest`

이미지가 없으면:
```bash
./scripts/build-images.sh
```

### 2. 마스터 노드 라벨 확인

```bash
kubectl get nodes --show-labels | grep master
# 또는
kubectl get node k8s-master-01 --show-labels
```

예상 라벨: `node-role.kubernetes.io/control-plane=`

### 3. 마스터 노드 Taint 확인

```bash
kubectl describe node k8s-master-01 | grep Taints
```

일반적으로 `node-role.kubernetes.io/control-plane:NoSchedule` taint가 있습니다.

## 배포 프로세스

### 1단계: 이미지 빌드

**마스터 노드에서만 빌드하면 됩니다** (Pod가 마스터 노드에서만 실행되므로):

```bash
# 마스터 노드에서 실행
cd ~/network-collector
./scripts/build-images.sh
```

### 2단계: 배포

```bash
# 마스터 노드에서 실행
./scripts/deploy.sh
```

## 문제 해결

### ImagePullBackOff / ErrImagePull

**증상:**
```
Events:
  Warning  Failed     15s (x3 over 54s)  kubelet  Error: ErrImagePull
  Warning  Failed     15s (x3 over 54s)  kubelet  Failed to pull image "network-collector:latest": failed to resolve image
```

**원인:**
1. 이미지가 마스터 노드에 없음
2. `imagePullPolicy`가 잘못 설정됨
3. Pod가 워커 노드에 스케줄됨 (이미지가 워커 노드에 없음)

**해결 방법:**

1. **이미지 확인:**
```bash
# 마스터 노드에서
docker images | grep network-collector
```

2. **이미지가 없으면 빌드:**
```bash
./scripts/build-images.sh
```

3. **Pod가 어느 노드에 있는지 확인:**
```bash
kubectl get pods -o wide
```

4. **Pod가 워커 노드에 있으면:**
   - Deployment의 `nodeSelector` 확인
   - Pod 삭제 후 재생성:
```bash
kubectl delete pod <pod-name>
```

5. **Deployment 설정 확인:**
```bash
kubectl get deployment network-collector -o yaml | grep -A 5 nodeSelector
kubectl get deployment network-collector -o yaml | grep imagePullPolicy
```

### Pod가 Pending 상태

**증상:**
```
NAME                              READY   STATUS    RESTARTS   AGE
network-collector-xxx             0/1     Pending   0          5m
```

**원인:**
1. `nodeSelector`가 맞지 않음
2. `tolerations`가 없어서 마스터 노드에 스케줄되지 않음
3. 리소스 부족

**해결 방법:**

1. **Pod 이벤트 확인:**
```bash
kubectl describe pod <pod-name>
```

2. **nodeSelector 확인:**
```bash
kubectl get deployment network-collector -o yaml | grep -A 3 nodeSelector
```

3. **tolerations 확인:**
```bash
kubectl get deployment network-collector -o yaml | grep -A 5 tolerations
```

4. **Deployment 재적용:**
```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
kubectl delete pod <pod-name>  # 새 설정으로 재생성
```

### Pod가 CrashLoopBackOff 상태

**증상:**
```
NAME                              READY   STATUS             RESTARTS   AGE
network-collector-xxx             0/1     CrashLoopBackOff   5          2m
```

**원인:**
1. 애플리케이션 오류
2. 환경 변수 누락
3. 데이터베이스 연결 실패

**해결 방법:**

1. **Pod 로그 확인:**
```bash
kubectl logs <pod-name>
kubectl logs <pod-name> --previous  # 이전 컨테이너 로그
```

2. **환경 변수 확인:**
```bash
kubectl describe pod <pod-name> | grep -A 20 "Environment:"
```

3. **ConfigMap 및 Secret 확인:**
```bash
kubectl get configmap network-collector-config
kubectl get secret network-collector-secrets
```

### Pod가 워커 노드에 스케줄됨

**증상:**
```
NAME                              READY   STATUS    NODE
network-collector-xxx             0/1     Pending   k8s-worker-01
```

**원인:**
- `nodeSelector`가 적용되지 않음
- Deployment가 업데이트되지 않음

**해결 방법:**

1. **기존 Pod 삭제:**
```bash
kubectl delete pods -l app=network-collector
```

2. **Deployment 재적용:**
```bash
kubectl apply -f backend/deployments/k8s/collector/deployment.yaml
kubectl apply -f backend/deployments/k8s/api/deployment.yaml
kubectl apply -f backend/deployments/k8s/frontend/deployment.yaml
```

3. **Pod 재생성 확인:**
```bash
kubectl get pods -o wide
```

## 진단 스크립트 사용

자동 진단 스크립트를 사용하여 문제를 확인하세요:

```bash
./scripts/check-deployment.sh
```

이 스크립트는 다음을 확인합니다:
- 마스터 노드 설정
- 이미지 존재 여부
- Deployment 설정
- Pod 상태
- ConfigMap 및 Secret

## 체크리스트

배포 전 확인 사항:

- [ ] 마스터 노드에서 이미지 빌드 완료
- [ ] 마스터 노드에 이미지 존재 확인 (`docker images | grep network-collector`)
- [ ] 마스터 노드 라벨 확인 (`node-role.kubernetes.io/control-plane=`)
- [ ] Deployment 파일의 `nodeSelector` 확인
- [ ] Deployment 파일의 `tolerations` 확인
- [ ] Deployment 파일의 `imagePullPolicy: Never` 확인
- [ ] ConfigMap 및 Secret 생성 확인
- [ ] MariaDB가 실행 중인지 확인

## 추가 정보 요청

문제가 계속되면 다음 정보를 제공해주세요:

1. **Pod 상태:**
```bash
kubectl get pods -o wide
```

2. **Pod 상세 정보:**
```bash
kubectl describe pod <pod-name>
```

3. **Pod 로그:**
```bash
kubectl logs <pod-name>
```

4. **Deployment 설정:**
```bash
kubectl get deployment network-collector -o yaml
```

5. **노드 정보:**
```bash
kubectl get nodes --show-labels
```

6. **이미지 목록:**
```bash
docker images | grep network-collector
```

7. **진단 스크립트 결과:**
```bash
./scripts/check-deployment.sh
```

