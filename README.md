# 🔭 OpenTelemetry マイクロサービス デモ

**Python (FastAPI)**、**Node.js (Express)**、**Go (Gin)**、**Java (Spring Boot)** で構築されたマイクロサービスを使用した包括的なOpenTelemetryデモンストレーション。分散トレーシング、メトリクス、ログの相関機能を完全実装。

## 🎯 機能

- **多言語マイクロサービス**: Python、Node.js、Go、Java
- **完全なオブザーバビリティスタック**: トレース、メトリクス、ログ
- **トレースID相関**: すべてのサービスとログ間でリクエストを追跡
- **OTLP Collector**: 集中型テレメトリー収集
- **Grafana Tempo**: 分散トレーシングバックエンド
- **Grafana Loki**: ログアグリゲーション
- **Prometheus**: メトリクスの収集とクエリ
- **Grafana**: 統合可視化ダッシュボード
- **SQLiteデータベース**: 各サービスごとに独立したデータベース
- **Web UI**: ワークフローをトリガーするインタラクティブダッシュボード
- **Docker Compose**: ワンコマンドでデプロイ

## 🏗️ アーキテクチャ

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

## 📊 サービス概要

| サービス | 言語 | フレームワーク | ポート | データベース | 用途 |
|---------|----------|-----------|------|----------|---------|
| Python | Python 3.11 | FastAPI | 8000 | orders.db | 注文管理 |
| Node.js | Node.js 18 | Express | 3001 | inventory.db | 在庫追跡 |
| Go | Go 1.23 | Gin | 8080 | pricing.db | 価格計算 |
| Java | Java 17 | Spring Boot | 8081 | notifications.db | 通知 |
| Web UI | - | Nginx | 80 | - | ユーザーインターフェース |
| OTLP Collector | - | - | 4317/4318 | - | テレメトリー収集 |
| Grafana Tempo | - | - | 3200 | - | トレースバックエンド |
| Grafana Loki | - | - | 3100 | - | ログアグリゲーション |
| Prometheus | - | - | 9090 | - | メトリクスバックエンド |
| Grafana | - | - | 3000 | - | 統合可視化 |

## 🚀 クイックスタート

### 前提条件

- Docker
- Docker Compose

### デモの起動

```bash
cd otel-demo
docker-compose up -d
```

### サービスへのアクセス

- **Web UI**: http://localhost
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090
- **Python Service**: http://localhost:8000
- **Node.js Service**: http://localhost:3001
- **Go Service**: http://localhost:8080
- **Java Service**: http://localhost:8081
- **Grafana Tempo**: http://localhost:3200
- **Grafana Loki**: http://localhost:3100

### デモの停止

```bash
docker-compose down
```

### 停止とクリーンアップ（ボリューム含む）

```bash
docker-compose down -v
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

### 2. トレースID相関

すべてのレスポンスには`trace_id`フィールドが含まれており、以下が可能です:

- **Grafanaで表示**: すべてのサービス間の完全なリクエストフローを確認
- **ログ検索**: 特定のリクエストに関連するすべてのログエントリを検索
- **問題のデバッグ**: 分散システム全体でエラーを追跡

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
otel-demo/
├── python-service/       # Python FastAPI サービス
│   ├── main.py
│   ├── requirements.txt
│   └── Dockerfile
├── nodejs-service/       # Node.js Express サービス
│   ├── index.js
│   ├── package.json
│   └── Dockerfile
├── go-service/           # Go Gin サービス
│   ├── main.go
│   ├── go.mod
│   └── Dockerfile
├── java-service/         # Java Spring Boot サービス
│   ├── src/
│   ├── pom.xml
│   └── Dockerfile
├── web-ui/               # Webインターフェース
│   └── index.html
├── collector/            # OTLP Collector設定
│   └── otel-collector-config.yaml
├── data/                 # SQLiteデータベース（自動作成）
├── docker-compose.yml    # オーケストレーション
├── prometheus.yml        # Prometheus設定
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

### 3. メトリクス収集
- 各サービスがカスタムメトリクスを送信
- Prometheusがメトリクスをスクレイプして保存
- メトリクスをクエリしてサービスの健全性を把握

### 4. データベース操作
- 各サービスは独自のSQLiteデータベースを持つ
- データベース操作は子スパンとしてトレース
- 正確なSQLクエリとタイミングを確認

### 5. 多言語サポート
- OpenTelemetryは言語間で一貫して動作
- 同じ概念（トレース、メトリクス、ログ）がどこでも適用される
- OTLPが標準プロトコルを提供

## 🐛 トラブルシューティング

### サービスが起動しない

```bash
# ログを確認
docker-compose logs

# 特定のサービスを再起動
docker-compose restart python-service
```

### Grafanaにトレースが表示されない

1. OTLP collectorが実行中か確認: `docker-compose ps otel-collector`
2. collectorのログを確認: `docker-compose logs otel-collector`
3. Tempoのログを確認: `docker-compose logs tempo`
4. サービスがcollectorに到達できるか確認: `docker-compose logs python-service | grep collector`

### データベースロックエラー

```bash
# すべてのサービスを停止
docker-compose down

# データボリュームを削除
rm -rf data/

# 再起動
docker-compose up -d
```

## 🎯 次のステップ

1. **コードを修正**: 新しいエンドポイントやサービスを追加してみる
2. **スパンを追加**: ビジネスロジック用のカスタムスパンを作成
3. **アラートを作成**: Prometheusアラートルールを設定
4. **サンプリングを追加**: collectorでトレースサンプリングを設定
5. **他のバックエンドを試す**: Grafana、Datadogなどにエクスポート

## 📚 リソース

- [OpenTelemetry ドキュメント](https://opentelemetry.io/docs/)
- [Grafana Tempo ドキュメント](https://grafana.com/docs/tempo/latest/)
- [Grafana Loki ドキュメント](https://grafana.com/docs/loki/latest/)
- [Prometheus ドキュメント](https://prometheus.io/docs/)
- [Grafana ドキュメント](https://grafana.com/docs/grafana/latest/)
- [OTLP 仕様](https://opentelemetry.io/docs/specs/otlp/)

## 📝 ライセンス

これは教育目的のデモプロジェクトです。

---

**OpenTelemetryの探索を楽しんでください! 🚀**
