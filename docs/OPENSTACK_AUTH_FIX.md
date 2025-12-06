# OpenStack 인증 설정 가이드

## 문제 상황

로그에서 다음과 같은 에러가 반복적으로 발생합니다:
```
Authentication failed
complete collection failure: all resources failed to collect
```

이는 OpenStack 인증 정보가 올바르게 설정되지 않아서 발생하는 문제입니다.

## 해결 방법

### 1. OpenStack 인증 정보 확인

먼저 실제 OpenStack 환경의 인증 정보를 확인하세요:

```bash
# OpenStack Keystone 엔드포인트 확인
openstack endpoint list | grep keystone

# 프로젝트 목록 확인
openstack project list

# 사용자 확인
openstack user list
```

### 2. Secret 업데이트

Secret 파일을 수정하거나 kubectl로 직접 업데이트하세요:

#### 방법 1: Secret 파일 수정 후 적용

`backend/deployments/k8s/secret.yaml` 파일을 수정:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: network-collector-secrets
  namespace: default
type: Opaque
stringData:
  # Database credentials
  DB_PASSWORD: "root"  # 실제 DB 비밀번호로 변경
  
  # OpenStack credentials
  OPENSTACK_AUTH_URL: "http://<keystone-ip>:5000/v3"  # 실제 Keystone URL
  OPENSTACK_USERNAME: "admin"  # 실제 OpenStack 사용자명
  OPENSTACK_PASSWORD: "실제비밀번호"  # 실제 OpenStack 비밀번호
  OPENSTACK_PROJECT_ID: "프로젝트UUID"  # 실제 프로젝트 ID (UUID)
  
  # Authentication
  JWT_SECRET: "실제JWT시크릿"
  AUTH_TOKEN: "tRymL0UgmsMoFEa0QWz6kiJ4WZI2FdEZalm0Pstv1sQ="  # 이미 설정된 값
```

그 다음 적용:
```bash
kubectl apply -f backend/deployments/k8s/secret.yaml
```

#### 방법 2: kubectl로 직접 업데이트

```bash
# OpenStack 인증 정보 업데이트
kubectl create secret generic network-collector-secrets \
  --from-literal=OPENSTACK_AUTH_URL='http://<keystone-ip>:5000/v3' \
  --from-literal=OPENSTACK_USERNAME='admin' \
  --from-literal=OPENSTACK_PASSWORD='실제비밀번호' \
  --from-literal=OPENSTACK_PROJECT_ID='프로젝트UUID' \
  --from-literal=DB_PASSWORD='root' \
  --from-literal=JWT_SECRET='실제JWT시크릿' \
  --from-literal=AUTH_TOKEN='tRymL0UgmsMoFEa0QWz6kiJ4WZI2FdEZalm0Pstv1sQ=' \
  --dry-run=client -o yaml | kubectl apply -f -
```

### 3. Pod 재시작

Secret을 업데이트한 후 Pod를 재시작하여 새 환경 변수를 적용:

```bash
# Collector Pod 재시작
kubectl rollout restart deployment/network-collector

# API Pod 재시작 (선택사항)
kubectl rollout restart deployment/network-collector-api

# 재시작 완료 대기
kubectl rollout status deployment/network-collector
```

### 4. 확인

```bash
# Collector 로그 확인
kubectl logs -f deployment/network-collector | grep -i "authentication\|error\|success"

# 환경 변수 확인
kubectl exec -it $(kubectl get pods -l app=network-collector -o jsonpath='{.items[0].metadata.name}') -- env | grep OPENSTACK
```

## OpenStack 인증 정보 찾는 방법

### Keystone URL 확인

```bash
# OpenStack CLI 사용
openstack endpoint list | grep keystone

# 또는 환경 변수 확인
echo $OS_AUTH_URL
```

### 프로젝트 ID 확인

```bash
# 프로젝트 목록
openstack project list

# 현재 프로젝트 ID
openstack project show $(openstack project list -f value -c Name | head -1) -f value -c id
```

### 사용자명 및 비밀번호

- OpenStack 관리자에게 문의하거나
- 기존 OpenStack 설정 파일(`openrc` 또는 `clouds.yaml`) 확인

## 테스트

인증 정보를 업데이트한 후:

```bash
# Collector 로그에서 성공 메시지 확인
kubectl logs deployment/network-collector --tail=50 | grep -i "collection\|success"

# 데이터베이스에 데이터가 수집되었는지 확인
kubectl exec -it mariadb-0 -- mysql -u openstack_monitor -proot openstack_monitor -e "SELECT COUNT(*) FROM instances;"
```

## 주의사항

1. **AUTH_TOKEN은 이미 설정되어 있음**: `tRymL0UgmsMoFEa0QWz6kiJ4WZI2FdEZalm0Pstv1sQ=` - 이 값은 유지하세요
2. **DB_PASSWORD**: 현재 `root`로 설정되어 있음
3. **OPENSTACK_AUTH_URL**: `http://keystone:5000/v3`는 클러스터 내부 DNS 이름입니다. 실제 IP나 외부 접근 가능한 URL로 변경해야 할 수 있습니다.

