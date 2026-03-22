# 1. 目的

自社が管理する OpenAPI 定義を正本として、フロントエンド開発用の WireMock 実行資産を機械生成する。

今回ほしいものは次の 3 つです。

1. OpenAPI に追従しやすいこと
2. パスパラメータ・クエリパラメータ・JSON ボディでレスポンスを返し分けられること
3. レスポンス本文は設定ファイルとは別ファイルとして管理できること

このため、以下の 3 層構成にする。

1. OpenAPI
   API の骨格の正本
2. case YAML
   モックの返し分け条件
3. 生成ツール
   WireMock の `mappings/` と `__files/` を生成

# 2. 全体構成

```text
OpenAPI
  + case YAML
  + responses-root 配下のレスポンスJSON
    ↓
oapi2wire
    ↓
wiremock-out/
  ├─ mappings/
  └─ __files/
    ↓
WireMock
```

# 3. 役割分担

## 3.1 OpenAPI

持つものは以下。

* path
* method
* operationId
* path parameter / query parameter / requestBody の定義
* response の status / schema / example

OpenAPI は API の設計正本であり、モックケースの細かい返し分けまでは持たせない。

## 3.2 case YAML

持つものは以下。

* operationId ごとのモックケース
* path parameter の一致条件
* query parameter の一致条件
* request body(JSON) の一致条件
* 返す status
* 返す bodyFile
* priority
* 必要に応じた明示 fallback

## 3.3 responses-root

持つものは以下。

* 実際に返すレスポンス本文 JSON
* case YAML の `bodyFile` から相対参照されるファイル群

## 3.4 生成ツール

持つ責務は以下。

* OpenAPI を読む
* case YAML を読む
* operationId から path / method を解決する
* WireMock mapping JSON を生成する
* bodyFile を `__files/` へコピーする
* fallback を自動生成する
* case YAML テンプレートとレスポンス雛形を初期生成する

# 4. コマンド仕様

CLI は 3 コマンドに分ける。

## 4.1 init

OpenAPI から case YAML テンプレートとレスポンス雛形を生成する。

```bash
oapi2wire init \
  --openapi ./openapi.yaml \
  --out-cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

必要に応じて以下を持つ。

```bash
--force
--strict
```

意味は以下。

* `--force`
  既存ファイルがあっても上書きする
* `--strict`
  OpenAPI の不整合を厳密にエラーにする

## 4.2 build

OpenAPI + case YAML + responses-root から WireMock 実行資産を生成する。

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out
```

追加オプションは以下。

```bash
--clean
--strict
--fail-on-missing-operation
--fail-on-missing-body-file
--no-auto-fallback
```

意味は以下。

* `--clean`
  出力先を削除してから再生成
* `--strict`
  不明項目や未対応記法をエラーにする
* `--fail-on-missing-operation`
  case YAML の operationId が OpenAPI に無い場合エラー終了
* `--fail-on-missing-body-file`
  bodyFile が responses-root に無い場合エラー終了
* `--no-auto-fallback`
  fallback 自動生成を無効化

## 4.3 validate

OpenAPI と case YAML の整合性だけ確認する。

```bash
oapi2wire validate \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

# 5. 対応範囲

MVP では以下に対応する。

1. OpenAPI 3.x YAML / JSON
2. operationId 解決
3. path parameter matcher
4. query parameter matcher
5. request body(JSON) matcher
6. response status / headers / bodyFile
7. fallback 自動生成
8. case YAML テンプレート自動生成
9. レスポンス雛形自動生成

MVP では見送る。

1. multipart/form-data
2. file upload
3. SOAP
4. stateful scenario
5. proxy / callback / fault injection
6. OpenAPI extension へのケース埋め込み
7. Admin API への直接投入専用運用

# 6. ディレクトリ構成

## 6.1 入力側

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

## 6.2 出力側

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

# 7. case YAML 仕様

## 7.1 トップレベル

```yaml
version: 1
defaults:
  response:
    headers:
      Content-Type: application/json
