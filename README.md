# rusudenkun

AsteriskのARIを利用した留守電文字起しSlack通知ツール

## 概要

`rusudenkun`は、AsteriskのARIを利用して留守電メッセージを取得し、音声認識を行い、その結果をSlackに通知するツールです。  
さくらのAI Engineを使った文字起しを試してみたいので作ってみました。

雑に作ってるので、お遊び程度にどうぞ。

## Asteriskの設定

ARIが使えるようにする

* ari.conf
  ```
  [general]
  enabled = yes
  
  [asterisk]
  type = user
  read_only = no
  password = asterisk
  password_format = plain
  ```

* http.conf
  ```
  [general]
  servername=Asterisk
  enabled=yes
  bindaddr=127.0.0.1
  bindport=8088
  ```

* extensions.conf (試しに999にTELしたら実行される)
  ```
  [default]
  exten => 999,1,Wait(1)
    same => n,Stasis(rusudenkun)
    same => n,Hangup()
  ```

* 留守電応答メッセージの配置
  ```
  /usr/share/asterisk/sounds/custom/rusuden.ulaw あたりに音声ファイルを配置してください。
  MessageSound: "custom/rusuden" で呼び出します。
  ```
