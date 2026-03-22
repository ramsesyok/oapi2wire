# oapi2wire

OpenAPI 定義を正本として、WireMock 実行資産（`mappings/` と `__files/`）を機械生成する CLI ツールです。

## 概要

```text
OpenAPI
  + case YAML
  + responses-root/（レスポンス JSON）
        ↓
    oapi2wire
        ↓
wiremock-out/
  ├─ mappings/
  └─ __files/
        ↓
    WireMock
```

OpenAPI を正本としつつ、返し分け条件（パスパラメータ・クエリパラメータ・JSON ボディ）を case YAML で定義することで、柔軟なモック資産を生成します。

## インストール

```bash
go install github.com/ramsesyok/oapi2wire@latest
```

または、ソースからビルド：

```bash
git clone https://github.com/ramsesyok/oapi2wire.git
cd oapi2wire
go build -o oapi2wire .
```

## クイックスタート

### 1. テンプレート生成

```bash
oapi2wire init \
  --openapi ./openapi.yaml \
  --out-cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

OpenAPI の各 `operationId` に対して case YAML テンプレートとレスポンス JSON 雛形を生成します。

### 2. case YAML を編集

生成された `mock-cases.yaml` を編集して、返し分け条件とレスポンス内容を定義します。

### 3. WireMock 資産を生成

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out
```

### 4. 整合性検証

```bash
oapi2wire validate \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

## コマンドリファレンス

### `init`

```
oapi2wire init --openapi <path> [flags]
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイルのパス |
| `--out-cases` | `mock-cases.yaml` | 出力する case YAML のパス |
| `--responses-root` | `mock-responses` | レスポンス雛形の出力ディレクトリ |
| `--force` | `false` | 既存ファイルを上書きする |
| `--strict` | `false` | OpenAPI の不整合をエラーとして扱う |

### `build`

```
oapi2wire build --openapi <path> [flags]
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイルのパス |
| `--cases` | `mock-cases.yaml` | case YAML のパス |
| `--responses-root` | `mock-responses` | レスポンス JSON のディレクトリ |
| `--out` | `wiremock-out` | 出力ディレクトリ |
| `--clean` | `false` | 出力先を削除してから再生成する |
| `--strict` | `false` | 不明項目をエラーとして扱う |
| `--fail-on-missing-operation` | `false` | operationId が OpenAPI に存在しない場合エラー |
| `--fail-on-missing-body-file` | `false` | bodyFile が存在しない場合エラー |
| `--no-auto-fallback` | `false` | fallback の自動生成を無効化する |

### `validate`

```
oapi2wire validate --openapi <path> [flags]
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイルのパス |
| `--cases` | `mock-cases.yaml` | case YAML のパス |
| `--responses-root` | `mock-responses` | レスポンス JSON のディレクトリ |

## case YAML 仕様

### トップレベル構造

```yaml
version: 1
defaults:
  response:
    headers:
      Content-Type: application/json

cases:
  - ...
```

### 1 ケースの構造

```yaml
- id: getUser_detail_100
  operationId: getUser
  priority: 10
  request:
    pathParams:
      id:
        equalTo: "100"
    query:
      mode:
        equalTo: "detail"
    body:
      equalToJson:
        role: admin
      matchesJsonPath:
        - "$.options.mode[?(@ == 'detail')]"
  response:
    status: 200
    headers:
      Content-Type: application/json
    bodyFile: getUser/getUser_detail_100.json
```

### matcher 記法

| 種別 | フィールド | 対象 |
|---|---|---|
| `equalTo` | 文字列の完全一致 | `pathParams`, `query` |
| `matches` | 正規表現一致 | `pathParams`, `query` |
| `equalToJson` | JSON オブジェクトの一致 | `body` |
| `matchesJsonPath` | JSONPath 式による一致 | `body` |

`equalToJson` と `matchesJsonPath` は同時指定可能です。

### fallback ケース