cases:
  - ...
```

## 7.2 case 1件の構造

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

## 7.3 必須項目

必須は以下。

* `id`
* `operationId`
* `response.status`
* `response.bodyFile`
  ただし auto fallback の自動生成ケースは内部生成なので不要

## 7.4 任意項目

任意は以下。

* `priority`
* `fallback`
* `request.pathParams`
* `request.query`
* `request.body`
* `response.headers`

# 8. matcher 記法

要望どおり、MVP では以下だけに絞る。

1. `equalTo`
2. `matches`
3. `equalToJson`
4. `matchesJsonPath`

## 8.1 pathParams

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

## 8.2 query

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

## 8.3 body.equalToJson

```yaml
request:
  body:
    equalToJson:
      role: admin
      active: true
```

## 8.4 body.matchesJsonPath

```yaml
request:
  body:
    matchesJsonPath:
      - "$.role[?(@ == 'admin')]"
      - "$.options.mode[?(@ == 'detail')]"
```

## 8.5 body の併用

`equalToJson` と `matchesJsonPath` は同時指定可能とする。

```yaml
request:
  body:
    equalToJson:
      type: admin
    matchesJsonPath:
      - "$.options.mode[?(@ == 'detail')]"
```

# 9. fallback 仕様

## 9.1 明示 fallback

明示的に fallback を書ける。

```yaml
- id: getUser_fallback
  operationId: getUser
  fallback: true
  priority: 10000
  response:
    status: 501
    bodyFile: common/getUser_fallback.json
```

ルールは以下。

* `fallback: true` のとき `request` は書けない
* `priority` は通常ケースより低優先になるよう大きい値を推奨
* operationId ごとに 1 件まで

## 9.2 自動 fallback

`build` 時に、各 operationId に対して fallback が無ければ自動生成する。

自動生成内容は以下。

* mapping ファイル名
  `_generated__fallback__<operationId>.json`
* body ファイル名
  `_generated/fallback/<operationId>.json`
* status
  501
* body 内容
  自動生成 JSON

例:

```json
{
  "message": "no mock case matched",
  "operationId": "getUser",
  "method": "GET",
  "path": "/users/{id}"
}
```

`--no-auto-fallback` 指定時は生成しない。

# 10. init 機能

## 10.1 方針

case YAML は完全手動ではなく、OpenAPI から初期テンプレートを生成する。

生成単位は operationId ごとに 1 ケース。

つまり `init` 実行後は、最低限以下がそろう。

1. operationId ごとの case YAML テンプレート
2. operationId ごとの response body 雛形 JSON

その後、人が条件値やレスポンス内容を編集する。

## 10.2 生成ルール

各 operation に対して以下を行う。

1. `operationId` を取得
2. `method` と `path` を取得
3. path parameter の有無を調べる
4. query parameter の有無を調べる
5. requestBody(JSON) の有無を調べる
6. 代表 response status を決める
7. case テンプレートを 1 件生成
8. response body 雛形を 1 件生成

## 10.3 代表 response status の決め方

以下の順に決める。

1. 最初の 2xx
2. 2xx が無ければ最初の response
3. response が無ければ 200

## 10.4 path parameter がある場合のテンプレート

OpenAPI:

```yaml
/users/{id}:
  get:
    operationId: getUser
```

生成例:

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

## 10.5 query parameter がある場合のテンプレート

required の query parameter は実体として生成する。
optional の query parameter はコメントで案内する。

生成例:

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

## 10.6 requestBody(JSON) がある場合のテンプレート

OpenAPI に example があればそれを `equalToJson` に使う。
example が無ければ schema から最小サンプルを作る。

生成例:

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

## 10.7 response body 雛形の生成

優先順位は以下。

1. response example
2. response schema から作る最小 JSON
3. 空オブジェクト `{}`

## 10.8 schema から最小 JSON を作るルール

最小ルールは以下。

* object
  `properties` を再帰展開
* array
  1 要素だけ生成
* string
  `"TODO"`
* integer
  `0`
* number
  `0`
* boolean
  `true`

例:

```yaml
type: object
properties:
  id:
    type: string
  age:
    type: integer
  active:
    type: boolean
