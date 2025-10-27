# AWS ECS Fargate デプロイメントガイド

このガイドでは、OpenTelemetry計装デモアプリケーションをAWS ECS Fargateにデプロイする手順を説明します。

## アーキテクチャ概要

- **AWS for Fluent Bit 3.0.0**: メトリクス、トレース、ログを集約
  - OTLP gRPC (4317) / OTLP HTTP (4318) でテレメトリーデータを受信
  - Forward (24224) でコンテナログを収集
- **バックエンド**: Tempo (トレース), Loki (ログ), Prometheus Remote Write対応エンドポイント (メトリクス)
- **アプリケーション**: Python FastAPI, Node.js Express, Go Gin, Java Spring Boot の4つのマイクロサービス

## 前提条件

1. AWS CLI がインストールされ、設定されていること
2. Docker がインストールされていること
3. 以下のAWSリソースが作成済みであること:
   - ECS Cluster
   - VPC とサブネット
   - セキュリティグループ
   - ECR リポジトリ

## ステップ 1: Docker イメージのビルドとプッシュ

### ECRリポジトリの作成

```bash
# リージョンとアカウントIDを設定
export AWS_REGION=ap-northeast-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# ECRリポジトリの作成
# 単一のリポジトリに異なるタグでイメージを管理する方法
aws ecr create-repository --repository-name otel-demo --region ${AWS_REGION}

# または個別のリポジトリを作成する方法
# aws ecr create-repository --repository-name otel-demo-python --region ${AWS_REGION}
# aws ecr create-repository --repository-name otel-demo-nodejs --region ${AWS_REGION}
# aws ecr create-repository --repository-name otel-demo-go --region ${AWS_REGION}
# aws ecr create-repository --repository-name otel-demo-java --region ${AWS_REGION}
# aws ecr create-repository --repository-name otel-demo-fluent-bit --region ${AWS_REGION}
```

### ECRへのログイン

```bash
aws ecr get-login-password --region ${AWS_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com
```

### Docker イメージのビルドとプッシュ

```bash
# プロジェクトルートに移動
cd /path/to/otel-instrumentation-demo

# Custom Fluent Bit Image
docker build -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:fluent-bit ./ADOT/fluent-bit-custom
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:fluent-bit

# Python FastAPI Service
docker build -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:python ./ADOT/python-service
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:python

# Node.js Express Service
docker build -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:nodejs ./ADOT/nodejs-service
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:nodejs

# Go Gin Service
docker build -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:go ./ADOT/go-service
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:go

# Java Spring Boot Service
docker build -t ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:java ./ADOT/java-service
docker push ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:java
```

## ステップ 2: IAMロールの作成

### Task Execution Role

```bash
# Trust Policyの作成
cat > ecs-task-execution-trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

# Roleの作成
aws iam create-role \
  --role-name ecsTaskExecutionRole \
  --assume-role-policy-document file://ecs-task-execution-trust-policy.json

# 必要なポリシーのアタッチ
aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/CloudWatchLogsFullAccess
```

### Task Role

```bash
# Task Roleの作成
aws iam create-role \
  --role-name ecsTaskRole \
  --assume-role-policy-document file://ecs-task-execution-trust-policy.json

# CloudWatch Logsアクセス用のポリシー
aws iam attach-role-policy \
  --role-name ecsTaskRole \
  --policy-arn arn:aws:iam::aws:policy/CloudWatchLogsFullAccess
```

## ステップ 3: Task Definition の登録

`ecs-task-definition.json` を編集して、実際の値に置き換えます:

### 必要な設定値

1. **アカウントID**: `executionRoleArn` と `taskRoleArn` のアカウントID
2. **バックエンドエンドポイント**: Fluent Bitコンテナの環境変数
   - `LOKI_ENDPOINT`: Lokiのエンドポイント (ポート3100)
   - `TEMPO_ENDPOINT`: Tempoのエンドポイント (ポート4318)
   - `THANOS_ENDPOINT`: Prometheus Remote Write対応エンドポイント (ポート19291)
3. **ECSクラスター名**: `ECS_CLUSTER` (オプション)

### Task Definitionの編集例

