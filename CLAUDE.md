# CLAUDE.md

## 概要

このリポジトリでは、OpenAPI 定義と case YAML を入力として、WireMock の `mappings/` と `__files/` を生成する CLI ツール `oapi2wire` を Go で実装する。

想定ユースケースは以下である。

- 自社が OpenAPI と WebAPI を設計・実装する
- フロントエンドは別会社または別チームが SPA として実装する
- フロントエンド開発を早めるため、柔軟にレスポンスを返し分けられるモックサーバ資産を生成したい
- 将来の複数案件でも再利用できるよう、OpenAPI を正本とした共通モック生成基盤を持ちたい

このツール自体は WireMock の実行エンジンではなく、WireMock 実行用ファイルを生成するジェネレータである。

## 目的

実装の目的は次の 3 点である。

1. OpenAPI を正本としつつ、モック定義の保守コストを下げる
2. パスパラメータ、クエリパラメータ、JSON リクエストボディでレスポンスを返し分ける
3. レスポンス本文を設定ファイル本体から分離し、外部 JSON ファイルとして管理する

## スコープ

MVP で対応する範囲は以下とする。

- OpenAPI 3.x YAML / JSON の読込
- operationId を主キーとした operation 解決
- case YAML の読込、検証
- case YAML テンプレートの自動生成
- レスポンス雛形 JSON の自動生成
- WireMock `mappings/*.json` の生成
- WireMock `__files/*` の生成
- fallback stub の自動生成
- `init`, `build`, `validate` の 3 コマンド

MVP で対応しない範囲は以下とする。

- multipart/form-data
- file upload
- SOAP
- stateful scenario
- proxy / callback / fault injection
- OpenAPI extension にモックケースを埋め込む方式
- WireMock Admin API への直接投入専用運用
- 複数 OpenAPI ファイルの統合解決

## 入出力モデル

### 入力

1. OpenAPI 定義ファイル
   - YAML または JSON
   - OpenAPI 3.x
2. case YAML
   - 独自仕様
   - operationId ごとの返し分け条件を定義
3. responses-root
   - `bodyFile` で参照されるレスポンス JSON の格納ディレクトリ

### 出力

WireMock 実行用ディレクトリを生成する。

```text
wiremock-out/
├─ mappings/
└─ __files/
```

## 全体アーキテクチャ

```text
OpenAPI
  + case YAML
  + responses-root
    ↓
oapi2wire
    ↓
wiremock-out/
  ├─ mappings/
  └─ __files/
    ↓
WireMock
```

役割分担は以下の通り。

- OpenAPI
  - API の骨格の正本
  - path / method / operationId / parameter / requestBody / responses を保持
- case YAML
  - モックの返し分けルール
- responses-root
  - レスポンス本文 JSON
- generator
  - WireMock 資産へ変換

## CLI 仕様

### 1. init

OpenAPI から case YAML テンプレートとレスポンス雛形を生成する。

```bash
oapi2wire init \
  --openapi ./openapi.yaml \
  --out-cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

サポートするオプション:

- `--force`
  - 既存ファイルを上書きする
- `--strict`
  - OpenAPI の不整合を厳格に扱う

### 2. build

OpenAPI + case YAML + responses-root から WireMock 実行資産を生成する。

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out
```

サポートするオプション:

- `--clean`
  - 出力先を削除してから生成
- `--strict`
  - 不明項目や未対応記法をエラー化
- `--fail-on-missing-operation`
  - case YAML の operationId が OpenAPI に存在しない場合エラー
- `--fail-on-missing-body-file`
  - `bodyFile` の実ファイルが存在しない場合エラー
- `--no-auto-fallback`
  - fallback 自動生成を無効化

### 3. validate

OpenAPI と case YAML の整合性だけを検証する。

```bash
oapi2wire validate \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses
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
- id: getUser_default
  operationId: getUser
  priority: 100
  fallback: false
  request:
    pathParams:
      id:
        equalTo: "100"
    query:
      mode:
        equalTo: "detail"
    body:
      equalToJson:
        type: admin
      matchesJsonPath:
        - "$.options.mode[?(@ == 'detail')]"
  response:
    status: 200
    headers:
      Content-Type: application/json
    bodyFile: getUser/getUser_default.json
```

### 必須項目

- `id`
- `operationId`
- `response.status`
- `response.bodyFile`

ただし、自動生成 fallback は内部生成物なので入力 case YAML には存在しなくてよい。

### fallback ケース