```

生成:

```json
{
  "id": "TODO",
  "age": 0,
  "active": true
}
```

## 10.9 init の出力例

生成される `mock-cases.yaml` 例:

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

生成される `mock-responses/getUser/getUser_default.json` 例:

```json
{
  "id": "TODO",
  "name": "TODO"
}
```

生成される `mock-responses/createUser/createUser_default.json` 例:

```json
{
  "result": "TODO"
}
```

# 11. build 時の変換ルール

## 11.1 operationId 解決

各 case は `operationId` により OpenAPI の operation を参照する。
ツールは OpenAPI から以下を取り出す。

* HTTP method
* path

## 11.2 request の生成

例:

OpenAPI path:

```text
/users/{id}
```

case YAML:

```yaml
request:
  pathParams:
    id:
      equalTo: "100"
  query:
    mode:
      equalTo: "detail"
```

生成される WireMock request:

```json
{
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
}
```

## 11.3 body の生成

### equalToJson

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

### matchesJsonPath

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

### 併用

両方指定時は `bodyPatterns` に両方並べる。

## 11.4 response の生成

case YAML:

```yaml
response:
  status: 200
  headers:
    Content-Type: application/json
  bodyFile: getUser/getUser_default.json
```

↓

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

## 11.5 headers のマージ

マージ順は以下。

1. `defaults.response.headers`
2. `case.response.headers`

同名キーは case 側を優先する。

# 12. WireMock mapping 生成例

入力 case:

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
  response:
    status: 200
    bodyFile: getUser/getUser_detail_100.json
```

生成される `mappings/getUser__getUser_detail_100.json`:

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

# 13. bodyFile と responses-root の扱い

## 13.1 基本ルール

`bodyFile` は `responses-root` からの相対パスとする。

例:

```yaml
bodyFile: getUser/getUser_default.json
```

実体は以下に置く。

```text
mock-responses/getUser/getUser_default.json
```

build 時にこれを以下へコピーする。

```text
wiremock-out/__files/getUser/getUser_default.json
```

## 13.2 禁止ルール

以下は禁止。

* 絶対パス
* `..` を含む相対パス
* `responses-root` の外を参照するパス

# 14. バリデーション仕様

## 14.1 エラー

以下はエラー。

1. OpenAPI が読めない
2. OpenAPI 内の operationId が重複
3. case の id が重複
4. case の operationId が OpenAPI に存在しない
5. fallback が 1 operationId に複数ある
6. fallback ケースに request がある
7. pathParams 名が OpenAPI の path parameter 名と一致しない
8. `bodyFile` が禁止パス
9. `--fail-on-missing-body-file` 指定時に実ファイルが無い

## 14.2 警告

以下は警告。

1. matcher が空なのに fallback でない
2. 同一 operationId・同一 request 条件のケースが複数ある
3. priority が重複している
4. optional query parameter がコメントのみで、実ケースに未反映
5. OpenAPI に requestBody があるが case に body matcher が無い

# 15. 実装アーキテクチャ

Go 実装を前提にする。
理由は単一バイナリ化しやすく、社内配布しやすいため。

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

# 16. 内部処理フロー

## 16.1 init

1. OpenAPI 読込
2. operation 一覧抽出
3. 各 operation ごとに method/path/parameters/requestBody/responses を解析
4. case テンプレート生成
5. response 雛形生成
6. `mock-cases.yaml` 出力
7. `responses-root` 配下に雛形 JSON 出力

## 16.2 build

1. OpenAPI 読込
2. case YAML 読込
3. バリデーション
4. 各 case の operationId 解決
5. WireMock mapping 生成
6. bodyFile 存在確認
7. `__files` へコピー
8. fallback 自動生成
9. `mappings/` 出力

## 16.3 validate

1. OpenAPI 読込
2. case YAML 読込
3. ルールチェック
4. 結果表示

