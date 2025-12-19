# OpenTelemetry Collector: Building and Running as Binary and Image

## Section 1: Building and Running as a Binary

### Prerequisites
- Go >= 1.20

### 1. Build the Collector Binary
Run the following command from the root of the repository to build the collector binary:

```bash
make otelcorecol
```
The binary will be generated in the `bin` directory.

### 2. Prepare a Sample Collector Config
Create a config file (e.g., `collector-config.yaml`) with the following content:

```yaml
receivers:
  otlp/opsramp_logs_rx:
    protocols:
      grpc:
        endpoint: 127.0.0.1:9084
processors:
  filter:
    error_mode: ignore
    logs:
      log_record:
        - 'not(IsMatch(body, "^(?P<time>[^\\s]+) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*)?(?P<log>.*)$"))'
        - 'IsMatch(attributes["level"], "Debug")'
  batch/opsramp_logs:
    timeout: 2s
    send_batch_size: 1000
    send_batch_max_size: 5000
exporters:
  debug:
    verbosity: detailed
  opsrampotlp/opsramp_logs:
    timeout: 5s
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      randomization_factor: 0.5
      multiplier: 1.5
      max_interval: 30s
      max_elapsed_time: 5m
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 1000
    endpoint: <opsramp_otlp_endpoint>
    tls:
      insecure: false
      insecure_skip_verify: true
    compression: "gzip"
    headers:
      "tenantId": <your_tenant_id>
    read_buffer_size: 1024
    write_buffer_size: 1024
    security:
      oauth_service_url: <auth_service_url>
      client_id: <agent-api-key>
      client_secret: <agent-api-secret>
      otel_exporter_setting:
        otel_exporter_read_buffer_size: 1024
        otel_exporter_write_buffer_size: 1024
        grpc_max_call_send_msg_size: 1024000
        grpc_max_call_recv_msg_size: 1024000
service:
  pipelines:
    logs:
      receivers: [otlp/opsramp_logs_rx]
      processors: [filter, batch/opsramp_logs]
      exporters: [debug, opsrampotlp/opsramp_logs]
```


- Update `<opsramp_otlp_endpoint>`, `<auth_service_url>`, `<agent-api-key>`, `<agent-api-secret>`, and `<your_tenant_id>` with actual values.
- Replace "^(?P<time>[^\\s]+) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*)?(?P<log>.*)$" with your actual log format regex.
- Save this file as `collector-config.yaml`.

### 3. Run the Collector Binary

```bash
./bin/otelcorecol --config collector-config.yaml
```

---

## Section 2: Building and Running as a Docker Image (Kubernetes)

### Prerequisites
- Docker
- kubectl (for Kubernetes deployment)

### 1. Build the Docker Image

```bash
docker build -t <your-dockerhub-username>/otelcol:latest .
```

Push the image to your registry:

```bash
docker push <your-dockerhub-username>/otelcol:latest
```

### 2. Deploy on Kubernetes

Apply the following resources (update placeholders as needed):

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    app: otelcol-apiserver
  name: otelcol
  namespace: default
data:
  config.yaml: |
    receivers:
      otlp/opsramp_logs_rx:
        protocols:
          grpc:
            endpoint: 127.0.0.1:9084
    processors:
      filter:
        error_mode: ignore
        logs:
          log_record:
            - 'not(IsMatch(body, "^(?P<time>[^\\s]+) (?P<stream>stdout|stderr) (?P<logtag>[^ ]*)?(?P<log>.*)$"))'
            - 'IsMatch(attributes["level"], "Debug")'
      batch/opsramp_logs:
        timeout: 2s
        send_batch_size: 1000
        send_batch_max_size: 5000
    exporters:
      debug:
        verbosity: detailed
      opsrampotlp/opsramp_logs:
        timeout: 5s
        retry_on_failure:
          enabled: true
          initial_interval: 5s
          randomization_factor: 0.5
          multiplier: 1.5
          max_interval: 30s
          max_elapsed_time: 5m
        sending_queue:
          enabled: true
          num_consumers: 10
          queue_size: 1000
        endpoint: <opsramp_otlp_endpoint>
        tls:
          insecure: false
          insecure_skip_verify: true
        compression: "gzip"
        headers:
          "tenantId": <your_tenant_id>
        read_buffer_size: 1024
        write_buffer_size: 1024
        security:
          oauth_service_url: <auth_service_url>
          client_id: <agent-api-key>
          client_secret: <agent-api-secret>
          otel_exporter_setting:
            otel_exporter_read_buffer_size: 1024
            otel_exporter_write_buffer_size: 1024
            grpc_max_call_send_msg_size: 1024000
            grpc_max_call_recv_msg_size: 1024000
    service:
      pipelines:
        logs:
          receivers: [otlp/opsramp_logs_rx]
          processors: [filter, batch/opsramp_logs]
          exporters: [debug, opsrampotlp/opsramp_logs]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app: otelcol
  name: otelcol
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    app: otelcol
  name: otelcol
rules:
  - apiGroups:
      - "*"
    resources:
      - "*"
    verbs:
      - "*"
  - nonResourceURLs:
      - "*"
    verbs:
      - "*"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otelcol
  labels:
    app: otelcol
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otelcol
subjects:
  - kind: ServiceAccount
    name: otelcol
    namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otelcol
  namespace: default
  labels:
    app: otelcol
spec:
  replicas: 1
  selector:
    matchLabels:
      app: otelcol
  template:
    metadata:
      labels:
        app: otelcol
    spec:
      securityContext:
        runAsUser: 0
      serviceAccountName: otelcol
      serviceAccount: otelcol
      containers:
        - name: otelcol
          image: <your-dockerhub-username>/otelcol:latest
          imagePullPolicy: Always
          command:
            - "./otelcollector"
          args:
            - "--config"
            - "/etc/config/config.yaml"
            - "--feature-gates"
            - "+filelog.mtimeSortType"
          env:
            - name: K8S_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: config
              mountPath: /etc/config
            - name: varlog
              mountPath: /host/var/log
              readOnly: true
      volumes:
        - name: config
          configMap:
            name: otelcol
        - name: varlog
          hostPath:
            path: /var/log
```

Apply with:
```bash
kubectl apply -f deployment.yaml
```

---

- Update placeholders (`<your-dockerhub-username>`, etc.) with your actual values.
- For troubleshooting, check pod logs with `kubectl logs <pod-name>` if the pod does not start.
- Refer to [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/) for more details.
