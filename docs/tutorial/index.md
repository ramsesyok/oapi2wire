# PetStore モックサーバ構築チュートリアル

このチュートリアルでは、PetStore の OpenAPI 定義から WireMock standalone で起動できるモックサーバ資産を作成します。

`oapi2wire` は WireMock を直接起動するツールではありません。OpenAPI 定義、case YAML、レスポンス JSON を入力として、WireMock が読み込む `mappings/` と `__files/` を生成するツールです。

```text
OpenAPI
  + case YAML
  + mock-responses/
        ↓
    oapi2wire
        ↓
wiremock-out/
  ├─ mappings/
  └─ __files/
        ↓
WireMock standalone
```

生成したモックサーバは、フロントエンドの動作確認だけでなく、runnora で作成したテストケースの接続先として利用できます。テストケースごとに期待するリクエスト条件とレスポンスを case YAML とレスポンス JSON に分けて管理できるため、OpenAPI を正本にしながらモックの返し分けを育てていけます。

## 前提条件

このチュートリアルでは、リポジトリルートから見て次のファイルがある前提で説明します。

```text
docs/
├─ tutorial/
│  ├─ index.md
│  └─ openapi.yaml
└─ tools/
   └─ wiremock-standalone-3.13.2.jar
```

必要なものは次のとおりです。

- `oapi2wire` コマンド
- Java
- PetStore OpenAPI 定義: `docs/tutorial/openapi.yaml`
- WireMock standalone: `docs/tools/wiremock-standalone-3.13.2.jar`

`docs/tools/wiremock-standalone-3.13.2.jar` は Git には登録しない前提です。チュートリアルを実行する環境ごとに配置してください。

`oapi2wire` をソースからビルドする場合は、リポジトリルートで次のように実行します。

```bash
go build -o oapi2wire .
```

以降のコマンド例では、`oapi2wire` にパスが通っている前提で記載します。ローカルでビルドしたバイナリを使う場合は、環境に合わせて `../../oapi2wire` などに読み替えてください。

## 作成するファイル

このチュートリアルでは、`docs/tutorial` 配下に次のファイルとディレクトリを作成します。

```text
docs/tutorial/
├─ openapi.yaml
├─ mock-cases.yaml
├─ mock-responses/
└─ wiremock-out/
   ├─ mappings/
   └─ __files/
```

それぞれの役割は次のとおりです。

| パス | 役割 |
|---|---|
| `openapi.yaml` | API の path、method、operationId、パラメータ、レスポンス定義 |
| `mock-cases.yaml` | operationId ごとの返し分け条件 |
| `mock-responses/` | WireMock から返すレスポンス JSON |
| `wiremock-out/mappings/` | WireMock の mapping JSON |
| `wiremock-out/__files/` | WireMock が参照するレスポンス body ファイル |

## 1. 作業ディレクトリへ移動する

リポジトリルートから `docs/tutorial` に移動します。

```bash
cd docs/tutorial
```

以降のコマンドは `docs/tutorial` をカレントディレクトリとして実行します。

## 2. OpenAPI から雛形を生成する

まず、PetStore OpenAPI 定義から case YAML とレスポンス JSON の雛形を生成します。

```bash
oapi2wire init \
  --openapi ./openapi.yaml \
  --out-cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

実行すると、OpenAPI 内の各 `operationId` に対して、`mock-cases.yaml` にケース雛形が作成されます。あわせて、各ケースが参照するレスポンス JSON の雛形が `mock-responses/` に作成されます。

PetStore の OpenAPI には、たとえば次のような operationId が含まれます。

| operationId | 内容 | チュートリアルで見る観点 |
|---|---|---|
| `getPetById` | ペット ID でペットを取得する | path parameter |
| `findPetsByStatus` | ステータスでペット一覧を検索する | query parameter |
| `placeOrder` | 注文を作成する | JSON request body |
| `loginUser` | ユーザー名とパスワードでログインする | 複数 query parameter |

PetStore には multipart/form-data や form-urlencoded の操作も含まれますが、oapi2wire の MVP では主に path parameter、query parameter、JSON request body による返し分けを扱います。このチュートリアルでも、まずはその範囲に絞ります。

## 3. case YAML を編集する

生成された `mock-cases.yaml` を編集して、リクエスト条件とレスポンスを定義します。

ここでは、代表的な 3 種類の返し分けを作ります。

- `getPetById`: path parameter で返し分ける
- `findPetsByStatus`: query parameter で返し分ける
- `placeOrder`: JSON request body で返し分ける

### path parameter で返し分ける

`GET /pet/{petId}` に対して、`petId` が `100` のときだけ一致するケースを作ります。

```yaml
- id: getPetById_100
  operationId: getPetById
  priority: 10
  request:
    pathParams:
      petId:
        equalTo: "100"
  response:
    status: 200
    bodyFile: getPetById/getPetById_100.json
