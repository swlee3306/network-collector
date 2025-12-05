# Network Collector Helm Chart

이 Helm Chart는 OpenStack 모니터링 시스템을 Kubernetes에 배포하기 위한 것입니다.

## 전제 조건

- Kubernetes 1.19+
- Helm 3.0+
- MariaDB를 위한 PersistentVolume 지원

## 설치

### 기본 설치

```bash
helm install network-collector ./helm \
  --set secrets.dbPassword=your-db-password \
  --set secrets.openstackAuthURL=http://keystone:5000/v3 \
  --set secrets.openstackUsername=admin \
  --set secrets.openstackPassword=your-openstack-password \
  --set secrets.openstackProjectID=your-project-id \
  --set secrets.jwtSecret=your-jwt-secret \
  --set secrets.authToken=your-auth-token
```

### values.yaml 파일 사용

```bash
# values.yaml 파일 편집
vim values.yaml

# 설치
helm install network-collector ./helm -f values.yaml
```

### 업그레이드

```bash
helm upgrade network-collector ./helm \
  --set secrets.dbPassword=your-db-password \
  --set secrets.openstackAuthURL=http://keystone:5000/v3 \
  --set secrets.openstackUsername=admin \
  --set secrets.openstackPassword=your-openstack-password \
  --set secrets.openstackProjectID=your-project-id \
  --set secrets.jwtSecret=your-jwt-secret \
  --set secrets.authToken=your-auth-token
```

## 설정

### 필수 설정

다음 값들은 반드시 설정해야 합니다:

- `secrets.dbPassword`: MariaDB 비밀번호
- `secrets.openstackAuthURL`: OpenStack Keystone URL
- `secrets.openstackUsername`: OpenStack 사용자명
- `secrets.openstackPassword`: OpenStack 비밀번호
- `secrets.openstackProjectID`: OpenStack 프로젝트 ID
- `secrets.jwtSecret`: JWT 토큰 서명 키
- `secrets.authToken`: API 인증 토큰

### 선택적 설정

- `collector.replicas`: Collector 복제본 수 (기본값: 1)
- `api.replicas`: API 서비스 복제본 수 (기본값: 2)
- `frontend.replicas`: Frontend 복제본 수 (기본값: 2)
- `mariadb.storage.size`: MariaDB 스토리지 크기 (기본값: 20Gi)
- `ingress.enabled`: Ingress 활성화 (기본값: false)

## 값 설정 예시

```yaml
# values.yaml
collector:
  replicas: 1
  resources:
    requests:
      memory: "256Mi"
      cpu: "100m"
    limits:
      memory: "1Gi"
      cpu: "500m"

api:
  replicas: 3
  service:
    type: LoadBalancer

mariadb:
  storage:
    size: 50Gi
    storageClass: fast-ssd

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: network-collector.example.com
      paths:
        - path: /
          pathType: Prefix
        - path: /api
          pathType: Prefix
```

## 제거

```bash
helm uninstall network-collector
```

## 트러블슈팅

### Pod가 시작되지 않음

1. Secret 값이 올바르게 설정되었는지 확인:
   ```bash
   kubectl get secret network-collector-secrets -o yaml
   ```

2. ConfigMap 확인:
   ```bash
   kubectl get configmap network-collector-config -o yaml
   ```

3. Pod 이벤트 확인:
   ```bash
   kubectl describe pod <pod-name>
   ```

### 데이터베이스 연결 실패

1. MariaDB Pod 상태 확인:
   ```bash
   kubectl get pods -l app=mariadb
   ```

2. MariaDB 로그 확인:
   ```bash
   kubectl logs -l app=mariadb
   ```

### 이미지 Pull 실패

1. 이미지 레지스트리 확인
2. imagePullSecrets 설정:
   ```yaml
   global:
     imagePullSecrets:
       - name: my-registry-secret
   ```

## 참고

- 모든 Secret 값은 `--set` 플래그나 values.yaml 파일을 통해 설정해야 합니다.
- 프로덕션 환경에서는 Secret 관리를 위해 Kubernetes Secret 관리 도구(예: Sealed Secrets, External Secrets)를 사용하는 것이 좋습니다.

