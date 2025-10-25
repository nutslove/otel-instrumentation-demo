# 🔭 OpenTelemetry マイクロサービス デモ

**Python (FastAPI)**、**Node.js (Express)**、**Go (Gin)**、**Java (Spring Boot)** で構築されたマイクロサービスを使用した包括的なOpenTelemetryデモンストレーション。分散トレーシング、メトリクス、ログの相関機能を完全実装。

## 🎯 機能

- **多言語マイクロサービス**: Python、Node.js、Go、Java
- **複数のインストルメンテーション方式**:
  - 手動計装（OpenTelemetry SDK）
  - Envoyサイドカーによる自動トレーシング
  - Envoyサイドカー + トレースヘッダー手動伝播
  - eBPFベースの自動計装（実験的）
- **完全なオブザーバビリティスタック**: トレース、メトリクス、ログ
- **トレースヘッダー伝播デモ**: Envoyサイドカー使用時のヘッダー伝播の問題と解決策を実演
- **分散トレーシング**: すべてのサービス間でリクエストを追跡
- **OTLP Collector**: 集中型テレメトリー収集
- **Grafana Tempo**: 分散トレーシングバックエンド
- **Grafana Loki**: ログアグリゲーション
- **Prometheus**: メトリクスの収集とクエリ
- **Grafana**: 統合可視化ダッシュボード
- **SQLiteデータベース**: 各サービスごとに独立したデータベース
- **Web UI**: ワークフローをトリガーするインタラクティブダッシュボード
- **Docker Compose**: 複数の構成をサポート（Linux/Mac、手動計装/Envoy/eBPF）

## 🏗️ アーキテクチャ

### 基本アーキテクチャ（手動計装版）

```
Web UI (Nginx)
    ↓
Python Service (FastAPI) - 注文管理
    ↓
Node.js Service (Express) - 在庫管理
    ↓
Go Service (Gin) - 価格計算
    ↓
Java Service (Spring Boot) - 通知
    ↓
OpenTelemetry Collector
    ↓
├── Grafana Tempo (トレース)
├── Prometheus (メトリクス)
└── Grafana Loki (ログ)
    ↓
Grafana (統合可視化)
```

### Envoyサイドカーアーキテクチャ（ヘッダー伝播なし版）

```
Web UI (Nginx)
    ↓
Python Service (FastAPI) - 注文管理
    ↓
Node.js Service (Express) - 在庫管理
    ↓ (ポート10000経由)
┌─────────────────────────────┐
│ Envoy Sidecar (ingress)      │ ← ingressトレース生成
│    ポート10000               │
│         ↓                    │
│ Go Service (ポート8080)      │ ← 手動計装なし
│         ↓                    │    ヘッダー伝播なし ❌
│ Envoy Sidecar (egress)       │
│    ポート14318               │
└─────────────────────────────┘
    ↓ (トレースが途切れる)
Java Service (Spring Boot) - 通知
    ↓
OpenTelemetry Collector
    ↓
├── Grafana Tempo (トレース)
├── Prometheus (メトリクス)
└── Grafana Loki (ログ)
    ↓
Grafana (統合可視化)
```

### Envoyサイドカーアーキテクチャ（ヘッダー伝播あり版）

```
Web UI (Nginx)
    ↓
Python Service (FastAPI) - 注文管理
    ↓
Node.js Service (Express) - 在庫管理
    ↓ (ポート10000経由)
┌─────────────────────────────┐
│ Envoy Sidecar (ingress)      │ ← ingressトレース生成
│    ポート10000               │
│         ↓                    │
│ Go Service (ポート8080)      │ ← 手動計装なし
│         ↓                    │    ヘッダー手動伝播 ✅
│ Envoy Sidecar (egress)       │
│    ポート14318               │
└─────────────────────────────┘
    ↓ (トレースが繋がる)
Java Service (Spring Boot) - 通知
    ↓
OpenTelemetry Collector
    ↓
├── Grafana Tempo (トレース)
├── Prometheus (メトリクス)
└── Grafana Loki (ログ)
    ↓
Grafana (統合可視化)
```

## 📊 サービス概要