# 17. 命名規則

## 17.1 case id

推奨:

```text
<operationId>_<caseName>
```

例:

```text
getUser_detail_100
createUser_admin_error
```

## 17.2 mapping ファイル名

```text
<operationId>__<caseId>.json
```

## 17.3 bodyFile の推奨

```text
<operationId>/<caseId>.json
```

例:

```text
getUser/getUser_detail_100.json
createUser/createUser_admin_error.json
```

# 18. 運用イメージ

## 18.1 初回

1. OpenAPI を用意
2. `init` 実行
3. 生成された `mock-cases.yaml` と `mock-responses/` を編集
4. `build` 実行
5. `wiremock-out/` を別会社へ共有
6. 相手側で WireMock 起動

## 18.2 OpenAPI 更新時

1. OpenAPI 更新
2. `init --force` または差分反映機能でテンプレート更新
3. 必要な case を修正
4. `build` 再実行

# 19. 最小のサンプル

## 19.1 OpenAPI

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

## 19.2 init 後の case YAML

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

## 19.3 編集後の case YAML

```yaml
version: 1
defaults:
  response:
    headers:
      Content-Type: application/json

cases:
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
    response:
      status: 200
      bodyFile: getUser/getUser_detail_100.json

  - id: createUser_admin_error
    operationId: createUser
    priority: 10
    request:
      body:
        matchesJsonPath:
          - "$.role[?(@ == 'admin')]"
    response:
      status: 403
      bodyFile: createUser/createUser_admin_error.json
```

# 20. MVP の確定事項

今回の要件に合わせ、以下を確定とする。

1. matcher 記法は `equalTo`, `matches`, `equalToJson`, `matchesJsonPath` のみ
2. `--responses-root` を持つ
3. fallback stub は自動生成する
4. case YAML は完全手動ではなく、`init` でテンプレート生成する
5. レスポンス本文は case YAML に埋め込まず、別 JSON ファイルとする
6. operationId を主キーとして OpenAPI と case YAML を結び付ける

# 21. 次の実装単位

実装は以下の順で進めるのが安全。

1. OpenAPI 読込と operationId 解決
2. `init` による case YAML テンプレート生成
3. response 雛形生成
4. case YAML 読込とバリデーション
5. `build` による mappings / __files 生成
6. fallback 自動生成
7. `validate` 実装


# 23.  case YAML 用の JSON Schema
YAML はパース後に JSON と同等のデータ構造になるため、この Schema は「YAML を読み込んだ後の構造」を検証する前提です。

想定しているルールは次です。

