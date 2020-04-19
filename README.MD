controller部署文档：
以下配置假设namespace为apisix

1.使用docker打包推送镜像；
```
docker build -t ???/???/ingress-hw:??? .
docker push ???/???/ingress-hw:???
```

2.配置环境变量；
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: apisix-config
  namespace: apisix
data:
  ETCD_SERVER_INTERNAL: '["http://127.0.0.1:2379","http://127.0.0.1:2380","http://127.0.0.1:2381"]'
  SYSLOG_HOST: 127.0.0.1
  APISIX_BASE_URL: "http://apisix.apisix.svc.cluster.local:9180/apisix/admin"
```

3.配置RBAC;
```
apiVersion: v1
kind: ServiceAccount
metadata:
  name: apisix-view-serviceaccount
  namespace: apisix
```

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: apisix-view-clusterrolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: apisix-view-clusterrole
subjects:
- kind: ServiceAccount
  name: apisix-view-serviceaccount
  namespace: apisix
```

```
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: apisix-view-clusterrole
rules:
- apiGroups:
  - ""
  resources:
  - configmaps
  - endpoints
  - persistentvolumeclaims
  - pods
  - replicationcontrollers
  - replicationcontrollers/scale
  - serviceaccounts
  - services
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - bindings
  - events
  - limitranges
  - namespaces/status
  - pods/log
  - pods/status
  - replicationcontrollers/status
  - resourcequotas
  - resourcequotas/status
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - apps
  resources:
  - controllerrevisions
  - daemonsets
  - deployments
  - deployments/scale
  - replicasets
  - replicasets/scale
  - statefulsets
  - statefulsets/scale
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - autoscaling
  resources:
  - horizontalpodautoscalers
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - batch
  resources:
  - cronjobs
  - jobs
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - extensions
  resources:
  - daemonsets
  - deployments
  - deployments/scale
  - ingresses
  - networkpolicies
  - replicasets
  - replicasets/scale
  - replicationcontrollers/scale
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - policy
  resources:
  - poddisruptionbudgets
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - networking.k8s.io
  resources:
  - ingresses
  - networkpolicies
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - metrics.k8s.io
  resources:
  - pods
  verbs:
  - get
  - list
  - watch
```

4.修改deployment；
```
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: ingress-controller
  namespace: apisix
spec:
  minReadySeconds: 5
  replicas: 1
  selector:
    matchLabels:
      app: apisix
      tier: backend
  strategy:
    rollingUpdate:
      maxSurge: 50%
      maxUnavailable: 1
    type: RollingUpdate
  template:
    metadata:
      annotations: {}
      labels:
        app: ingress-controller
    spec:
      serviceAccount: apisix-view-serviceaccount
      containers:
      - envFrom:
        - configMapRef:
            name: apisix-config
        image: ???/???/ingress-controller:???
        imagePullPolicy: IfNotPresent
        name: ingress-controller
        ports:
        - containerPort: 8080
          hostPort: 8080
      terminationGracePeriodSeconds: 60
```

5.启动；
通过kubectl执行deployment