```yaml
- id: getUser_fallback
  operationId: getUser
  fallback: true
  priority: 10000
  response:
    status: 501
    bodyFile: common/getUser_fallback.json
```

- `fallback: true` のとき `request` は指定できません
- `operationId` ごとに 1 件まで
- 明示 fallback がない場合、`build` 時に自動生成されます（status: 501）

## ディレクトリ構成

### 入力

```text
project/
├─ openapi.yaml
├─ mock-cases.yaml
└─ mock-responses/
   ├─ getUser/
   │  └─ getUser_default.json
   └─ createUser/
      └─ createUser_default.json
```

### 出力（`build` 後）

```text
wiremock-out/
├─ mappings/
│  ├─ getUser__getUser_default.json
│  ├─ createUser__createUser_default.json
│  └─ _generated__fallback__getUser.json
└─ __files/
   ├─ getUser/
   │  └─ getUser_default.json
   ├─ createUser/
   │  └─ createUser_default.json
   └─ _generated/
      └─ fallback/
         └─ getUser.json
```

### 命名規則

| 種別 | パターン |
|---|---|
| case id | `<operationId>_<caseName>` |
| mapping ファイル | `<operationId>__<caseId>.json` |
| bodyFile | `<operationId>/<caseId>.json` |
| 自動 fallback mapping | `_generated__fallback__<operationId>.json` |
| 自動 fallback body | `_generated/fallback/<operationId>.json` |

## サンプル

### OpenAPI

```yaml
openapi: 3.0.3
info:
  title: Sample API
  version: 1.0.0
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - in: path
          name: id
          required: true
          schema:
            type: string
        - in: query
          name: mode
          required: false
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  name:
                    type: string
  /users:
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                role:
                  type: string
      responses:
        '201':
          description: Created
```

### `init` 後の case YAML

```yaml
version: 1
defaults:
  response:
    headers:
      Content-Type: application/json

cases:
  - id: getUser_default
    operationId: getUser
    priority: 100
    request:
      pathParams:
        id:
          equalTo: "TODO"
      # optional query parameters:
      # mode:
      #   equalTo: "TODO"
    response:
      status: 200
      bodyFile: getUser/getUser_default.json

  - id: createUser_default
    operationId: createUser
    priority: 100
    request:
      body:
        equalToJson:
          name: "TODO"
          role: "TODO"
    response:
      status: 201
      bodyFile: createUser/createUser_default.json
```

### 生成される WireMock mapping

```json
{
  "id": "getUser_detail_100",
  "name": "getUser_detail_100",
  "priority": 10,
  "metadata": {
    "generator": "oapi2wire",
    "operationId": "getUser",
    "caseId": "getUser_detail_100"
  },
  "request": {
    "method": "GET",
    "urlPathTemplate": "/users/{id}",
    "pathParameters": {
      "id": { "equalTo": "100" }
    },
    "queryParameters": {
      "mode": { "equalTo": "detail" }
    }
  },
  "response": {
    "status": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "bodyFileName": "getUser/getUser_detail_100.json"
  }
}
```

## バリデーション

### エラー（終了コード 1）

1. OpenAPI が読めない
2. OpenAPI 内の `operationId` が重複している
3. case の `id` が重複している
4. case の `operationId` が OpenAPI に存在しない（`--fail-on-missing-operation` 時）
5. 同じ `operationId` に fallback が複数ある
6. fallback ケースに `request` が指定されている
7. `pathParams` のキー名が OpenAPI の path parameter 名と一致しない
8. `bodyFile` に絶対パス・`..` を含む禁止パスが指定されている
9. `bodyFile` が存在しない（`--fail-on-missing-body-file` 時）

### 警告

1. matcher が空で fallback でもないケース
2. 同一 `operationId` + 同一 request 条件のケースが重複している
3. priority が重複している
4. OpenAPI に `requestBody` があるが body matcher が指定されていない

## 開発

```bash
# テスト実行
go test ./...

# ビルド
go build -o oapi2wire .
```

## ライセンス

[LICENSE](LICENSE) を参照してください。