| サービス | 言語 | フレームワーク | ポート | データベース | 用途 |
|---------|----------|-----------|------|----------|---------|
| Python | Python 3.11 | FastAPI | 8000 | orders.db | 注文管理 |
| Node.js | Node.js 18 | Express | 3001 | inventory.db | 在庫追跡 |
| Go | Go 1.23 | Gin | 8080 | pricing.db | 価格計算 |
| Java | Java 17 | Spring Boot | 8081 | notifications.db | 通知 |
| Envoy Proxy | - | Envoy v1.36 | 10000 (ingress)<br>9901 (admin) | - | サイドカープロキシ<br>（Envoy版のみ） |
| Web UI | - | Nginx | 80 | - | ユーザーインターフェース |
| OTLP Collector | - | - | 4317/4318 | - | テレメトリー収集 |
| Grafana Tempo | - | - | 3200 | - | トレースバックエンド |
| Grafana Loki | - | - | 3100 | - | ログアグリゲーション |
| Prometheus | - | - | 9090 | - | メトリクスバックエンド |
| Grafana | - | - | 3000 | - | 統合可視化 |

## 🚀 クイックスタート

### 前提条件

- Docker
- Docker Compose (または docker compose)

### デモの起動

プロジェクトには複数のdocker-composeファイルが用意されています：

#### 1. 基本版（手動計装）- Linux

```bash
# 全サービスでOpenTelemetry SDKを使用した手動計装
docker compose up -d
# または
docker compose -f docker-compose.yml up -d
```

#### 2. 基本版（手動計装）- Mac

```bash
# Mac環境用（Java serviceのビルドが異なる）
docker compose -f docker-compose-mac.yml up -d
```

#### 3. Envoyサイドカー版 - Linux

```bash
# Go serviceでEnvoyがトレーシングを担当（手動計装不要）
docker compose -f docker-compose-envoy.yml up -d
```

#### 4. Envoyサイドカー版 - Mac

```bash
# Mac環境用のEnvoy版
docker compose -f docker-compose-envoy-mac.yml up -d
```

#### 5. eBPF自動計装版 - Linux

```bash
# eBPFベースの自動計装（実験的）
docker compose -f docker-compose-ebpf.yml up -d
```

#### 6. eBPF自動計装版 - Mac

```bash
# Mac環境用のeBPF版
docker compose -f docker-compose-ebpf-mac.yml up -d
```

#### 7. Envoyサイドカー版（トレースヘッダー伝播あり）- Linux

```bash
# Go serviceでトレースヘッダーを手動伝播（トレースが完全に繋がる）
docker compose -f docker-compose-envoy-propagation.yml up -d
```

#### 8. Envoyサイドカー版（トレースヘッダー伝播あり）- Mac

```bash
# Mac環境用のEnvoy + トレースヘッダー伝播版
docker compose -f docker-compose-envoy-propagation-mac.yml up -d
```

**📝 Note**: `docker-compose-envoy.yml`と`docker-compose-envoy-propagation.yml`の違い:
- **envoy.yml**: Go serviceが他サービスへの通信時にトレースヘッダーを伝播しない → Go → Javaでトレースが途切れる（問題を示すバージョン）
- **envoy-propagation.yml**: Go serviceがトレースヘッダーを手動で伝播 → 完全なトレースが繋がる（解決策バージョン）

### サービスへのアクセス

#### 基本版・eBPF版

- **Web UI**: http://localhost
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090
- **Python Service**: http://localhost:8000
- **Node.js Service**: http://localhost:3001
- **Go Service**: http://localhost:8080
- **Java Service**: http://localhost:8081
- **Grafana Tempo**: http://localhost:3200
- **Grafana Loki**: http://localhost:3100

#### Envoy版（追加ポート）

上記に加えて：
- **Envoy Proxy (Go Service経由)**: http://localhost:10000
- **Envoy Admin**: http://localhost:9901

### デモの停止

```bash
# 起動時に使用したファイルを指定
docker compose down
# または
docker compose -f docker-compose-envoy.yml down
```

### 停止とクリーンアップ（ボリューム含む）

```bash
docker compose down -v
# または
docker compose -f docker-compose-envoy.yml down -v
```

## 🎮 デモの使い方

### 1. Web UIダッシュボード

ブラウザで http://localhost を開いてインタラクティブダッシュボードにアクセスします。

#### 完全なワークフロー注文の作成

1. 注文フォームに入力:
   - User ID: 1001
   - Product: Laptop/Mouse/Keyboard
   - Quantity: 2

2. 「Create Order & Trigger Workflow」をクリック

3. 完全なフローがトリガーされます:
   - Pythonが注文を作成 → Node.jsが在庫をチェック → Goが価格を計算 → Javaが通知を送信

4. 結果から**トレースID**をコピー

