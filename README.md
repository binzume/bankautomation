
ネットバンクの残高調整用.

Googleスプレッドシートに記帳したり，Slackに通知したり，口座に送金したりします．

色々雑です．他の方がそのまま使うことは想定していません．

察してください．

# 設定

config.json

- google_credential: スプレッドシートに書き込むためのアカウント情報
- slack_url: Incoming WebhookのURL
- items: 色々
- item.login: 参考: https://github.com/binzume/go-banking/tree/master/examples

## action

### type:

- balance: 残高判定
- history: 取引履歴に特定文字列が現れるかチェック
- error: エラー時に実行
- always: 常に実行

### op, threshold, match:

条件指定

balance_item が指定されている場合は，そのitemのbalanceの値が使われます．(特定口座の残高が閾値を切ったときに補充するためのもの)

### interval (hours)

実行間隔の最低時間．省略時や interval:0 の場合は毎回．

### 実行内容:

- trans: 送金します (password2 に 送金用パスワードを記入)
- slack: slackにメッセージを書き込みます (slack_url に Incoming WebhookのURLを書いてください)
- log: ログを出しますデバッグ用

## 例

- test01 の残高が 2000000 以上の，場合は testtest2 宛に送金．500000円 単位で 1500000円を下回らないように送金します.送金の上限は3000000円.
- test01 の残高が 1000000 以下になると，slackに通知＆ログにも同様のメッセージを残す
- test01 の残高が 1500000 以下になると, test02 から testtest 宛に500000円送金します． (testtest = test01の登録名)

``` json
{
  "google_credential": "accounts/test-test-abcdef0123456.json",
  "slack_url":"https://hooks.slack.com/services/XXXXXXXXXXXXXXXXXXXXXXXX",
  "items":[
    {
      "name": "test01",
      "login": "accounts/stub1.json",
      "save_status": "./status_test.json",
      "spreadsheet": "1j8bUzrOm0po9xMqLymdMS4y2-jfEoTi-2DF1o2p61Eo:履歴",
      "password2": "hoge",
      "actions": [
         {"type": "balance", "op": ">", "threshold": 2000000, "interval":240, "trans": {"target": "testtest2", "base":1500000, "unit":500000, "limit":3000000}},
         {"type": "balance", "op": "<", "threshold": 1000000, "interval":24, "slack": {"message": "test < 1000000"}, "log": {"message": "test < 1000000"}}
        ]
    },
    {
      "name": "test02",
      "login": "accounts/stub2.json",
      "save_status": "./status_test2.json",
      "password2": "hoge",
      "actions": [
         {"type": "balance", "op": "<", "threshold": 1500000, "interval":240, "balance_item":"test01", "trans": {"target": "testtest", "amount":500000, "limit":1000000}}
        ]
    }
  ]
}
```
