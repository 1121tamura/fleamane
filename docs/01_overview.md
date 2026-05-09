# Fleamane 設計書

---

## 1. システム概要

### アプリ情報
| 項目 | 内容 |
|------|------|
| アプリ表示名 | **Fleamane** |
| リポジトリ名 | **fleamane** |
| バイナリ名 | **fleamane** |
| DBファイル名 | **fleamane.db** |

### 目的
メルカリ・ヤフオク（ヤフフリ）の運営業務を自動化・効率化するデスクトップアプリケーション。

### 対象ユーザー
- 主ユーザー：フリマ運営者（非エンジニア）
- 利用形態：1PC複数人（夫婦共有等）を想定
- アプリ内ユーザーログイン機能あり

### 基本方針
- **スクレイピング禁止**：Chrome拡張による合法的なデータ取得のみ（公式APIは両プラットフォームとも一般向け提供なし）
- **セキュリティ重視**：APIキー・認証情報の安全な管理
- **ユーザー操作最小化**：ダブルクリックで起動・設定はGUIで完結
- **オプション式サービス**：DB上の設定で必要なサービスのみ有効化
- **config.yaml廃止**：全設定はSQLite（DB）で管理・GUIから操作

---

## 2. 技術スタック

| レイヤー | 技術 | 備考 |
|---------|------|------|
| バックエンド言語 | Go 1.23+ | 単一バイナリ配布 |
| HTTPルーティング | Chi | 標準net/http互換 |
| OpenAPI | ogen（API First） | openapi.yamlからコード生成 |
| フロントエンド | React 19 + TypeScript | ローカルWebアプリ |
| UIコンポーネント | shadcn/ui + Tailwind CSS | 最もモダン・カスタマイズ性高 |
| ルーティング | TanStack Router | 型安全・SPA向き |
| サーバー状態管理 | TanStack Query | APIデータ・キャッシュ・ローディング管理 |
| クライアント状態管理 | Zustand | UI状態・ログイン情報・WebSocket状態 |
| フォーム | React Hook Form + Zod | バリデーション込み |
| DB | SQLite（WALモード） | |
| マイグレーション | golang-migrate | |
| Chrome拡張 | Manifest V3 + TypeScript | |
| サービス間通信 | REST API + WebSocket | |
| 開発環境 | devcontainer（単一） | Go + Node混在 |

---

## 3. アーキテクチャ

### 全体構成

```
ユーザー
    ↓ ダブルクリック
Goメインプロセス（単一バイナリ）
    ↓ DBから設定読み込み（初回はデフォルト設定でDB自動生成）
    ↓ 有効サービスを子プロセスとして起動
    ↓ localhost:8080でReactを配信（embed済み）
    ↓ ブラウザを自動起動
React UI（ブラウザ）
    ↓ REST API / WebSocket
Goマイクロサービス群（:8001〜:8006）
    ↓ localhost REST API
Chrome拡張（Manifest V3）
    ↓
メルカリ・ヤフオクページ操作
    ↓
外部API（ヤフオクAPI・LINE・Google Sheets）
```

### サービス起動方式（サブコマンド方式）

```bash
# メインプロセスが自分自身をサブコマンドで子プロセス起動
./frima serve --service=mercari       # :8001
./frima serve --service=yahooauction  # :8002
./frima serve --service=listing       # :8003
./frima serve --service=notification  # :8004
./frima serve --service=pricing       # :8005
./frima serve --service=accounting    # :8006
```

### DDDレイヤー構成（各サービス共通）

```
Interface層（handler/）     ← Chiが関与するのはここのみ
    ↓
UseCase層（usecase/）       ← フレームワーク依存ゼロ
    ↓
Domain層（domain/）         ← フレームワーク依存ゼロ
    ↑
Infrastructure層（repository/） ← DB・外部API
```

---

## 4. プロジェクト構成