5. Grafana (http://localhost:3000) を開き、トレースIDで検索して完全な分散トレースを表示

### 2. 分散トレーシングの確認

1. 注文作成後、Grafana (http://localhost:3000) を開く
2. 左メニューから「Explore」を選択
3. データソースで「Tempo」を選択
4. 「Search」タブで最近のトレースを検索
5. トレースをクリックして、完全な分散トレースを表示:
   - Python → Node.js → Go → Javaの完全なフロー
   - 各サービスでの処理時間
   - データベースクエリの詳細

### 3. オブザーバビリティの探索

#### Grafana（トレース、メトリクス、ログ）

1. http://localhost:3000 にアクセス
2. 左メニューから「Explore」を選択
3. データソースを選択:
   - **Tempo**: トレースの検索と可視化
   - **Loki**: ログの検索とフィルタリング
   - **Prometheus**: メトリクスのクエリ
4. トレース検索（Tempo）:
   - トレースIDで検索
   - サービス別にトレースを検索
   - 以下を確認:
     - サービス間のリクエストフロー
     - タイミング情報
     - データベース操作
     - エラー詳細

#### Prometheus（メトリクス）

1. http://localhost:9090 にアクセス
2. 以下のクエリを試す:
   ```promql
   # サービス別のリクエストレート
   rate(python_service_requests_total[1m])
   rate(nodejs_service_requests_total[1m])

   # リクエスト数
   python_service_requests_total
   nodejs_service_requests_total
   java_service_requests_total
   ```

#### ログ

各サービスのログを確認:

```bash
# Python サービスのログ
docker-compose logs -f python-service

# Node.js サービスのログ
docker-compose logs -f nodejs-service

# Go サービスのログ
docker-compose logs -f go-service

# Java サービスのログ
docker-compose logs -f java-service
```

すべてのログには相関のためのトレースIDが含まれています!

## 🔍 ワークフロー例

### 注文の作成（完全なトレース例）

```bash
# 1. Pythonサービス経由で注文を作成
curl -X POST http://localhost:8000/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id": 1001, "product_name": "Laptop", "quantity": 2}'

# レスポンスにはtrace_idが含まれます:
# {
#   "order_id": 1,
#   "status": "pending",
#   "inventory_check": {"available": true, ...},
#   "trace_id": "abc123..."
# }

# 2. trace_idを使用してGrafanaでトレースを検索
# 開く: http://localhost:3000 (Explore > Tempo)

# 3. 完全なフローが表示されます:
# - python-fastapi-service: POST /orders
#   - db_insert_order (SQLite)
#   - call_nodejs_inventory (HTTP)
#     - nodejs-express-service: POST /inventory/check
#       - db_select_inventory (SQLite)
```

## 📁 プロジェクト構成

```
otel-instrumentation-demo/
├── python-service/            # Python FastAPI サービス（手動計装）
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
├── nodejs-service/            # Node.js Express サービス（手動計装）
│   ├── index.js
│   ├── instrumentation.js
│   ├── package.json
│   └── Dockerfile
├── nodejs-service-envoy/      # Node.js（Envoy版 - ポート10000経由でGo接続）
│   ├── index.js               # ← go-service:10000に接続
│   ├── instrumentation.js
│   ├── package.json
│   └── Dockerfile
├── go-service/                # Go Gin サービス（手動計装）
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── go-service-ebpf/           # Go Gin サービス（手動計装なし、ヘッダー伝播なし）
│   ├── main.go                # ← OpenTelemetry SDKなし、トレースヘッダー伝播なし
│   ├── go.mod
│   └── Dockerfile
├── go-service-ebpf-propagation/ # Go Gin サービス（手動計装なし、ヘッダー伝播あり）
│   ├── main.go                # ← OpenTelemetry SDKなし、トレースヘッダーを手動伝播
│   ├── go.mod
│   └── Dockerfile
├── java-service/              # Java Spring Boot サービス（Linux用）
│   ├── src/
│   ├── pom.xml
│   └── Dockerfile
├── java-service-mac/          # Java Spring Boot サービス（Mac用）
│   ├── src/
│   ├── pom.xml
│   └── Dockerfile
├── envoy/                     # Envoy設定
│   └── envoy-go-service.yaml  # Go service用Envoyサイドカー設定
├── web-ui/                    # Webインターフェース
│   └── index.html
├── collector/                 # OTLP Collector設定
│   └── otel-collector-config.yaml
├── grafana/                   # Grafana設定
│   └── provisioning/
├── tempo/                     # Tempo設定
│   └── tempo.yaml
├── loki/                      # Loki設定
│   └── loki-config.yaml
├── data/                      # SQLiteデータベース（自動作成）
├── docker-compose.yml         # 基本版（Linux、手動計装）
├── docker-compose-mac.yml     # 基本版（Mac）
├── docker-compose-envoy.yml   # Envoyサイドカー版（Linux、ヘッダー伝播なし）
├── docker-compose-envoy-mac.yml # Envoyサイドカー版（Mac、ヘッダー伝播なし）
├── docker-compose-envoy-propagation.yml   # Envoyサイドカー版（Linux、ヘッダー伝播あり）
├── docker-compose-envoy-propagation-mac.yml # Envoyサイドカー版（Mac、ヘッダー伝播あり）
├── docker-compose-ebpf.yml    # eBPF自動計装版（Linux）
├── docker-compose-ebpf-mac.yml # eBPF自動計装版（Mac）
├── prometheus.yml             # Prometheus設定
└── README.md
```

## 🔧 設定

### OpenTelemetry Collector

Collectorは以下のように設定されています:
- ポート4317（gRPC）と4318（HTTP）でOTLPデータを受信
- トレースをGrafana Tempoにエクスポート
- ログをGrafana Lokiにエクスポート
- メトリクスをPrometheusにエクスポート
- すべてのテレメトリーデータをログ出力

設定ファイル: `collector/otel-collector-config.yaml`

### サービスのインストルメンテーション

すべてのサービスは以下で計装されています:
- **トレース**: 自動HTTPインストルメンテーション + カスタムスパン
- **メトリクス**: リクエスト追跡用のカスタムカウンター
- **ログ**: トレースコンテキスト付きの構造化ログ

## 🎓 学習ポイント

### 1. 分散トレーシング
- 単一のリクエストが複数のサービスをどのように流れるかを確認
- レイテンシーのボトルネックを理解
- どのサービスがエラーを引き起こしているかを特定

### 2. トレースID相関
- すべてのサービス操作に同じtrace_idがタグ付けされる
- trace_idを使用してログを検索し、関連エントリを見つける
- trace_idを追跡してサービス間の問題をデバッグ

### 3. 異なるインストルメンテーション方式の比較
- **手動計装**: OpenTelemetry SDKを使用してコード内で明示的にトレースを生成
- **Envoyサイドカー**: アプリケーションコード変更不要でプロキシレイヤーでトレース生成
- **eBPF**: カーネルレベルでの自動計装（実験的）

### 4. Envoyサイドカーパターン
- アプリケーションからトレーシングロジックを分離
- サービスメッシュアーキテクチャの基礎
- `network_mode: "service:xxx"`でコンテナ間のネットワーク名前空間を共有
- Envoyが透過的にトレースヘッダーを伝播（W3C Trace Context）

### 5. トレースヘッダー伝播の重要性
**問題**: Envoyがingressで受け取ったトレースコンテキストは、アプリケーションコードが他サービスへHTTPリクエストを送る際に自動的に伝播されない

**原因**:
- Envoyはingressでトレースヘッダーをアプリケーションに渡す
- しかし、手動計装なしのアプリケーションは、egressリクエストにヘッダーをコピーしない
- 結果: Go → Javaの通信でトレースが途切れる

**解決策**:
1. **完全な手動計装** (OpenTelemetry SDK使用) - 自動的にヘッダー伝播
2. **最小限のヘッダー伝播コード** - OpenTelemetry SDKなしで、HTTPヘッダーを手動でコピー
   ```go
   // 受信リクエストからヘッダー抽出
   traceparent := c.GetHeader("traceparent")

   // 送信リクエストにヘッダー設定
   httpReq.Header.Set("traceparent", traceparent)
   ```

**デモで確認**:
- `docker-compose-envoy.yml`: トレースが途切れる（問題）
- `docker-compose-envoy-propagation.yml`: トレースが繋がる（解決）

### 6. メトリクス収集
- 各サービスがカスタムメトリクスを送信
- Prometheusがメトリクスをスクレイプして保存
- メトリクスをクエリしてサービスの健全性を把握

### 7. データベース操作
- 各サービスは独自のSQLiteデータベースを持つ
- データベース操作は子スパンとしてトレース
- 正確なSQLクエリとタイミングを確認

### 8. 多言語サポート
- OpenTelemetryは言語間で一貫して動作
- 同じ概念（トレース、メトリクス、ログ）がどこでも適用される
- OTLPが標準プロトコルを提供

## 🐛 トラブルシューティング

### サービスが起動しない

```bash
# ログを確認
docker compose logs

# 特定のサービスを再起動
docker compose restart python-service
```

### Grafanaにトレースが表示されない

1. OTLP collectorが実行中か確認: `docker compose ps otel-collector`
2. collectorのログを確認: `docker compose logs otel-collector`
3. Tempoのログを確認: `docker compose logs tempo`
4. サービスがcollectorに到達できるか確認: `docker compose logs python-service | grep collector`

### Envoy版でGo Serviceのトレースが見えない

```bash
# 1. Envoyが起動しているか確認
docker compose ps envoy-go-service

# 2. Envoyのログを確認
docker compose logs envoy-go-service

# 3. Envoyの統計情報を確認（トレース送信状況）
curl http://localhost:9901/stats | grep tracing

# 4. Node.jsがEnvoy経由で接続しているか確認
docker compose exec nodejs-service cat /app/index.js | grep go-service
# → "http://go-service:10000" になっているべき

# 5. コンテナを再ビルド・再起動
docker compose -f docker-compose-envoy.yml down
docker compose -f docker-compose-envoy.yml build nodejs-service
docker compose -f docker-compose-envoy.yml up -d
```

### Envoy設定の確認

```bash
# Envoyの設定をダンプ
curl http://localhost:9901/config_dump

# Envoyクラスターの状態確認
curl http://localhost:9901/clusters

# Envoyリスナーの状態確認
curl http://localhost:9901/listeners
```

### Envoy版でGo → Javaのトレースが途切れる

**症状**: Grafana TempoでPython → Node.js → Goまでは繋がるが、Go → Javaが別のトレースになる

**原因**: Go serviceがJavaへのHTTPリクエストにトレースヘッダーを伝播していない

**確認方法**:
```bash
# 1. Go serviceのログでヘッダー伝播を確認
docker compose logs go-service | grep "Propagating header"

# 出力なし → ヘッダー伝播なし（envoy版）
# 出力あり → ヘッダー伝播あり（envoy-propagation版）

# 2. 使用しているdocker-composeファイルとディレクトリを確認
docker compose ps go-service
# Image列で確認:
# - go-service-envoy:latest → ヘッダー伝播なし
# - go-service-envoy-propagation:latest → ヘッダー伝播あり
```

**解決策**:
```bash
# トレースヘッダー伝播版に切り替え
docker compose -f docker-compose-envoy.yml down
docker compose -f docker-compose-envoy-propagation.yml up --build -d
```

### データベースロックエラー

```bash
# すべてのサービスを停止
docker compose down

# データボリュームを削除
rm -rf data/

# 再起動
docker compose up -d
```

## 🎯 次のステップ

1. **トレースヘッダー伝播の問題を体験**:
   - `docker-compose-envoy.yml`でGo → Javaのトレース途切れを確認
   - `docker-compose-envoy-propagation.yml`で解決策を確認
   - 両者のコード差分を比較: `go-service-ebpf/main.go` vs `go-service-ebpf-propagation/main.go`

2. **異なる計装方式を比較**: 手動計装版とEnvoy版のトレースを比較してみる

3. **Envoy設定をカスタマイズ**: サンプリングレート、タイムアウトなどを調整

4. **他のサービスにもEnvoyを追加**: Node.jsやPythonサービスにもサイドカーを追加

5. **コードを修正**: 新しいエンドポイントやサービスを追加してみる

6. **スパンを追加**: ビジネスロジック用のカスタムスパンを作成

7. **アラートを作成**: Prometheusアラートルールを設定

8. **サンプリングを追加**: collectorやEnvoyでトレースサンプリングを設定

9. **他のバックエンドを試す**: Jaeger、Datadogなどにエクスポート

10. **サービスメッシュを探索**: IstioやLinkerdと統合してみる

## 📚 リソース

### OpenTelemetry
- [OpenTelemetry ドキュメント](https://opentelemetry.io/docs/)
- [OTLP 仕様](https://opentelemetry.io/docs/specs/otlp/)
- [OpenTelemetry Python](https://opentelemetry.io/docs/languages/python/)
- [OpenTelemetry JavaScript](https://opentelemetry.io/docs/languages/js/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Java](https://opentelemetry.io/docs/languages/java/)

### Envoy Proxy
- [Envoy Proxy ドキュメント](https://www.envoyproxy.io/docs/envoy/latest/)
- [Envoy OpenTelemetry トレーシング](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/observability/tracing)
- [Envoy サイドカーパターン](https://www.envoyproxy.io/docs/envoy/latest/configuration/best_practices/edge)

### オブザーバビリティバックエンド
- [Grafana Tempo ドキュメント](https://grafana.com/docs/tempo/latest/)
- [Grafana Loki ドキュメント](https://grafana.com/docs/loki/latest/)
- [Prometheus ドキュメント](https://prometheus.io/docs/)
- [Grafana ドキュメント](https://grafana.com/docs/grafana/latest/)

## 📝 ライセンス

これは教育目的のデモプロジェクトです。

---

**OpenTelemetryの探索を楽しんでください! 🚀**