* `version` は `1`
* `cases` は配列
* matcher 記法は `equalTo`, `matches`, `equalToJson`, `matchesJsonPath` のみ
* `fallback: true` の場合は `request` を書けない
* `bodyFile` は相対パスのみ
* 余計な項目は基本的に禁止

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://example.local/schemas/case-yaml.schema.json",
  "title": "oapi2wire case YAML schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "cases"],
  "properties": {
    "version": {
      "type": "integer",
      "const": 1
    },
    "defaults": {
      "$ref": "#/$defs/defaults"
    },
    "cases": {
      "type": "array",
      "items": {
        "$ref": "#/$defs/case"
      }
    }
  },
  "$defs": {
    "defaults": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "response": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "headers": {
              "$ref": "#/$defs/headers"
            }
          }
        }
      }
    },

    "case": {
      "type": "object",
      "additionalProperties": false,
      "required": ["id", "operationId", "response"],
      "properties": {
        "id": {
          "type": "string",
          "minLength": 1,
          "pattern": "^[A-Za-z0-9_.-]+$"
        },
        "operationId": {
          "type": "string",
          "minLength": 1
        },
        "priority": {
          "type": "integer",
          "minimum": 1
        },
        "fallback": {
          "type": "boolean"
        },
        "request": {
          "$ref": "#/$defs/request"
        },
        "response": {
          "$ref": "#/$defs/response"
        }
      },
      "allOf": [
        {
          "if": {
            "properties": {
              "fallback": {
                "const": true
              }
            },
            "required": ["fallback"]
          },
          "then": {
            "not": {
              "required": ["request"]
            }
          }
        }
      ]
    },

    "request": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "pathParams": {
          "$ref": "#/$defs/namedMatchers"
        },
        "query": {
          "$ref": "#/$defs/namedMatchers"
        },
        "body": {
          "$ref": "#/$defs/bodyMatcher"
        }
      },
      "minProperties": 1
    },

    "namedMatchers": {
      "type": "object",
      "propertyNames": {
        "type": "string",
        "minLength": 1
      },
      "additionalProperties": {
        "$ref": "#/$defs/stringMatcher"
      },
      "minProperties": 1
    },

    "stringMatcher": {
      "oneOf": [
        {
          "type": "object",
          "additionalProperties": false,
          "required": ["equalTo"],
          "properties": {
            "equalTo": {
              "type": "string"
            }
          }
        },
        {
          "type": "object",
          "additionalProperties": false,
          "required": ["matches"],
          "properties": {
            "matches": {
              "type": "string",
              "minLength": 1
            }
          }
        }
      ]
    },

    "bodyMatcher": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "equalToJson": {
          "$ref": "#/$defs/jsonValue"
        },
        "matchesJsonPath": {
          "type": "array",
          "items": {
            "type": "string",
            "minLength": 1
          },
          "minItems": 1
        }
      },
      "anyOf": [
        { "required": ["equalToJson"] },
        { "required": ["matchesJsonPath"] }
      ]
    },

    "response": {
      "type": "object",
      "additionalProperties": false,
      "required": ["status", "bodyFile"],
      "properties": {
        "status": {
          "type": "integer",
          "minimum": 100,
          "maximum": 599
        },
        "headers": {
          "$ref": "#/$defs/headers"
        },
        "bodyFile": {
          "type": "string",
          "minLength": 1,
          "pattern": "^(?!/)(?!.*\\.\\.)(?!.*\\\\).+$"
        }
      }
    },

    "headers": {
      "type": "object",
      "propertyNames": {
        "type": "string",
        "minLength": 1
      },
      "additionalProperties": {
        "type": "string"
      }
    },

    "jsonValue": {
      "oneOf": [
        { "type": "string" },
        { "type": "number" },
        { "type": "integer" },
        { "type": "boolean" },
        { "type": "null" },
        {
          "type": "array",
          "items": {
            "$ref": "#/$defs/jsonValue"
          }
        },
        {
          "type": "object",
          "additionalProperties": {
            "$ref": "#/$defs/jsonValue"
          }
        }
      ]
    }
  }
}
```

補足です。

1. `bodyFile` の pattern

   * 先頭 `/` を禁止
   * `..` を禁止
   * Windows の `\` も禁止
     という最小制約にしています。
     つまり `getUser/getUser_default.json` のような相対パスを想定しています。

2. `fallback: true`

   * この Schema では `request` を書けないようにしています。
   * 「fallback ケースは matcher なし」という設計に合わせています。

3. `equalToJson`

   * JSON object だけでなく、配列・文字列・数値・真偽値・null も許可しています。
   * ただし実務では object が中心になるはずです。

4. `additionalProperties: false`

   * typo を早く見つけるため、かなり厳しめにしています。
   * これは `--strict` 寄りの設計です。

参考として、この Schema に通る最小例はこうです。

```yaml
version: 1
defaults:
  response:
    headers:
      Content-Type: application/json

cases:
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
    response:
      status: 200
      bodyFile: getUser/getUser_detail_100.json

  - id: createUser_admin_error
    operationId: createUser
    priority: 20
    request:
      body:
        matchesJsonPath:
          - "$.role[?(@ == 'admin')]"
    response:
      status: 403
      bodyFile: createUser/createUser_admin_error.json

  - id: getUser_fallback
    operationId: getUser
    fallback: true
    priority: 10000
    response:
      status: 501
      bodyFile: common/getUser_fallback.json
```