```
{{APP_NAME}}/
├── cmd/
│   └── frima/
│       └── main.go                  # エントリーポイント・サブコマンド管理
│
├── internal/
│   ├── launcher/                    # サービス起動・終了管理
│   │   └── launcher.go
│   ├── server/                      # メインHTTPサーバー（React配信）
│   │   └── server.go
│   ├── browser/                     # ブラウザ自動起動
│   │   └── browser.go
│   ├── shared/                      # 共通コード
│   │   ├── db/                      # SQLite接続・マイグレーション
│   │   ├── middleware/              # 認証・ロギング・CORS
│   │   └── models/                  # 共通モデル
│   │
│   └── services/                    # マイクロサービス群
│       ├── mercari/                 # :8001
│       │   ├── handler/
│       │   ├── usecase/
│       │   ├── domain/
│       │   └── repository/
│       ├── yahooauction/            # :8002
│       │   ├── handler/
│       │   ├── usecase/
│       │   ├── domain/
│       │   └── repository/
│       ├── listing/                 # :8003
│       │   └── ...
│       ├── notification/            # :8004
│       │   └── ...
│       ├── pricing/                 # :8005
│       │   └── ...
│       └── accounting/              # :8006
│           └── ...
│
├── api/
│   └── openapi.yaml                 # OpenAPI仕様書（API First）
│
├── db/
│   ├── migrations/                  # マイグレーションファイル
│   └── frima.db                     # SQLiteファイル（.gitignore対象）
│
├── frontend/                        # React 19 フロントエンド
│   ├── src/
│   │   ├── components/
│   │   │   ├── ui/                  # shadcn/uiコンポーネント
│   │   │   └── common/              # 共通コンポーネント
│   │   ├── pages/                   # ページコンポーネント
│   │   │   ├── Dashboard/
│   │   │   ├── Mercari/
│   │   │   ├── YahooAuction/
│   │   │   ├── Listing/
│   │   │   ├── Pricing/
│   │   │   ├── Accounting/
│   │   │   └── Settings/
│   │   ├── hooks/                   # カスタムフック
│   │   ├── stores/                  # Zustand（クライアント状態）
│   │   │   ├── authStore.ts         # ログイン・ユーザー情報
│   │   │   ├── uiStore.ts           # サイドバー・モーダル等
│   │   │   └── wsStore.ts           # WebSocket接続状態
│   │   ├── queries/                 # TanStack Query（サーバー状態）
│   │   │   ├── mercari.ts           # 出品・売上データ
│   │   │   ├── yahooauction.ts      # ヤフオクデータ
│   │   │   ├── jobs.ts              # 取得ジョブ進捗
│   │   │   └── settings.ts          # 設定データ
│   │   ├── api/                     # Axiosクライアント定義
│   │   ├── router/                  # TanStack Router設定
│   │   ├── schemas/                 # Zodバリデーションスキーマ
│   │   │   ├── fetchForm.ts         # データ取得フォーム
│   │   │   ├── discountForm.ts      # 値下げフォーム
│   │   │   └── settingsForm.ts      # 設定フォーム
│   │   └── types/                   # TypeScript型定義
│   └── package.json
│
├── chrome-extension/                # Chrome拡張
│   ├── manifest.json
│   └── src/
│       ├── background/              # Service Worker
│       ├── content/
│       │   ├── mercari.ts           # メルカリページ操作
│       │   └── yahoo.ts             # ヤフオクページ操作
│       └── types/
│
├── scripts/
│   ├── build.sh                     # 全体ビルド
│   ├── dev.sh                       # 開発用起動
│   └── release.sh                   # Mac/Winリリースビルド
│
├── .devcontainer/
│   └── devcontainer.json            # Go + Node単一devcontainer
│
├── go.mod
├── go.sum
├── .gitignore
└── README.md
```

---

## 5. 配布物

```
fleamane-mac/
├── fleamane              # バイナリ1つ（全サービス + React dist embed）
├── fleamane.db           # 初回起動時に自動生成
└── chrome-extension/     # Chrome拡張フォルダ（野良インストール用）
    ├── manifest.json
    └── ...
```

**ユーザーの操作：**
1. `fleamane` をダブルクリックして起動
2. 初回セットアップ画面の案内に従い `chrome-extension/` フォルダを読み込む

### バイナリサイズ目安

| 内容 | サイズ |
|------|--------|
| Goコード（全サービス） | 約15〜20MB |
| React dist（embed） | 約2〜5MB |
| **合計** | **約17〜25MB** |