- `fallback: true` のとき `request` は指定不可
- operationId ごとに 1 件まで
- 明示 fallback がなければ `build` 時に自動生成する

## matcher 記法

MVP では以下のみ対応する。

- `equalTo`
- `matches`
- `equalToJson`
- `matchesJsonPath`

### pathParams

```yaml
request:
  pathParams:
    id:
      equalTo: "100"
```

または

```yaml
request:
  pathParams:
    id:
      matches: "^[0-9]{3}$"
```

### query

```yaml
request:
  query:
    mode:
      equalTo: "detail"
```

または

```yaml
request:
  query:
    mode:
      matches: "detail|simple"
```

### body.equalToJson

```yaml
request:
  body:
    equalToJson:
      role: admin
      active: true
```

### body.matchesJsonPath

```yaml
request:
  body:
    matchesJsonPath:
      - "$.role[?(@ == 'admin')]"
      - "$.options.mode[?(@ == 'detail')]"
```

### body 併用

`equalToJson` と `matchesJsonPath` は同時指定可能とする。

## WireMock 生成ルール

### operation 解決

case YAML の各 case は `operationId` を使って OpenAPI 内の operation にひも付ける。

解決結果として必要なのは以下。

- HTTP method
- path
- path parameter 名一覧
- query parameter 名一覧
- requestBody が JSON かどうか
- representative response status

### request 生成

OpenAPI 側の path をそのまま WireMock の `urlPathTemplate` に使う。

例:

- OpenAPI: `/users/{id}`
- WireMock: `urlPathTemplate: /users/{id}`

`pathParams` があれば `pathParameters` を生成する。
`query` があれば `queryParameters` を生成する。
`body` があれば `bodyPatterns` を生成する。

### bodyPatterns 生成

#### equalToJson

```yaml
request:
  body:
    equalToJson:
      role: admin
```

↓

```json
"bodyPatterns": [
  {
    "equalToJson": "{\n  \"role\": \"admin\"\n}"
  }
]
```

#### matchesJsonPath

```yaml
request:
  body:
    matchesJsonPath:
      - "$.role[?(@ == 'admin')]"
```

↓

```json
"bodyPatterns": [
  {
    "matchesJsonPath": "$.role[?(@ == 'admin')]"
  }
]
```

両方ある場合は `bodyPatterns` に順番に追加する。

### response 生成

case YAML:

```yaml
response:
  status: 200
  headers:
    Content-Type: application/json
  bodyFile: getUser/getUser_default.json
```

WireMock:

```json
{
  "status": 200,
  "headers": {
    "Content-Type": "application/json"
  },
  "bodyFileName": "getUser/getUser_default.json"
}
```

### headers マージ

マージ順序は以下。

1. `defaults.response.headers`
2. `case.response.headers`

同名キーは case 側を優先する。

### metadata

全 mapping に以下の metadata を付与する。

```json
{
  "generator": "oapi2wire",
  "operationId": "getUser",
  "caseId": "getUser_detail_100"
}
```

## fallback 仕様

### 明示 fallback

```yaml
- id: getUser_fallback
  operationId: getUser
  fallback: true
  priority: 10000
  response:
    status: 501
    bodyFile: common/getUser_fallback.json
```

### 自動 fallback

条件:

- `build` 実行時
- `--no-auto-fallback` でない
- 当該 operationId に明示 fallback が存在しない

生成物:

- mapping ファイル名: `_generated__fallback__<operationId>.json`
- body ファイル名: `_generated/fallback/<operationId>.json`
- status: `501`
- body 内容:

```json
{
  "message": "no mock case matched",
  "operationId": "getUser",
  "method": "GET",
  "path": "/users/{id}"
}
```

## init 仕様

### 目的

case YAML を完全手動で作らず、OpenAPI から編集しやすいテンプレートを起こす。

### 生成単位

各 operationId ごとに以下を 1 セット生成する。

- case YAML テンプレート 1 件
- response body 雛形 JSON 1 件

### representative response status の決定

優先順:

1. 最初の 2xx
2. 2xx が無ければ最初の response
3. response が無ければ 200

### path parameter がある場合

```yaml
- id: getUser_default
  operationId: getUser
  priority: 100
  request:
    pathParams:
      id:
        equalTo: "TODO"
  response:
    status: 200
    bodyFile: getUser/getUser_default.json
```

### query parameter がある場合

- required query parameter は実体として書く
- optional query parameter はコメントで案内する

例:

```yaml
- id: searchUsers_default
  operationId: searchUsers
  priority: 100
  request:
    query:
      keyword:
        equalTo: "TODO"
    # optional query parameters:
    # mode:
    #   equalTo: "TODO"
  response:
    status: 200
    bodyFile: searchUsers/searchUsers_default.json
```

### requestBody(JSON) がある場合

- requestBody に example があれば `equalToJson` に使う
- 無ければ schema から最小サンプルを作る

例:

```yaml
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

### response body 雛形生成

優先順位:

1. response example
2. response schema から作る最小 JSON
3. 空オブジェクト `{}`

### schema から最小 JSON を作るルール

- object
  - `properties` を再帰展開
- array
  - 1 要素のみ生成
- string
  - `"TODO"`
- integer
  - `0`
- number
  - `0`
- boolean
  - `true`

## パスルール

### bodyFile

`bodyFile` は `responses-root` からの相対パスとする。

例:

- `bodyFile: getUser/getUser_default.json`
- 実体: `mock-responses/getUser/getUser_default.json`
- 出力: `wiremock-out/__files/getUser/getUser_default.json`

### 禁止事項

- 絶対パス
- `..` を含む相対パス
- `\` を含む Windows 形式パス
- `responses-root` の外を参照するパス

## バリデーション仕様

### エラー

以下は必ずエラーにする。

1. OpenAPI が読めない
2. OpenAPI 内の operationId が重複
3. case の id が重複
4. case の operationId が OpenAPI に存在しない
5. fallback が 1 operationId に複数ある
6. fallback ケースに request がある
7. pathParams 名が OpenAPI の path parameter 名と一致しない
8. `bodyFile` が禁止パス
9. `--fail-on-missing-body-file` 指定時に実ファイルが存在しない

### 警告

以下は warning とする。

1. matcher が空なのに fallback でない
2. 同一 operationId / 同一 request 条件のケースが複数ある
3. priority が重複している
4. optional query parameter がコメントのみで未使用
5. OpenAPI に requestBody があるが case に body matcher が無い

## 実装方針

### 言語

Go を採用する。

理由:

- 単一バイナリ化しやすい
- 社内配布しやすい
- CLI 実装に向く
- YAML / JSON / OpenAPI 処理のライブラリが十分ある

### 推奨ディレクトリ構成

```text
cmd/
  oapi2wire/
internal/
  openapi/
    loader.go
    resolver.go
    sample_generator.go
  cases/
    loader.go
    validator.go
    template_writer.go
  generator/
    mapping_builder.go
    fallback_builder.go
    file_copier.go
  model/
    openapi.go
    case_spec.go
    wiremock.go
  output/
    writer.go
  util/
    yamlutil.go
    pathutil.go