```

`bodyFile` は `mock-responses/` からの相対パスです。絶対パスや `..` を含むパスは使えません。

### query parameter で返し分ける

`GET /pet/findByStatus?status=available` に一致するケースを作ります。

```yaml
- id: findPetsByStatus_available
  operationId: findPetsByStatus
  priority: 10
  request:
    query:
      status:
        equalTo: "available"
  response:
    status: 200
    bodyFile: findPetsByStatus/findPetsByStatus_available.json
```

正規表現で一致させたい場合は `matches` を使えます。

```yaml
status:
  matches: "available|pending|sold"
```

### JSON request body で返し分ける

`POST /store/order` に対して、リクエスト body の内容で返し分けるケースを作ります。

```yaml
- id: placeOrder_pet_100
  operationId: placeOrder
  priority: 10
  request:
    body:
      equalToJson:
        petId: 100
        quantity: 1
        status: placed
        complete: false
  response:
    status: 200
    bodyFile: placeOrder/placeOrder_pet_100.json
```

JSONPath で一部の値だけを見たい場合は `matchesJsonPath` を使えます。

```yaml
request:
  body:
    matchesJsonPath:
      - "$.petId[?(@ == 100)]"
      - "$.status[?(@ == 'placed')]"
```

`equalToJson` と `matchesJsonPath` は同時に指定できます。

## 4. レスポンス JSON を編集する

次に、各ケースの `bodyFile` が参照するレスポンス JSON を編集します。

たとえば `getPetById/getPetById_100.json` は、`mock-responses/` 配下に作成します。

```json
{
  "id": 100,
  "name": "doggie",
  "status": "available",
  "category": {
    "id": 1,
    "name": "dogs"
  },
  "photoUrls": [
    "https://example.com/doggie.png"
  ],
  "tags": [
    {
      "id": 10,
      "name": "friendly"
    }
  ]
}
```

runnora のテストケース確認に使う場合は、テストケースが期待する HTTP status、レスポンス header、レスポンス body と、この case YAML / response JSON を対応させます。

たとえば、runnora のテストケースで「`GET /pet/100` は `status=available` のペットを返す」と期待しているなら、次の対応になります。

| runnora 側の観点 | oapi2wire 側で編集する場所 |
|---|---|
| リクエスト path | `operationId` と `request.pathParams` |
| リクエスト query | `request.query` |
| リクエスト JSON body | `request.body` |
| 期待 HTTP status | `response.status` |
| 期待レスポンス body | `mock-responses/` 配下の JSON |

## 5. 整合性を検証する

WireMock 用ファイルを生成する前に、OpenAPI と case YAML の整合性を確認します。

```bash
oapi2wire validate \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses
```

`operationId` の指定ミス、重複した case ID、禁止された `bodyFile` パスなどがある場合は、この段階で検出できます。

レスポンス JSON の存在も厳しく確認したい場合は、`build` 時に `--fail-on-missing-body-file` を付けます。

## 6. WireMock 用ファイルを生成する

case YAML とレスポンス JSON をもとに、WireMock が読み込む `mappings/` と `__files/` を生成します。

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out \
  --clean \
  --fail-on-missing-body-file
```

生成後の構成は次のようになります。

```text
wiremock-out/
├─ mappings/
│  ├─ findPetsByStatus__findPetsByStatus_available.json
│  ├─ getPetById__getPetById_100.json
│  ├─ placeOrder__placeOrder_pet_100.json
│  └─ _generated__fallback__*.json
└─ __files/
   ├─ findPetsByStatus/
   ├─ getPetById/
   ├─ placeOrder/
   └─ _generated/
```

明示的な fallback ケースを定義していない operationId には、自動 fallback が生成されます。リクエストがどのケースにも一致しない場合、fallback のレスポンスが返ります。

自動 fallback を生成したくない場合は、`--no-auto-fallback` を指定します。

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out \
  --no-auto-fallback
```

## 7. WireMock standalone を起動する

`wiremock-out` を root directory として WireMock standalone を起動します。

`docs/tutorial` で実行している場合、jar への相対パスは `../tools/wiremock-standalone-3.13.2.jar` です。

```bash
java -jar ../tools/wiremock-standalone-3.13.2.jar \
  --root-dir ./wiremock-out \
  --port 8080