```bash
cd ADOT

# 元のファイルのバックアップ
cp ecs-task-definition.json ecs-task-definition.json.backup

# 環境に合わせて編集（エディタで直接編集することを推奨）
# 以下は一括置換の例
export LOKI_ENDPOINT="your-loki-nlb.elb.ap-northeast-1.amazonaws.com"
export TEMPO_ENDPOINT="your-tempo-nlb.elb.ap-northeast-1.amazonaws.com"
export THANOS_ENDPOINT="your-thanos-nlb.elb.ap-northeast-1.amazonaws.com"

# macOSの場合
sed -i '' "s/637423497892/${AWS_ACCOUNT_ID}/g" ecs-task-definition.json
sed -i '' "s/lee-o11y-nlb-46edf6e7bd8276ca.elb.ap-northeast-1.amazonaws.com/${LOKI_ENDPOINT}/g" ecs-task-definition.json

# Linuxの場合
# sed -i "s/637423497892/${AWS_ACCOUNT_ID}/g" ecs-task-definition.json
# sed -i "s/lee-o11y-nlb-46edf6e7bd8276ca.elb.ap-northeast-1.amazonaws.com/${LOKI_ENDPOINT}/g" ecs-task-definition.json

# Task Definitionの登録
aws ecs register-task-definition \
  --cli-input-json file://ecs-task-definition.json \
  --region ${AWS_REGION}
```

### Task Definition の構成

このTask Definitionには以下の5つのコンテナが含まれています:

1. **fluent-bit**: AWS for Fluent Bit 3.0.0をベースにしたカスタムイメージ
   - FireLens設定でカスタム設定ファイル (`/custom-fluent-bit.conf`) を使用
   - ポート: 4317 (OTLP gRPC), 4318 (OTLP HTTP), 24224 (Forward)

2. **python-service**: Python FastAPI サービス
   - ポート: 8000
   - OTLP gRPC でテレメトリーを送信

3. **nodejs-service**: Node.js Express サービス
   - ポート: 3000
   - OTLP gRPC でテレメトリーを送信

4. **go-service**: Go Gin サービス
   - ポート: 8080
   - OTLP gRPC でテレメトリーを送信

5. **java-service**: Java Spring Boot サービス
   - ポート: 8081 (アプリケーション), 9464 (Prometheusメトリクス)
   - OTLP gRPC でトレースとログを送信
   - Prometheusメトリクスは9464ポートで直接公開

## ステップ 4: ECS Service の作成

```bash
# セキュリティグループとサブネットを設定
export SECURITY_GROUP_ID=<your-security-group-id>
export SUBNET_ID_1=<your-subnet-id-1>
export SUBNET_ID_2=<your-subnet-id-2>
export ECS_CLUSTER_NAME=otel-demo-cluster

# ECS Clusterの作成（未作成の場合）
aws ecs create-cluster \
  --cluster-name ${ECS_CLUSTER_NAME} \
  --region ${AWS_REGION}

# ECS Serviceの作成
aws ecs create-service \
  --cluster ${ECS_CLUSTER_NAME} \
  --service-name otel-demo-service \
  --task-definition otel-demo-app \
  --desired-count 1 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[${SUBNET_ID_1},${SUBNET_ID_2}],securityGroups=[${SECURITY_GROUP_ID}],assignPublicIp=ENABLED}" \
  --region ${AWS_REGION}
```

## ステップ 5: セキュリティグループの設定

以下のインバウンドルールを設定してください:

| プロトコル | ポート範囲 | ソース | 説明 |
|----------|----------|-------|------|
| TCP | 8000 | 0.0.0.0/0 または ALB/NLB SG | Python FastAPI Service |
| TCP | 3000 | 0.0.0.0/0 または ALB/NLB SG | Node.js Express Service |
| TCP | 8080 | 0.0.0.0/0 または ALB/NLB SG | Go Gin Service |
| TCP | 8081 | 0.0.0.0/0 または ALB/NLB SG | Java Spring Boot Service |
| TCP | 9464 | Prometheus SG | Java Prometheusメトリクス |
| TCP | 2020 | VPC CIDR | Fluent Bit HTTP Server (モニタリング) |

**注意**:
- 4317, 4318, 24224ポートは`localhost`通信のため、セキュリティグループで開放する必要はありません
- 本番環境では、各サービスポートはALB/NLBのセキュリティグループのみからのアクセスを許可することを推奨します

