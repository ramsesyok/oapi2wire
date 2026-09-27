# oapi2wire 利用マニュアル

oapi2wire は、OpenAPI 定義を正本として、WireMock が読み込む `mappings/` と `__files/` を生成する CLI です。
WireMock そのものは起動しません（起動は `java -jar wiremock-standalone-*.jar`）。

- 初めて使う場合は、手を動かしながら進める [PetStore チュートリアル](tutorial/index.md) から始めてください。
- 設計の考え方は [設計書](design.md) にあります。
- このマニュアルは、日常の運用で参照する「使い方の全体」をまとめたものです。

## 1. 全体の流れ

```text
OpenAPI ─┬─ oapi2wire init ──→ case YAML（雛形）+ レスポンス JSON（雛形）
         │                         │ 人が編集する（返し分けの条件と応答）
         │                         ▼
         ├─ oapi2wire validate ─→ 整合性の検査（書き出しはしない）
         │
         └─ oapi2wire build ────→ wiremock-out/mappings/ と __files/
                                          │
                                          ▼
                              java -jar wiremock-standalone.jar --root-dir wiremock-out
```

| 入力 | 役割 | 管理 |
|---|---|---|
| OpenAPI（YAML / JSON、3.x） | API の骨格の正本（path、method、operationId、パラメータ、requestBody、responses） | 設計側 |
| case YAML（既定 `mock-cases.yaml`） | operationId ごとの返し分けの条件と、返す応答 | Git で管理する |
| responses-root（既定 `mock-responses/`） | 応答本文の JSON（case YAML の `bodyFile` が参照する） | Git で管理する |
| 出力（既定 `wiremock-out/`） | WireMock の資産。`build` で作り直せる | 同じ入力なら同じ出力になるので、差分を見たいなら Git で管理してもよい |

## 2. インストール

