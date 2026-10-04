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

| 資料 | 内容 |
|---|---|
| [利用マニュアル](docs/manual.md) | コマンド・case YAML・fallback・エラーと警告・出力・WireMock の起動・runnora との組み合わせ・運用・困ったとき |
| [PetStore チュートリアル](docs/tutorial/index.md) | 雛形の生成から WireMock の起動・curl での確認までを順に試す |
| [設計書](docs/design.md) | 設計の考え方と仕様 |

## インストール

[Releases](https://github.com/ramsesyok/oapi2wire/releases) から OS / アーキテクチャに合ったアーカイブ（Windows は `oapi2wire_v<バージョン>_windows_amd64.zip`）をダウンロードし、展開した `oapi2wire` を PATH の通った場所に置きます。`oapi2wire --version` でバージョンを確認できます。

Go がある場合は `go install` でも入れられます。

```bash
go install github.com/ramsesyok/oapi2wire@latest
```

バージョンを固定して利用する場合は、[Releases](https://github.com/ramsesyok/oapi2wire/releases) のタグを指定します。

```bash
go install github.com/ramsesyok/oapi2wire@v<バージョン>
```

または、ソースからビルド：

```bash
git clone https://github.com/ramsesyok/oapi2wire.git
cd oapi2wire
go build -o oapi2wire .
```

### リリースの作り方（メンテナ向け）

`v0.4.0` のような `v` で始まるタグを付けると、GitHub Actions（`.github/workflows/release.yml`）がテストを実行してから、
GoReleaser（`.goreleaser.yaml`）で Linux / macOS / Windows（amd64 / arm64）向けの実行ファイルを作り、Release に添付します。

- ブラウザの場合：Releases → Draft a new release → Choose a tag で新しいタグ（例：`v0.4.0`）を入力し、`main` を対象に Publish release する
- Actions の画面から：Release → Run workflow でタグ名を入力する（`main` の先頭にタグを付けてリリースする）

## クイックスタート

### 1. テンプレート生成

```bash
oapi2wire init \
  --openapi ./openapi.yaml \
  --out-cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --tags pet
```

OpenAPI の各 `operationId` に対して case YAML テンプレートとレスポンス JSON 雛形を生成します（ケースは operationId の辞書順）。
`--tags` を指定した場合は、指定 tag を持つ operation のみを対象にします。
case YAML が既にあると止まります。`--force` は case YAML に加えて**既存のレスポンス JSON も雛形で上書きする**ので、編集済みのプロジェクトでは別の場所に書き出してください。

### 2. case YAML を編集

生成された `mock-cases.yaml` を編集して、返し分け条件とレスポンス内容を定義します。

### 3. WireMock 資産を生成

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out \
  --tags pet
```

### 4. 整合性検証（build の前に単独で実行することもできます）

```bash
oapi2wire validate \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --tags pet
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
| `--force` | `false` | 既存ファイル（case YAML とレスポンス JSON）を上書きする |
| `--strict` | `false` | OpenAPI の警告もエラーとして扱う |
| `--tags` | なし | 対象にする OpenAPI operation tag（複数指定可） |

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
| `--strict` | `false` | 警告が 1 件でもあればエラーとして扱う |
| `--fail-on-missing-operation` | `false` | operationId が OpenAPI に存在しない場合エラー（指定しないと警告で、そのケースは出力しない） |
| `--fail-on-missing-body-file` | `false` | bodyFile が存在しない場合エラー（指定しないと警告で、本文ファイルはコピーされない） |
| `--no-auto-fallback` | `false` | fallback の自動生成を無効化する |
| `--tags` | なし | 対象にする OpenAPI operation tag（複数指定可） |

### `validate`

```
oapi2wire validate --openapi <path> [flags]
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイルのパス |
| `--cases` | `mock-cases.yaml` | case YAML のパス |
| `--responses-root` | `mock-responses` | レスポンス JSON のディレクトリ |
| `--tags` | なし | 対象にする OpenAPI operation tag（複数指定可） |

validate はファイルを書き出しません。`--fail-on-missing-*` がないため、存在しない operationId と見つからない bodyFile は警告になります。

どのコマンドも、成功すれば終了コード 0、エラーがあれば 1 です。`oapi2wire --version` で版を表示します。

## OpenAPI tag フィルタ

`init` / `validate` / `build` は OpenAPI operation の `tags` で対象 operation を絞り込めます。

```bash
oapi2wire init --openapi ./openapi.yaml --tags pet
oapi2wire validate --openapi ./openapi.yaml --cases ./mock-cases.yaml --tags pet
oapi2wire build --openapi ./openapi.yaml --cases ./mock-cases.yaml --tags pet
```

- `--tags` 未指定の場合は、従来どおり全 operation が対象です
- `--tags pet` の場合は `tags: ["pet"]` を持つ operation のみ対象です
- `--tags pet,store` の場合は `pet` または `store` を持つ operation が対象です
- tag 比較は完全一致・大文字小文字を区別します
- tag フィルタ指定時、tag を持たない operation は対象外です
- `build` の自動 fallback 生成も tag 対象 operation のみに行われます

tag フィルタ指定時に case YAML 内へ対象外 operation の case が残っている場合、`validate` は missing operation エラーとは扱わず、対象外であることを warning として報告します。

Go の公開 API からも同じフィルタを利用できます。

```go
result, err := oapi2wire.Init(oapi2wire.InitOptions{
	OpenAPIPath:   "./openapi.yaml",
	OutCasesPath:  "./mock-cases.yaml",
	ResponsesRoot: "./mock-responses",
	Tags:          []string{"pet"},
})
```

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
- 明示 fallback がない場合、`build` 時に自動生成されます（status: 501、priority: 1000000。`--no-auto-fallback` で無効）

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
      tags:
        - pet
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
      tags:
        - user
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
```

### 生成される WireMock mapping

「1 ケースの構造」の `getUser_detail_100`（body の条件を除いたもの）から作られる `mappings/getUser__getUser_detail_100.json`：

```json
{
  "id": "643ae8a0-de04-5a84-afb5-78fb3d272bd9",
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

`id` は WireMock の決まりで UUID です。operationId と caseId から決まる値なので、同じ入力なら何度 build しても同じ出力になります（生成物の差分を確認したり、Git で管理したりできます）。

## バリデーション

### エラー（終了コード 1）

1. OpenAPI が読めない
2. OpenAPI 内の `operationId` が重複している
3. case の `id` が重複している
4. case の `operationId` が OpenAPI に存在しない（`build --fail-on-missing-operation` 時）
5. 同じ `operationId` に fallback が複数ある
6. fallback ケースに `request` が指定されている
7. `pathParams` のキー名が OpenAPI の path parameter 名と一致しない
8. `bodyFile` に絶対パス・`..`・`\` を含む禁止パスが指定されている
9. `bodyFile` が存在しない（`build --fail-on-missing-body-file` 時）

### 警告

1. case の `operationId` が OpenAPI に存在しない（`--fail-on-missing-operation` なし。build はそのケースを出力しない）
2. `bodyFile` が存在しない（`--fail-on-missing-body-file` なし）
3. `--tags` の対象外の operation のケース
4. matcher が空で fallback でもないケース
5. 同一 `operationId` + 同一 request 条件のケースが重複している
6. priority が重複している
7. OpenAPI に `requestBody` があるが body matcher が指定されていない

`build --strict` / `init --strict` は、警告が 1 件でもあればエラーにします。出力の例と、どのコマンドで何が出るかの一覧は [利用マニュアル](docs/manual.md) の 6 章を参照してください。

## 開発

```bash
# テスト実行
go test ./...

# ビルド
go build -o oapi2wire .
```

## ライセンス

自作部分は [MIT License](LICENSE)（Copyright (c) 2026 ramsesyok）です。
依存ライブラリの表示は [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)、利用・配布時の条件は [ライセンス方針](docs/license-policy.md) を参照してください。