---

## 6. 設定管理（DB管理）

**config.yamlは廃止。全設定はSQLiteで管理。**

設定はGUIの設定画面から変更する。

### 設定カテゴリ

| カテゴリ | 内容 | 画面 |
|---------|------|------|
| サービス有効化 | 各サービスのON/OFF・ポート番号 | 設定画面 |
| 外部API設定 | ヤフオクAPI・LINE・Google Sheets | 設定画面 |
| 会計設定 | 出力フォーマット・出力先 | 設定画面 |
| ユーザー設定 | アプリログイン情報 | 設定画面 |

詳細はDBスキーマの`app_settings`テーブルを参照。

---

## 7. セキュリティ方針

- APIキー・シークレットは**環境変数またはOSキーチェーン**に保存（コードへのハードコード禁止）
- アプリユーザーのパスワードは**bcryptハッシュ化**してSQLiteに保存
- Chrome拡張とGoサービス間の通信は**localhost限定 + リクエストトークン**で保護
- 外部APIキーはconfig.yamlに平文で書かず、**初回起動時にUIから設定**させる

---

## 8. 開発環境（devcontainer）

### 構成
- 単一devcontainer（Go + Node混在）
- VS Code + devcontainer拡張

### 必要ツール
- Go 1.23+
- Node.js 20+
- golang-migrate CLI
- ogen CLI
- air（Goホットリロード）

---

## 9. 将来拡張ロードマップ

### Phase 1（現在スコープ）

| 機能 | メルカリ | ヤフオク | 方式 |
|------|---------|---------|------|
| データ取得（出品・売上） | ✅ | ✅ | Chrome拡張 |
| 差分取得 | ✅ | ✅ | Chrome拡張 |
| 一括値下げ | ✅ | ✅ | Chrome拡張 |
| 弥生会計CSV出力 | ✅ | ✅ | accountingサービス |

**Chrome拡張の動作前提：**
- Chromeでメルカリとヤフオクをそれぞれ別タブでログイン済みの状態で開いておく
- Chrome拡張は野良インストール（配布フォルダをデベロッパーモードで読み込む）

### Phase 2（近い将来）
- 予約出品システム（F-03）
- 販促コメント自動返信Bot（F-01）
- 日報LINE自動配信（F-06）
- 反応低調商品レポート（F-07）

### Phase 3（AI連携）

**aiサービス（:8007）を追加するだけで対応可能。設計変更不要。**

```
internal/services/ai/
    ├── handler/
    ├── usecase/
    ├── domain/
    └── repository/
```

| 機能 | 内容 |
|------|------|
| 商品タイトル・説明文生成 | 写真→Claude Vision APIで自動生成（F-02） |
| 採寸表OCR | 写真内の採寸表を読み取り説明文に含める |
| 参考価格提案 | 類似商品の相場をAIが提案 |
| 自然言語操作 | 「売れてない商品値下げして」等 |
| フォロワー分析 | 競合出品者の傾向分析（F-05） |

**DB設定追加（Phase 3実装時）：**
```sql
('service.ai.enabled',      'false', 'service',      'AIサービス'),
('service.ai.port',         '8007',  'service',      'AIサービスポート'),
('external.claude.api_key', '',      'external_api', 'Claude APIキー'),
('external.openai.api_key', '',      'external_api', 'OpenAI APIキー'),
```

### Phase 4（クラウド移行）
- マイクロサービスを個別にクラウドへ移行
- 24時間稼働（PC起動不要）
- 現在の設計のままクラウド対応可能

---

## 10. 未決定事項（TODO）

| 項目 | 状態 |
|------|------|
| リポジトリ名・アプリ表示名 | ❌ 未定 |
| メルカリページのDOM構造調査 | ❌ 実機確認要 |
| ヤフオクAPI Developer登録 | ❌ 未実施 |
| LINE Messaging API登録 | ❌ 未実施 |
| Google Sheets API設定 | ❌ 未実施 |
| 弥生CSVフォーマット詳細 | ❌ 要調査 |
| 分析サービス（F-05） | 🔵 Phase 3スコープ |