[Releases](https://github.com/ramsesyok/oapi2wire/releases) から OS / アーキテクチャに合ったアーカイブ（Windows は `oapi2wire_v<バージョン>_windows_amd64.zip`）をダウンロードし、展開した `oapi2wire` を PATH の通った場所に置きます。

```bash
oapi2wire --version
```

Go がある場合は `go install github.com/ramsesyok/oapi2wire@latest`（版を固定するなら `@v<バージョン>`）でも入れられます。

WireMock を起動するには Java と WireMock standalone の jar（Maven Central の `org.wiremock:wiremock-standalone`。3.13.2 で確認）が必要です。

## 3. コマンド

どのコマンドも、成功すれば終了コード 0、エラーがあれば 1 で終わります。エラーと警告は標準エラーに出ます（6 章）。

### 3.1 init：雛形を作る

```bash
oapi2wire init --openapi openapi.yaml --out-cases mock-cases.yaml --responses-root mock-responses
```

| オプション | 既定 | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイル |
| `--out-cases` | `mock-cases.yaml` | 書き出す case YAML |
| `--responses-root` | `mock-responses` | レスポンス JSON の雛形を置くディレクトリ |
| `--tags` | なし | 対象にする operation の tag（`--tags pet,store` または複数回指定） |
| `--force` | false | 既存のファイルを上書きする（**注意**：下記） |
| `--strict` | false | OpenAPI の警告もエラーにする |

- operationId ごとに、ケース `<operationId>_default` を 1 件と、レスポンス JSON `<operationId>/<operationId>_default.json` を 1 件作ります。ケースは operationId の辞書順に並びます。
- 雛形の中身は 4 章の「init の雛形の決まり」のとおりです。
- case YAML が既にあると、`file already exists` で止まります（終了コード 1）。
- **`--force` は case YAML だけでなく、既にあるレスポンス JSON も雛形で上書きします。** 編集済みの応答を消さないよう、既存のプロジェクトで `--force` を使う場合は、別のディレクトリに書き出して必要な分だけ取り込んでください。

```bash
# 例: 新しく増えた operation の雛形だけ取り込む
oapi2wire init --openapi openapi.yaml --out-cases tmp/new-cases.yaml --responses-root tmp/new-responses --tags newtag
```

### 3.2 validate：整合性を検査する

```bash
oapi2wire validate --openapi openapi.yaml --cases mock-cases.yaml --responses-root mock-responses
```

| オプション | 既定 | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイル |
| `--cases` | `mock-cases.yaml` | case YAML |
| `--responses-root` | `mock-responses` | レスポンス JSON のディレクトリ |
| `--tags` | なし | 対象にする operation の tag |

- ファイルは書き出しません。問題がなければ `OK: no issues found` と表示します。
- エラーが 1 件でもあれば終了コード 1、警告だけなら 0 です。
- validate には `--fail-on-missing-*` がないので、**存在しない operationId と見つからない bodyFile は警告**として出ます。これらもエラーにしたい場合は、`build` をこのオプション付きで実行してください。

### 3.3 build：WireMock の資産を作る

```bash
oapi2wire build --openapi openapi.yaml --cases mock-cases.yaml --responses-root mock-responses \
  --out wiremock-out --clean --fail-on-missing-operation --fail-on-missing-body-file
```

| オプション | 既定 | 説明 |
|---|---|---|
| `--openapi` | （必須） | OpenAPI ファイル |
| `--cases` | `mock-cases.yaml` | case YAML |
| `--responses-root` | `mock-responses` | レスポンス JSON のディレクトリ |
| `--out` | `wiremock-out` | 出力先 |
| `--clean` | false | 出力先を削除してから作る（消したケースの mapping が残らない。通常は付ける） |
| `--fail-on-missing-operation` | false | case の operationId が OpenAPI にない場合をエラーにする（付けないと警告で、そのケースは出力しない） |
| `--fail-on-missing-body-file` | false | `bodyFile` の実ファイルがない場合をエラーにする（付けないと警告で、mapping は作るが本文ファイルはコピーされない） |
| `--no-auto-fallback` | false | 自動 fallback を作らない（5 章） |
| `--strict` | false | 警告が 1 件でもあればエラーにする |
| `--tags` | なし | 対象にする operation の tag（自動 fallback も対象の operation だけ） |

- build は最初に validate と同じ検査を行い、エラーがあれば何も書き出さずに終了コード 1 で終わります。
- CI やスクリプトでは、`--clean --fail-on-missing-operation --fail-on-missing-body-file` を付けるのがおすすめです（綴りの誤りや本文ファイルの置き忘れを見逃さない）。

### 3.4 共通

```bash
oapi2wire --version     # 版を表示する
oapi2wire --help        # コマンドの一覧
oapi2wire build --help  # オプションの一覧
```

## 4. case YAML

### 4.1 全体

```yaml
version: 1
defaults:
  response:
    headers:                     # 全ケースに付けるヘッダ (ケース側が同名なら上書き)
      Content-Type: application/json
cases:
  - id: getUser_detail_100       # 必須。ファイル全体で一意
    operationId: getUser         # 必須。OpenAPI の operationId
    priority: 10                 # 任意。小さいほど先に照合される (WireMock の priority)
    fallback: false              # 任意。true なら request を書かない (5 章)
    request:                     # 任意。返し分けの条件 (4.2)
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
    response:                    # 必須
      status: 200                # 必須
      headers:                   # 任意 (defaults.response.headers とマージ)
        X-Mock: "true"
      bodyFile: getUser/getUser_detail_100.json   # 必須。responses-root からの相対パス
```

### 4.2 返し分けの条件（matcher）

| 書く場所 | 使える matcher | 意味 |
|---|---|---|
| `request.pathParams.<名前>` | `equalTo` / `matches` | path parameter の完全一致 / 正規表現。名前は OpenAPI の path parameter と同じにする（違うとエラー） |
| `request.query.<名前>` | `equalTo` / `matches` | query parameter の完全一致 / 正規表現 |
| `request.body.equalToJson` | JSON（YAML で書く） | 本文の JSON 全体が一致 |
| `request.body.matchesJsonPath` | JSONPath の配列 | すべての JSONPath に一致（`equalToJson` と併用可。すべて満たすと一致） |

- 値は文字列として比較します。数値の path parameter も `equalTo: "100"` と引用符で書きます。
- 複数の条件を書くと、すべて満たしたときに一致します。
- WireMock は `priority` の小さいケースから照合し、最初に一致したものを返します。条件の厳しいケースほど小さい priority にします。

### 4.3 bodyFile

- `responses-root` からの相対パスで、`/` 区切りで書きます。
- 絶対パス、`..` を含むパス、`\` を含むパスはエラーです。
- 出力では `wiremock-out/__files/<bodyFile>` にコピーされ、mapping の `bodyFileName` になります。

### 4.4 init の雛形の決まり

| OpenAPI | 雛形の中身 |
|---|---|
| path parameter | `pathParams.<名前>.equalTo: "<パラメータの値>"` |
| 必須の query parameter | `query.<名前>.equalTo: "<パラメータの値>"` |
| 任意の query parameter | コメント（`# optional query parameters:`）で、値とともに案内する |
| JSON の requestBody | 本文の値を `body.equalToJson` に入れる |
| 応答の status | 最初の 2xx → なければ最初の応答 → 応答がなければ 200 |
| レスポンス JSON | 応答の本文の値 → 決まらなければ `{}` |

値の決め方（上から順に、最初に見つかったもの）：

| 対象 | 決め方 |
|---|---|
| パラメータ | parameter の `example` → `examples` の最初 → スキーマの値（配列なら要素の値）→ `"TODO"` |
| 本文（リクエスト・応答） | media type の `example` → `examples` の最初 → スキーマの値 |
| スキーマの値 | `example` → `default` → `enum` の最初 → `allOf` / `anyOf` / `oneOf` の最初 → 型による値 |
| 型による値 | object はプロパティを展開、array は 1 要素、string は `"TODO"`（format が `date` / `date-time` / `uuid` ならその形の値）、integer / number は `0`、boolean は `true` |

この決め方は runnora の `runnora generate`（テストケースの生成）と共通です（`pkg/oapisample`）。
同じ OpenAPI から作ったモックの雛形とテストケースは、パラメータ・リクエスト本文・応答本文が同じ値になります。
OpenAPI に `example` を書いておくと、両方の雛形がそのまま一致する具体的な値になります。

`"TODO"` のまま build すると、そのケースは実際のリクエストに一致しません（fallback が返ります）。必ず実際の値に書き換えてください。

## 5. fallback（どのケースにも一致しないとき）

| 種類 | 作り方 | 内容 |
|---|---|---|
| 明示 fallback | case YAML に `fallback: true` のケースを書く（`request` は書けない。operationId ごとに 1 件まで） | 書いたとおりの status と本文。`priority` は大きめ（例 10000）にする |
| 自動 fallback | 明示 fallback のない operationId に、build が自動で作る（`--no-auto-fallback` で無効） | status 501、priority 1000000、本文は下記 |

```json
{
  "message": "no mock case matched",
  "method": "GET",
  "operationId": "getUser",
  "path": "/users/{id}"
}
```

501 とこの本文が返ってきたら、「その operation には到達したが、どのケースの条件にも一致しなかった」という意味です（8 章）。

## 6. エラーと警告

出力の例：

```text
error: cases[4].request.pathParams.userId: path parameter "userId" not found in OpenAPI operation "getUser"
hint: valid path parameters: [id]
warning: cases[3].operationId: operationId "getUsr" was not found in OpenAPI
Error: validation failed with errors
```

`cases[番号]` は case YAML の `cases` の 0 始まりの位置です。

| 内容 | init | validate | build |
|---|---|---|---|
| OpenAPI が読めない、operationId の重複 | エラー | エラー | エラー |
| case の `id` の重複 | ― | エラー | エラー |
| fallback が同じ operationId に複数、fallback に `request` がある | ― | エラー | エラー |
| `pathParams` の名前が OpenAPI にない | ― | エラー | エラー |
| `bodyFile` が禁止パス（絶対・`..`・`\`） | ― | エラー | エラー |
| case の operationId が OpenAPI にない | ― | 警告 | 警告（`--fail-on-missing-operation` でエラー）。そのケースは出力しない |
| `bodyFile` の実ファイルがない | ― | 警告 | 警告（`--fail-on-missing-body-file` でエラー） |
| `--tags` の対象外の operationId のケース | ― | 警告 | 警告。そのケースは出力しない |
| fallback でないのに条件がない | ― | 警告 | 警告 |
| 同じ operationId で条件が同じケースが複数 | ― | 警告 | 警告 |
| 同じ operationId で priority が重複 | ― | 警告 | 警告 |
| requestBody があるのに body の条件がない | ― | 警告 | 警告 |

`--strict`（init・build）を付けると、警告が 1 件でもあれば終了コード 1 になります。

## 7. 出力

```text
wiremock-out/
├─ mappings/
│  ├─ getUser__getUser_detail_100.json      <operationId>__<caseId>.json
│  └─ _generated__fallback__getUser.json    自動 fallback
└─ __files/
   ├─ getUser/getUser_detail_100.json       bodyFile をそのままのパスでコピー
   └─ _generated/fallback/getUser.json      自動 fallback の本文
```

mapping の例：

```json
{
  "id": "643ae8a0-de04-5a84-afb5-78fb3d272bd9",
  "name": "getUser_detail_100",
  "priority": 10,
  "metadata": { "generator": "oapi2wire", "operationId": "getUser", "caseId": "getUser_detail_100" },
  "request": {
    "method": "GET",
    "urlPathTemplate": "/users/{id}",
    "pathParameters": { "id": { "equalTo": "100" } },
    "queryParameters": { "mode": { "equalTo": "detail" } }
  },
  "response": {
    "status": 200,
    "headers": { "Content-Type": "application/json" },
    "bodyFileName": "getUser/getUser_detail_100.json"
  }
}
```

- `id` は WireMock の決まりで UUID です。operationId と caseId から決まる値（名前ベースの UUID）なので、**同じ入力なら何度 build しても同じ**です。`name` と `metadata.caseId` にはケースの `id` が入ります。
- `urlPathTemplate` は OpenAPI の path そのものです。
- `equalToJson` は、WireMock の決まりに合わせて JSON の文字列として出力します。`body` の条件は `equalToJson`、`matchesJsonPath` の順に `bodyPatterns` に並びます。
- `priority` を書かなかったケースは mapping に `priority` を出しません（WireMock の既定の priority になります）。

## 8. WireMock を起動して確かめる

```bash
java -jar wiremock-standalone-3.13.2.jar --root-dir wiremock-out --port 8080
```

- `--bind-address 127.0.0.1` を付けると、自分の PC からだけ接続できます。
- build し直したら、WireMock を再起動してください（または管理 API の `POST /__admin/mappings/reset`）。

確認：

```bash
curl -i http://localhost:8080/users/100?mode=detail                  # ケースに一致すれば 200 と本文
curl -s http://localhost:8080/__admin/mappings | head                 # 読み込まれた mapping
curl -s http://localhost:8080/__admin/requests/unmatched              # どの mapping にも一致しなかったリクエスト
```

Windows PowerShell では `curl.exe` と書いてください（`curl` は別のコマンドの別名です）。

## 9. runnora と組み合わせる

runnora（新形式）では、`runnora.yaml` の環境に WireMock の URL を書き、スイートでその環境を選びます。

```yaml
# runnora.yaml
environments:
  mock:
    description: WireMock (oapi2wire 生成モック)
    vars:
      API_URL: "http://127.0.0.1:8080"          # runbook の runners に ${API_URL} と書く
      RUNNORA_BASE_URL: "http://127.0.0.1:8080" # runnora generate の template の接続先
suites:
  contract-mock:
    env: mock
    select:
      paths: [runbooks/contract/*.suite.yml]
```

```bash
oapi2wire validate --openapi openapi.yaml --cases mock-cases.yaml --responses-root mock-responses
oapi2wire build --openapi openapi.yaml --cases mock-cases.yaml --responses-root mock-responses --out wiremock-out --clean \
  --fail-on-missing-operation --fail-on-missing-body-file
java -jar wiremock-standalone-3.13.2.jar --root-dir wiremock-out --port 8080 --bind-address 127.0.0.1   # 別のターミナル
runnora run --suite contract-mock
```

- runnora のテストが期待する status と本文を、case YAML の `response.status` と `bodyFile` の JSON に合わせます。
- 契約テストの期待ファイルと、モックの `bodyFile` を同じファイルにしておくと、モックと実 API の期待がずれません（実例：[runnora-e2e の api-test](https://github.com/ramsesyok/runnora-e2e/tree/main/api-test)）。

## 10. 運用のしかた

**OpenAPI が変わったとき**

1. `oapi2wire validate` で、既存のケースが新しい OpenAPI と合っているかを確かめる（消えた operationId、変わった path parameter の名前）。
2. 新しい operation の雛形を、別の場所に `init` して必要な分だけ取り込む（3.1 の `--force` の注意）。
3. `oapi2wire build --clean ...` で作り直す。

**tag で分けて使う**

```bash
oapi2wire build --openapi openapi.yaml --cases mock-cases.yaml --tags pet,store --out wiremock-pet --clean
```

tag を指定すると、その tag を持つ operation だけが対象です（tag のない operation は対象外）。
case YAML に対象外の operation のケースが残っていても、エラーにはならず警告になります。

## 11. 困ったとき

| 症状 | 確認すること |
|---|---|
| 501 `no mock case matched` が返る | 条件がリクエストと一致していない。`"TODO"` が残っていないか、数値を引用符で書いたか、query の名前、JSON 本文のフィールドの過不足（`equalToJson` は全体一致。一部だけ見るなら `matchesJsonPath`） |
| 404 が返る | その path と method の mapping がない。operationId の綴り、`--tags` の対象か、build し直した後に WireMock を再起動したか |
| 本文が空・ファイルがないと言われる | `bodyFile` のパス。`--fail-on-missing-body-file` を付けて build すると、置き忘れを build のときに見つけられる |
| 意図しないケースが返る | 複数のケースに一致している。条件の厳しいケースの `priority` を小さくする（警告「same priority」「same request conditions」も確認） |
| `file already exists` | init の書き出し先に case YAML がある。別のパスに書き出す（`--force` はレスポンス JSON も上書きする） |
| multipart / form-urlencoded の operation | 現在の版では、path・query・JSON 本文による返し分けだけに対応している |