```

起動後、WireMock は `wiremock-out/mappings/` の mapping JSON と `wiremock-out/__files/` の body file を読み込んでリクエストを待ち受けます。

## 8. curl で動作確認する

別のターミナルから、モックサーバへリクエストを送ります。

### path parameter の確認

```bash
curl -i http://localhost:8080/pet/100
```

`getPetById_100` に一致すると、`mock-responses/getPetById/getPetById_100.json` の内容が返ります。

### query parameter の確認

```bash
curl -i "http://localhost:8080/pet/findByStatus?status=available"
```

`findPetsByStatus_available` に一致すると、`mock-responses/findPetsByStatus/findPetsByStatus_available.json` の内容が返ります。

### JSON request body の確認

```bash
curl -i \
  -X POST http://localhost:8080/store/order \
  -H "Content-Type: application/json" \
  -d '{"petId":100,"quantity":1,"status":"placed","complete":false}'
```

`placeOrder_pet_100` に一致すると、`mock-responses/placeOrder/placeOrder_pet_100.json` の内容が返ります。

### fallback の確認

一致しない条件でリクエストすると、自動生成された fallback が返ります。

```bash
curl -i http://localhost:8080/pet/999999
```

fallback が返った場合は、リクエスト条件が case YAML の matcher と一致していない可能性があります。

## 9. runnora から利用する

runnora で作成したテストケースの接続先を WireMock に向けると、OpenAPI から生成したモックサーバに対してテストケースを確認できます。

```text
base URL: http://localhost:8080
```

確認の流れは次のとおりです。

1. runnora のテストケースで送信する path、query、JSON body を確認する
2. `mock-cases.yaml` の `request` に同じ条件を定義する
3. `response.status` と `bodyFile` を、期待したいレスポンスに合わせる
4. `mock-responses/` の JSON を編集する
5. `oapi2wire build --clean` で WireMock 用ファイルを再生成する
6. WireMock を再起動して、runnora から実行する

テストケースが fallback に落ちる場合は、次の点を確認します。

- `operationId` が対象の OpenAPI operation と一致しているか
- path parameter の値が `request.pathParams` と一致しているか
- query parameter の名前と値が `request.query` と一致しているか
- JSON body が `equalToJson` または `matchesJsonPath` と一致しているか
- `bodyFile` が `mock-responses/` 配下に存在しているか

## トラブルシューティング

### WireMock standalone jar が見つからない

`docs/tools/wiremock-standalone-3.13.2.jar` があるか確認してください。

`docs/tutorial` から起動する場合、コマンド内の jar パスは `../tools/wiremock-standalone-3.13.2.jar` です。

### Java が見つからない

次のコマンドで Java が実行できるか確認してください。

```bash
java -version
```

### bodyFile が見つからない

`response.bodyFile` は `--responses-root` からの相対パスです。

```yaml
response:
  bodyFile: getPetById/getPetById_100.json
```

この場合、実ファイルは次の場所に必要です。

```text
mock-responses/getPetById/getPetById_100.json
```

`--fail-on-missing-body-file` を付けて `build` すると、ファイル不足を生成時に検出できます。

### matcher が一致しない

fallback が返る場合は、WireMock が通常ケースに一致していません。

よくある原因は次のとおりです。

- path parameter を数値のつもりで書いているが、matcher では文字列として比較している
- query parameter の名前が OpenAPI と case YAML でずれている
- JSON body のフィールドが足りない
- `equalToJson` に指定した値とリクエスト body の値が違う

まずは `matchesJsonPath` を使って、必要なフィールドだけを見る形にすると確認しやすくなります。

### multipart/form-data や form-urlencoded の操作を使っている

PetStore の OpenAPI には `uploadFile` や `updatePetWithForm` など、multipart/form-data や form-urlencoded を使う operation があります。

oapi2wire の MVP では、主に path parameter、query parameter、JSON request body を対象にしています。最初のチュートリアルでは、`getPetById`、`findPetsByStatus`、`placeOrder` のような操作から確認してください。

## 再生成する

case YAML やレスポンス JSON を編集したら、`build --clean` で WireMock 用ファイルを再生成します。

```bash
oapi2wire build \
  --openapi ./openapi.yaml \
  --cases ./mock-cases.yaml \
  --responses-root ./mock-responses \
  --out ./wiremock-out \
  --clean \
  --fail-on-missing-body-file
```

WireMock がすでに起動している場合は、再生成後に WireMock を再起動してください。

## Git 管理の目安

チュートリアルの元になるファイルは Git に登録します。

- `docs/tutorial/index.md`
- `docs/tutorial/openapi.yaml`

チュートリアルを実行して生成されるファイルは、運用方針に応じて扱います。

- `docs/tutorial/mock-cases.yaml`
- `docs/tutorial/mock-responses/`
- `docs/tutorial/wiremock-out/`

`mock-cases.yaml` と `mock-responses/` は、チームで共有したいモック定義として Git 管理してもよいです。`wiremock-out/` は `build` で再生成できるため、通常は Git 管理しない運用が扱いやすいです。

`docs/tools/wiremock-standalone-3.13.2.jar` は Git に登録しない前提です。
