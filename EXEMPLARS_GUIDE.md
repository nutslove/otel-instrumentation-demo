# OpenTelemetry Exemplars 実装ガイド

## 目次
1. [Exemplarsとは](#exemplarsとは)
2. [前提条件](#前提条件)
3. [各言語のExemplarsサポート状況](#各言語のexemplarsサポート状況)
4. [実装方法](#実装方法)
5. [トラブルシューティング](#トラブルシューティング)
6. [確認方法](#確認方法)

---

## Exemplarsとは

**Exemplars（例示値）** は、メトリクスデータポイントにトレース情報（trace_id/span_id）を紐付ける機能です。

### メリット
- メトリクスグラフから直接対応するトレースにジャンプできる
- 異常なメトリクス値の原因を素早く特定できる
- メトリクスとトレースの相関分析が容易になる

### 仕組み
```
Histogram Bucket: latency < 100ms: count=1000
  └─ Exemplar: {trace_id="abc123", span_id="def456", value=95ms, timestamp=...}
```

メトリクスの集計値（例：ヒストグラムのバケット）に対して、代表的なサンプルのトレース情報を保存します。

---

## 前提条件

### 必須コンポーネントとバージョン

| コンポーネント | 推奨バージョン | 理由 |
|--------------|--------------|------|
| OpenTelemetry Collector | **v0.136.0以降** | `enable_open_metrics: true`サポート |
| Prometheus | **v3.5.0以降** | Exemplar Storage機能の安定版 |
| Python SDK | **v1.28.0以降** | Exemplarsサポート追加 |
| Java SDK | **v1.33.0以降** | Micrometer Bridge対応 |
| Node.js SDK | **未サポート (2025年10月時点)** | 仕様はあるが実装なし |

---

## 各言語のExemplarsサポート状況

### ✅ Python (完全サポート)

**サポート開始**: SDK v1.28.0 (2024年11月)
**現在の最新版**: v1.37.0 (2025年9月)

#### 必要な設定

##### 1. requirements.txt
```python
fastapi==0.109.0
uvicorn==0.27.0
httpx==0.26.0
opentelemetry-distro  # 最新版を自動取得
opentelemetry-exporter-otlp-proto-grpc
opentelemetry-instrumentation-fastapi
opentelemetry-instrumentation-httpx
```

##### 2. 環境変数 (docker-compose.yml)
```yaml
python-service:
  environment:
    - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
    - OTEL_SERVICE_NAME=python-fastapi-service
    - OTEL_TRACES_EXPORTER=otlp
    - OTEL_METRICS_EXPORTER=otlp  # 重要: OTLPで送信
    - OTEL_LOGS_EXPORTER=otlp
    - OTEL_METRICS_EXEMPLAR_FILTER=always_on  # Exemplarを常に有効化
```

##### 3. 起動コマンド (Dockerfile)
```dockerfile
CMD ["opentelemetry-instrument", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
```

#### 動作確認
```bash
curl -H "Accept: application/openmetrics-text" http://localhost:8889/metrics | grep "python-fastapi" | grep "trace_id"
```

**出力例**:
```
otel_demo_http_client_duration_milliseconds_bucket{...,le="5.0"} 2 # {trace_id="10262aae290d1d197e20043c7dd15dfe",span_id="edaf6afd945a9e60"} 4.0 1.7596106280019937e+09
```

---

### ✅ Java (完全サポート)

**サポート方法**: Micrometer Tracing Bridge + OpenTelemetry Agent
**現在の推奨バージョン**: OTel Java Agent v1.33.0以降

#### 必要な設定

##### 1. pom.xml
```xml
<dependencies>
    <!-- Spring Boot Actuator -->
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-actuator</artifactId>
    </dependency>

    <!-- Micrometer Prometheus Registry -->
    <dependency>
        <groupId>io.micrometer</groupId>
        <artifactId>micrometer-registry-prometheus</artifactId>
    </dependency>

    <!-- Micrometer Tracing Bridge for OpenTelemetry -->
    <dependency>
        <groupId>io.micrometer</groupId>
        <artifactId>micrometer-tracing-bridge-otel</artifactId>
    </dependency>

    <!-- OpenTelemetry API -->
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-api</artifactId>
        <version>1.33.0</version>
    </dependency>

    <!-- OpenTelemetry Context -->
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-context</artifactId>
        <version>1.33.0</version>
    </dependency>
</dependencies>
```

##### 2. application.properties
```properties
# Actuator Endpoints
management.endpoints.web.exposure.include=health,prometheus
management.metrics.export.prometheus.enabled=true
management.metrics.distribution.percentiles-histogram.http.server.requests=true

# Exemplars設定
management.prometheus.metrics.export.step=10s
management.metrics.tags.application=${spring.application.name}

# Micrometer Tracing
management.tracing.enabled=true
management.tracing.sampling.probability=1.0
```

##### 3. 環境変数 (docker-compose.yml)
```yaml
java-service:
  environment:
    - OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
    - OTEL_SERVICE_NAME=java-spring-boot-service
    - OTEL_TRACES_EXPORTER=otlp
    - OTEL_METRICS_EXPORTER=otlp
    - OTEL_LOGS_EXPORTER=otlp
    - OTEL_METRICS_EXEMPLAR_FILTER=ALWAYS_ON
```

##### 4. Dockerfile
```dockerfile
FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY target/*.jar app.jar

# OpenTelemetry Java Agentダウンロード
ADD https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/download/v1.33.0/opentelemetry-javaagent.jar /app/opentelemetry-javaagent.jar

# Java Agent経由で起動
ENTRYPOINT ["java", "-javaagent:/app/opentelemetry-javaagent.jar", "-jar", "app.jar"]
```

#### Prometheusスクレープ設定
```yaml
scrape_configs:
  - job_name: 'java-service'
    scrape_interval: 5s
    metrics_path: '/actuator/prometheus'
    static_configs:
      - targets: ['java-service:8081']
```

**重要**: JavaはSpring Boot Actuatorの`/actuator/prometheus`エンドポイントから直接Prometheusがスクレープする方式です。OTel Collector経由ではありません。

---

### ❌ Node.js (未サポート)

**現状**: OpenTelemetry JavaScript SDK v0.52.0時点でExemplarsは**未実装**

#### サポート状況の詳細

1. **GitHub Issue**: [#2594](https://github.com/open-telemetry/opentelemetry-js/issues/2594)
   - Exemplarsの仕様は存在するが実装なし
   - 現在誰も作業していない
   - 仕様が安定版になったら関心が高まると予想

2. **メトリクスエクスポート自体の問題**
   - 自動計装（`@opentelemetry/auto-instrumentations-node`）では、環境変数だけではメトリクスエクスポート設定ができない
   - プログラマティックな初期化（`NodeSDK`）が必要

#### 将来的な対応予定

Node.jsでExemplarsを利用するには:
1. OpenTelemetry JavaScript SDKでExemplarsが実装されるのを待つ
2. または、手動でトレースコンテキストをメトリクスに紐付ける独自実装

**推奨**: 現時点ではNode.jsでのExemplars利用は見送り

---

## 実装方法

### 1. Prometheusの設定

#### prometheus.yml
```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# Exemplar Storageを有効化
storage:
  exemplars:
    max_exemplars: 100000

scrape_configs:
  - job_name: 'otel-collector'
    scrape_interval: 10s
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'otel_demo_.*'
        action: keep
    static_configs:
      - targets: ['otel-collector:8889']
        labels:
          service: 'otel-collector'

  - job_name: 'java-service'
    scrape_interval: 5s
    metrics_path: '/actuator/prometheus'
    static_configs:
      - targets: ['java-service:8081']
        labels:
          service: 'java-spring-boot'
```

#### docker-compose.yml
```yaml
prometheus:
  image: prom/prometheus:v3.5.0  # v3.5.0以降を使用
  command:
    - '--config.file=/etc/prometheus/prometheus.yml'
    - '--storage.tsdb.path=/prometheus'
    - '--enable-feature=exemplar-storage'  # 必須フラグ
  volumes:
    - ./prometheus.yml:/etc/prometheus/prometheus.yml
    - prometheus-data:/prometheus
  ports:
    - "9090:9090"
```

---

### 2. OpenTelemetry Collectorの設定

#### otel-collector-config.yaml
```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 10s
    send_batch_size: 1024

  memory_limiter:
    check_interval: 1s
    limit_mib: 512

  resource:
    attributes:
      - key: environment
        value: demo
        action: insert

exporters:
  # Tempo for traces
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true

  # Prometheus for metrics
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: otel_demo
    enable_open_metrics: true  # Exemplarsサポートに必須
    resource_to_telemetry_conversion:
      enabled: true

  # Debug for debugging
  debug:
    verbosity: detailed  # または normal
    sampling_initial: 5
    sampling_thereafter: 200

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [otlp/tempo, debug]

    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [prometheus, debug]

    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch, resource]
      exporters: [debug]
```

#### docker-compose.yml
```yaml
otel-collector:
  image: otel/opentelemetry-collector-contrib:0.136.0  # 最新版を使用
  command: ["--config=/etc/otel-collector-config.yaml"]
  volumes:
    - ./collector/otel-collector-config.yaml:/etc/otel-collector-config.yaml
  ports:
    - "4317:4317"   # OTLP gRPC receiver
    - "4318:4318"   # OTLP HTTP receiver
    - "8889:8889"   # Prometheus metrics exporter
```

**重要な変更点** (v0.91.0 → v0.136.0):
- `logging` exporter → `debug` exporterに変更（`logging`は非推奨）
- `enable_open_metrics: true`が必須

---

### 3. Grafanaの設定

#### datasources.yaml
```yaml
apiVersion: 1

datasources:
  - name: Tempo
    type: tempo
    access: proxy
    url: http://tempo:3200
    uid: tempo
    jsonData:
      tracesToLogsV2:
        datasourceUid: loki
      tracesToMetrics:
        datasourceUid: prometheus
      nodeGraph:
        enabled: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    uid: loki
    jsonData:
      derivedFields:
        - datasourceUid: tempo
          matcherRegex: "trace_id[=:]\\s*(\\w+)"
          name: TraceID
          url: "$${__value.raw}"

  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    uid: prometheus
    jsonData:
      exemplarTraceIdDestinations:
        - datasourceUid: tempo
          name: trace_id
      httpMethod: POST
```

**重要**: `exemplarTraceIdDestinations`でPrometheusのExemplarをTempoトレースに紐付けます。

---

## トラブルシューティング

### 1. Exemplarsが表示されない

#### チェックリスト

1. **Prometheusのフラグ確認**
   ```bash
   docker logs prometheus 2>&1 | grep exemplar
   ```
   `--enable-feature=exemplar-storage`が有効か確認

2. **OTel Collectorの設定確認**
   ```yaml
   # enable_open_metrics: trueが設定されているか
   prometheus:
     enable_open_metrics: true
   ```

3. **SDKバージョン確認**
   - Python: v1.28.0以降
   - Java: OTel Agent v1.33.0 + Micrometer Bridge
   - Node.js: 未サポート

4. **環境変数確認**
   ```yaml
   - OTEL_METRICS_EXPORTER=otlp  # prometheusではなくotlp
   - OTEL_METRICS_EXEMPLAR_FILTER=always_on  # または ALWAYS_ON
   ```

#### Python特有の問題

**症状**: Python SDK v1.22.0ではExemplarsが生成されない

**原因**: v1.28.0からExemplarsサポートが追加された

**解決方法**:
```python
# requirements.txtを修正
opentelemetry-distro  # バージョン指定なしで最新版を取得
opentelemetry-exporter-otlp-proto-grpc
```

依存関係の競合を避けるため、バージョンを固定しない方が安全です。

#### Java特有の問題

**症状**: Javaのメトリクスが `/actuator/prometheus` で見えるが、Exemplarsがない

**原因**: Micrometer Tracing Bridgeが設定されていない

**解決方法**:
```xml
<dependency>
    <groupId>io.micrometer</groupId>
    <artifactId>micrometer-tracing-bridge-otel</artifactId>
</dependency>
```

また、`application.properties`で以下を確認:
```properties
management.tracing.enabled=true
management.tracing.sampling.probability=1.0
```

---

### 2. OTel Collectorのエラー

#### "unknown exporter: logging"

**症状**:
```
Error: unknown exporter "logging" for data type "metrics"
```

**原因**: Collector v0.136.0で`logging` exporterが非推奨になり、`debug` exporterに変更された

**解決方法**:
```yaml
exporters:
  debug:  # loggingではなくdebug
    verbosity: detailed
```

---

### 3. メトリクスが送信されない (Node.js)

**症状**: Node.jsからメトリクスが全く送信されない

**原因**: 自動計装では環境変数だけではメトリクス設定ができない

**解決方法** (2つのアプローチ):

#### アプローチ1: プログラマティック初期化
```javascript
// tracing.js
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { PeriodicExportingMetricReader } = require('@opentelemetry/sdk-metrics');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-grpc');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

const sdk = new NodeSDK({
  metricReader: new PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter({
      url: 'http://otel-collector:4317',
    }),
    exportIntervalMillis: 10000,
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();
```

```javascript
// index.js
require('./tracing.js');
const express = require('express');
// ... rest of your app
```

#### アプローチ2: Exemplarsは諦める
- Node.jsではトレースのみ使用
- メトリクスはPythonやJavaサービスで取得

**推奨**: 現時点ではアプローチ2（トレースのみ）

---

## 確認方法

### 1. コマンドラインで確認

#### Pythonのメトリクスを確認
```bash
curl -H "Accept: application/openmetrics-text" http://localhost:8889/metrics | grep "python-fastapi" | grep "trace_id"
```

**期待される出力**:
```
otel_demo_http_client_duration_milliseconds_bucket{...} 2 # {trace_id="abc123...",span_id="def456..."} 4.0 1234567890
```

#### Javaのメトリクスを確認
```bash
curl http://localhost:8081/actuator/prometheus | grep "trace_id"
```

#### OTel Collectorのメトリクスエンドポイント確認
```bash
curl -H "Accept: application/openmetrics-text" http://localhost:8889/metrics | grep "trace_id" | head -20
```

---

### 2. Grafanaで確認

1. **Prometheusダッシュボードにアクセス**
   - URL: http://localhost:9090

2. **メトリクスを検索**
   ```promql
   otel_demo_http_server_duration_milliseconds_bucket
   ```

3. **Exemplarsタブをクリック**
   - メトリクスデータポイントに紐付いたトレース情報が表示される

4. **Grafanaダッシュボードにアクセス**
   - URL: http://localhost:3000

5. **Exploreでメトリクスを表示**
   - Data source: Prometheus
   - Query: `rate(otel_demo_http_server_duration_milliseconds_bucket[1m])`
   - グラフ上のデータポイントをクリック → "Exemplar"が表示される
   - Exemplarをクリック → Tempoの対応するトレースに自動ジャンプ

---

### 3. テストリクエストを送信

```bash
# 複数リクエストを送信してExemplarを生成
for i in {1..5}; do
  curl -X POST http://localhost:8000/orders \
    -H "Content-Type: application/json" \
    -d '{"user_id": '$i', "product_name": "Laptop", "quantity": 1}'
  sleep 1
done

# 15秒待ってからExemplarを確認（メトリクスのエクスポート間隔を考慮）
sleep 15

# Exemplarsを確認
curl -H "Accept: application/openmetrics-text" http://localhost:8889/metrics | grep "trace_id" | head -20
```

---

## まとめ

### 動作状況

| 言語 | Exemplarsサポート | 実装方法 | 備考 |
|------|-----------------|---------|------|
| Python | ✅ 完全対応 | SDK v1.37.0 + OTLP | 環境変数のみで設定可能 |
| Java | ✅ 完全対応 | Micrometer Bridge + OTel Agent | Spring Boot Actuator経由 |
| Node.js | ❌ 未対応 | - | 2025年10月時点で未実装 |
| Go | ⚠️ 手動実装のみ | 手動でExemplar追加 | 自動計装なし |

### ベストプラクティス

1. **最新バージョンを使用**
   - OTel Collector: v0.136.0以降
   - Prometheus: v3.5.0以降
   - Python SDK: v1.37.0以降

2. **環境変数設定**
   ```yaml
   - OTEL_METRICS_EXPORTER=otlp
   - OTEL_METRICS_EXEMPLAR_FILTER=always_on
   ```

3. **Prometheus設定**
   ```yaml
   storage:
     exemplars:
       max_exemplars: 100000
   ```
   ```bash
   --enable-feature=exemplar-storage
   ```

4. **OTel Collector設定**
   ```yaml
   prometheus:
     enable_open_metrics: true
   ```

5. **Grafana Datasource設定**
   ```yaml
   exemplarTraceIdDestinations:
     - datasourceUid: tempo
       name: trace_id
   ```

### 参考リンク

- [OpenTelemetry Specification - Exemplars](https://opentelemetry.io/docs/specs/otel/metrics/sdk/#exemplar)
- [Prometheus Exemplars Documentation](https://prometheus.io/docs/prometheus/latest/feature_flags/#exemplars-storage)
- [OpenTelemetry Python SDK Releases](https://github.com/open-telemetry/opentelemetry-python/releases)
- [OpenTelemetry Java Agent Releases](https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases)
- [OpenTelemetry JavaScript Issue #2594 - Exemplars](https://github.com/open-telemetry/opentelemetry-js/issues/2594)