```

### CLI フレームワーク

CLI 実装には Cobra を用いること。

初期化は手作業で骨組みを作らず、必ず `cobra-cli init` を使ってプロジェクトの CLI 雛形を生成すること。

サブコマンド追加も手作業でファイルを増やさず、必ず `cobra-cli add` を使って追加すること。
少なくとも以下のサブコマンドは `cobra-cli add` で生成すること。

- `init`
- `build`
- `validate`

期待する初期作業の流れは次のとおり。

```bash
cobra-cli init
cobra-cli add init
cobra-cli add build
cobra-cli add validate
```

生成された `cmd/` 配下のファイルを土台に実装を進めること。
`rootCmd` や各サブコマンドの生成コードは Cobra の標準構成に沿って維持し、大きく崩さないこと。
CLI の引数・フラグ・実処理は、生成されたコマンド実装から `internal/` 配下の処理を呼び出す形に分離すること。

### コマンドごとの処理フロー

#### init

1. OpenAPI 読込
2. operation 一覧抽出
3. method / path / parameters / requestBody / responses を解析
4. case テンプレート生成
5. response 雛形 JSON 生成
6. `mock-cases.yaml` を出力
7. `responses-root` 配下へ雛形 JSON を出力

#### build

1. OpenAPI 読込
2. case YAML 読込
3. バリデーション
4. case ごとに operationId 解決
5. WireMock mapping 生成
6. bodyFile 存在確認
7. `__files` へコピー
8. fallback 自動生成
9. `mappings/` を出力

#### validate

1. OpenAPI 読込
2. case YAML 読込
3. ルールチェック
4. 結果表示

## Go 実装上の要求

### 構造体設計

少なくとも以下の内部モデルを定義すること。

- `CaseFile`
- `Defaults`
- `CaseSpec`
- `RequestSpec`
- `StringMatcher`
- `BodyMatcher`
- `ResponseSpec`
- `ResolvedOperation`
- `WireMockMapping`
- `WireMockRequest`
- `WireMockResponse`

### JSON Schema 検証

case YAML は YAML として読み込んだ後、JSON 相当の map / struct として JSON Schema 検証を行える設計にすること。

以下のどちらでもよい。

- JSON Schema を Go へ埋め込んで検証
- JSON Schema と等価な独自検証ロジックを Go で実装

ただし、最終的なエラーメッセージは人が修正しやすいものにすること。

### エラー設計

エラーは次の方針で扱うこと。

- CLI では入力起因の問題をまとめて表示できるようにする
- 最初の 1 件で即終了するより、可能な限り複数件収集する
- ただし OpenAPI 自体が読めないなど致命的なものは即終了でもよい

推奨:

- `[]Diagnostic` を集約する
- `Severity` は `error` と `warning`
- `Path`, `Message`, `Hint` を持たせる

例:

```text
error: cases[2].operationId: operationId "createUsr" was not found in OpenAPI
hint: did you mean "createUser"?
```

### ログ方針

- 通常出力は人間向けに簡潔にする
- 詳細デバッグは `--verbose` など将来拡張可能な構成にする
- ライブラリ内部では `fmt.Println` を乱用しない

### 依存ライブラリ方針

必要最小限に留めること。

推奨カテゴリ:

- YAML パーサ
- OpenAPI パーサ
- JSON Schema 検証
- CLI パーサ

ただし、依存を増やしすぎず、保守容易性を優先すること。

## コーディング規約

### 一般

- 可読性を優先する
- 1 関数 1 責務を意識する
- ドメインモデルと I/O 処理を分離する
- `internal/` 配下を責務ごとに分ける
- 早すぎる抽象化を避ける

### 命名

- operationId, caseId などドメイン用語は設計書と合わせる
- WireMock の `bodyFileName`, `urlPathTemplate` などは出力 JSON に合わせる
- ただし Go の構造体名は読みやすい PascalCase を使う

### テスト

最低限、以下をテーブル駆動でテストすること。

1. operationId 解決
2. path/query/body matcher の変換
3. `init` テンプレート生成
4. fallback 自動生成
5. `bodyFile` パス検証
6. 重複 id / 重複 operationId / invalid fallback の検出

ゴールデンファイルテストを推奨する。

対象:

- generated `mock-cases.yaml`
- generated `mappings/*.json`
- generated `__files/*.json`

### JSON 出力

- `mappings/*.json` は安定した整形で出力する
- インデントありで保存する
- 同じ入力なら同じファイル順・同じキー順の出力になるよう意識する

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
          content:
            application/json:
              schema:
                type: object
                properties:
                  result:
                    type: string
```

### init 後の case YAML 例

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

### build 後の WireMock mapping 例

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
      "id": {
        "equalTo": "100"
      }
    },
    "queryParameters": {
      "mode": {
        "equalTo": "detail"
      }
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

## 実装優先順

以下の順序で実装すること。

1. OpenAPI 読込と operationId 解決
2. case YAML 構造体とバリデーション
3. `init` テンプレート生成
4. response 雛形生成
5. `build` による mappings / __files 生成
6. fallback 自動生成
7. `validate` 実装
8. ゴールデンファイルテスト整備

## 受け入れ条件

最低限、以下を満たしたら MVP 完了とする。

1. `init` により operationId ごとの case YAML テンプレートが生成される
2. `init` により response 雛形 JSON が生成される
3. `build` により WireMock `mappings/` と `__files/` が生成される
4. path/query/body matcher が設計どおり変換される
5. explicit fallback と auto fallback の両方が機能する
6. 禁止パス、重複 ID、存在しない operationId を検出できる
7. ゴールデンファイルベースのテストが最低限存在する

## 実装時の注意

- 設計にない独自拡張を勝手に増やさない
- まずは MVP を完成させる
- 例外的な高度機能より、日常運用しやすいことを優先する
- CLI の引数、出力ファイル名、ディレクトリ構成は安定させる
- case YAML と OpenAPI の責務を混ぜない

## 最終的な実装イメージ

このツールは以下の運用を支える。

1. 自社で OpenAPI を更新する
2. `init` でケース雛形を起こす
3. 開発担当が case YAML と response JSON を編集する
4. `build` で WireMock 資産を作る
5. フロントエンド側が WireMock を起動して利用する

この運用が最短で回るように、実装はシンプルさと安定性を最優先にすること。