## ステップ 6: 動作確認

### サービスの状態確認

```bash
# タスクの状態確認
aws ecs list-tasks \
  --cluster ${ECS_CLUSTER_NAME} \
  --service-name otel-demo-service \
  --region ${AWS_REGION}

# タスクの詳細確認
aws ecs describe-tasks \
  --cluster ${ECS_CLUSTER_NAME} \
  --tasks <task-arn> \
  --region ${AWS_REGION}
```

### ログの確認

```bash
# CloudWatch Logsの確認
aws logs tail /ecs/otel-demo-app --follow --region ${AWS_REGION}
```

### アプリケーションへのアクセス

タスクのパブリックIPアドレスを取得して、各サービスにアクセス:

```bash
# タスクのパブリックIPを取得
TASK_ARN=$(aws ecs list-tasks --cluster ${ECS_CLUSTER_NAME} --service-name otel-demo-service --region ${AWS_REGION} --query 'taskArns[0]' --output text)
TASK_IP=$(aws ecs describe-tasks --cluster ${ECS_CLUSTER_NAME} --tasks ${TASK_ARN} --region ${AWS_REGION} --query 'tasks[0].attachments[0].details[?name==`networkInterfaceId`].value' --output text | xargs -I {} aws ec2 describe-network-interfaces --network-interface-ids {} --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "Task Public IP: ${TASK_IP}"
```

各サービスのエンドポイント:
- **Python FastAPI Service**: `http://${TASK_IP}:8000`
  - Health Check: `http://${TASK_IP}:8000/health`
  - Order作成: `POST http://${TASK_IP}:8000/orders`
- **Node.js Express Service**: `http://${TASK_IP}:3000`
  - Health Check: `http://${TASK_IP}:3000/health`
  - 在庫確認: `GET http://${TASK_IP}:3000/inventory/{item_id}`
- **Go Gin Service**: `http://${TASK_IP}:8080`
  - Health Check: `http://${TASK_IP}:8080/health`
  - 価格計算: `POST http://${TASK_IP}:8080/calculate-price`
- **Java Spring Boot Service**: `http://${TASK_IP}:8081`
  - Health Check: `http://${TASK_IP}:8081/health`
  - 通知送信: `POST http://${TASK_IP}:8081/notifications`
  - Prometheusメトリクス: `http://${TASK_IP}:9464/metrics`

## トラブルシューティング

### Fluent Bit が起動しない

**原因と対処法**:

1. **カスタムDockerイメージの問題**
   - イメージが正しくビルドされているか確認
   - 設定ファイル（`custom-fluent-bit.conf`）がイメージに含まれているか確認
   ```bash
   # イメージの確認
   docker run --rm ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/otel-demo:fluent-bit ls -la /
   ```

2. **FireLens設定の問題**
   - Task Definitionで`config-file-value`が`/custom-fluent-bit.conf`になっているか確認
   - `firelensConfiguration.type`が`fluentbit`になっているか確認

3. **CloudWatch Logsでエラー確認**
   ```bash
   aws logs tail /ecs/otel-demo-app --follow --filter-pattern "fluent-bit" --region ${AWS_REGION}
   ```

### アプリケーションからOTLPデータが送信されない

**確認項目**:

1. **Fluent Bitの起動確認**
   ```bash
   # Fluent Bit HTTP Serverで確認
   curl http://${TASK_IP}:2020/api/v1/health
   ```

2. **環境変数の確認**
   - Python/Node.js/Go/Javaサービスの`OTEL_EXPORTER_OTLP_*_ENDPOINT`が`http://localhost:4317`になっているか
   - `OTEL_EXPORTER_OTLP_PROTOCOL`が`grpc`になっているか

3. **コンテナ間通信の確認**
   - すべてのコンテナが同じTask内で起動しているか確認（`awsvpc`ネットワークモードで`localhost`通信が可能）
   - `dependsOn`設定でFluent Bitが先に起動するようになっているか確認

4. **アプリケーションログの確認**
   ```bash
   aws logs tail /ecs/otel-demo-app --follow --filter-pattern "python-service" --region ${AWS_REGION}
   ```

### バックエンドにデータが届かない

**確認項目**:

1. **Fluent Bit設定の確認**
   - `LOKI_ENDPOINT`, `TEMPO_ENDPOINT`, `THANOS_ENDPOINT`環境変数が正しく設定されているか
   - エンドポイントがホスト名のみでポート番号を含んでいないか確認（ポートは設定ファイルで指定）

2. **ネットワーク接続の確認**
   ```bash
   # タスク内でネットワーク接続を確認（ECS Execが有効な場合）
   aws ecs execute-command \
     --cluster ${ECS_CLUSTER_NAME} \
     --task ${TASK_ARN} \
     --container fluent-bit \
     --interactive \
     --command "/bin/sh"

   # コンテナ内で実行
   curl -v http://${LOKI_ENDPOINT}:3100/ready
   curl -v http://${TEMPO_ENDPOINT}:4318/v1/traces
   ```

3. **バックエンドのログ確認**
   - Loki: ログが受信されているか確認
   - Tempo: トレースが受信されているか確認
   - Prometheus Remote Write: メトリクスが受信されているか確認

4. **Fluent Bitのメトリクス確認**
   ```bash
   # Fluent Bitの内部メトリクスを確認
   curl http://${TASK_IP}:2020/api/v1/metrics
   ```

### コンテナが頻繁に再起動する

**確認項目**:

1. **Health Checkの失敗**
   - Health Checkのタイムアウト設定を確認
   - `startPeriod`が十分な時間設定されているか確認（現在60秒）

2. **メモリ/CPU不足**
   - Task Definitionのリソース割り当てを確認
   - CloudWatch Metricsでリソース使用率を確認
   ```bash
   aws cloudwatch get-metric-statistics \
     --namespace AWS/ECS \
     --metric-name MemoryUtilization \
     --dimensions Name=ServiceName,Value=otel-demo-service Name=ClusterName,Value=${ECS_CLUSTER_NAME} \
     --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
     --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
     --period 300 \
     --statistics Average \
     --region ${AWS_REGION}
   ```

### ECS Execの有効化（デバッグ用）

デバッグを容易にするため、ECS Execを有効化することを推奨します:

```bash
# Serviceの更新（ECS Exec有効化）
aws ecs update-service \
  --cluster ${ECS_CLUSTER_NAME} \
  --service otel-demo-service \
  --enable-execute-command \
  --region ${AWS_REGION}

# Task RoleにSSMポリシーを追加
aws iam attach-role-policy \
  --role-name ecsTaskRole \
  --policy-arn arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore
```

## スケーリング

### タスク数の調整

```bash
aws ecs update-service \
  --cluster ${ECS_CLUSTER_NAME} \
  --service otel-demo-service \
  --desired-count 3 \
  --region ${AWS_REGION}
```

### オートスケーリングの設定

```bash
# Application Auto Scalingターゲットの登録
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/${ECS_CLUSTER_NAME}/otel-demo-service \
  --min-capacity 1 \
  --max-capacity 10 \
  --region ${AWS_REGION}

# スケーリングポリシーの作成
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/${ECS_CLUSTER_NAME}/otel-demo-service \
  --policy-name cpu-scaling-policy \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration file://scaling-policy.json \
  --region ${AWS_REGION}
```

`scaling-policy.json`:
```json
{
  "TargetValue": 70.0,
  "PredefinedMetricSpecification": {
    "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
  },
  "ScaleInCooldown": 300,
  "ScaleOutCooldown": 60
}
```

## クリーンアップ

リソースを削除する際は、以下の順序で実行してください:

```bash
# 1. ECS Serviceの削除
aws ecs update-service \
  --cluster ${ECS_CLUSTER_NAME} \
  --service otel-demo-service \
  --desired-count 0 \
  --region ${AWS_REGION}

aws ecs delete-service \
  --cluster ${ECS_CLUSTER_NAME} \
  --service otel-demo-service \
  --force \
  --region ${AWS_REGION}

# 2. Task Definitionの登録解除（全リビジョン）
for revision in $(aws ecs list-task-definitions --family-prefix otel-demo-app --region ${AWS_REGION} --query 'taskDefinitionArns[]' --output text); do
  aws ecs deregister-task-definition --task-definition ${revision} --region ${AWS_REGION}
done

# 3. ECS Clusterの削除
aws ecs delete-cluster \
  --cluster ${ECS_CLUSTER_NAME} \
  --region ${AWS_REGION}

# 4. CloudWatch Logsのロググループ削除
aws logs delete-log-group \
  --log-group-name /ecs/otel-demo-app \
  --region ${AWS_REGION}

# 5. ECRリポジトリの削除
aws ecr delete-repository \
  --repository-name otel-demo \
  --force \
  --region ${AWS_REGION}

# 6. IAMロールの削除（必要に応じて）
# ポリシーのデタッチ
aws iam detach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy

aws iam detach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::aws:policy/CloudWatchLogsFullAccess

aws iam detach-role-policy \
  --role-name ecsTaskRole \
  --policy-arn arn:aws:iam::aws:policy/CloudWatchLogsFullAccess

aws iam detach-role-policy \
  --role-name ecsTaskRole \
  --policy-arn arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore

# ロールの削除
aws iam delete-role --role-name ecsTaskExecutionRole
aws iam delete-role --role-name ecsTaskRole
```

## 補足情報

### Fluent Bit の設定ファイルについて

このデプロイメントでは、カスタムFluent Bit設定ファイル（`ADOT/fluent-bit-custom/custom-fluent-bit.conf`）を使用しています:

**INPUT**:
- `forward` (24224): FireLensからコンテナログを受信
- `opentelemetry` (4317): OTLP gRPCトレース/メトリクスを受信
- `opentelemetry` (4318): OTLP HTTPトレース/メトリクスを受信

**FILTER**:
- `aws`: ECSメタデータを自動付与
- `modify`: カスタムラベルの追加

**OUTPUT**:
- `loki`: コンテナログをLokiに送信
- `opentelemetry`: トレースをTempoに送信（OTLP HTTP形式）
- `prometheus_remote_write`: メトリクスをPrometheus Remote Write対応エンドポイントに送信
- `cloudwatch_logs`: CloudWatch Logsにバックアップ

### 各サービスのOpenTelemetry設定

#### Python FastAPI Service
- **SDK**: AWS Distro for OpenTelemetry Python
- **自動計装**: `opentelemetry-instrument`コマンドで起動
- **エクスポーター**: OTLP gRPC
- **特徴**: ログ自動計装とログコリレーション有効

#### Node.js Express Service
- **SDK**: OpenTelemetry JS
- **自動計装**: `@opentelemetry/auto-instrumentations-node`
- **エクスポーター**: OTLP gRPC
- **特徴**: Express、HTTP、その他の自動計装

#### Go Gin Service
- **SDK**: OpenTelemetry Go
- **計装方式**: 手動計装
- **エクスポーター**: OTLP gRPC
- **特徴**: Ginフレームワークのミドルウェア使用

#### Java Spring Boot Service
- **SDK**: OpenTelemetry Java Agent
- **自動計装**: `-javaagent`オプションで起動
- **エクスポーター**: OTLP gRPC (トレース/ログ) + Prometheus (メトリクス)
- **特徴**: Exemplar対応、Prometheusメトリクスを9464ポートで直接公開

### ローカル開発との違い

このECSデプロイメントは、`docker-compose.yml`のローカル環境と以下の点で異なります:

1. **テレメトリーコレクター**:
   - ローカル: ADOT Collector
   - ECS: AWS for Fluent Bit 3.0.0 (FireLens統合)

2. **ログ収集**:
   - ローカル: Docker Loki Driver
   - ECS: FireLens (Fluent Bit) → Loki + CloudWatch Logs

3. **ネットワーク**:
   - ローカル: Docker Bridge Network
   - ECS: AWS VPC (awsvpc mode)

4. **サービス間通信**:
   - ローカル: サービス名でDNS解決
   - ECS: localhost通信（同一Task内）

## 参考資料

- [AWS for Fluent Bit](https://github.com/aws/aws-for-fluent-bit)
- [Fluent Bit Documentation](https://docs.fluentbit.io/)
- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS ECS FireLens](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/using_firelens.html)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [AWS Distro for OpenTelemetry](https://aws-otel.github.io/)
- [OpenTelemetry Python](https://opentelemetry-python.readthedocs.io/)
- [OpenTelemetry JavaScript](https://opentelemetry.io/docs/languages/js/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Java](https://opentelemetry.io/docs/languages/java/)
