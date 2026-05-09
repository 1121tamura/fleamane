# CLAUDE.md

このファイルはClaude Codeがこのリポジトリで作業する際の指示書です。
必ず最初にこのファイルを読んでから作業を開始してください。

---

## プロジェクト概要

**Fleamane**（フリマネ）- フリマ売上管理アプリ
メルカリ・ヤフオク運営の自動化・効率化デスクトップアプリ。

| 項目 | 内容 |
|------|------|
| アプリ名 | Fleamane |
| リポジトリ | fleamane |
| バイナリ | fleamane |
| DB | fleamane.db |

詳細は `docs/` ディレクトリの設計書を参照すること。

```
docs/
├── 01_overview.md    # システム概要・アーキテクチャ
├── 02_features.md    # 機能要件定義
└── 03_db_schema.md   # DBスキーマ設計
```

---

## 基本方針

- **Chrome拡張のみ**：メルカリ・ヤフオクともに公式API一般提供なし・Chrome拡張で対応
- **config.yaml廃止**：全設定はSQLiteで管理・GUIから操作
- **React distはバイナリにembed**：配布物はfleamaneバイナリ＋fleamane.db＋chrome-extension/フォルダのみ
- **仕様不明時は必ず確認**：憶測で実装しない

---

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| バックエンド | Go 1.23+ |
| HTTPルーター | Chi |
| OpenAPI | ogen（API First） |
| フロントエンド | React 19 + TypeScript |
| UI | shadcn/ui + Tailwind CSS |
| ルーティング | TanStack Router |
| サーバー状態管理 | TanStack Query（APIデータ・キャッシュ・ローディング） |
| クライアント状態管理 | Zustand（UI状態・ログイン情報・WebSocket状態） |
| フォーム | React Hook Form + Zod |
| DB | SQLite（WALモード） |
| マイグレーション | golang-migrate |
| Chrome拡張 | Manifest V3 + TypeScript |

---

## アーキテクチャ原則

### DDD・クリーンアーキテクチャを厳守すること

```
handler/（Interface層）   ← Chiはここだけ
    ↓
usecase/（UseCase層）     ← フレームワーク依存禁止
    ↓
domain/（Domain層）       ← フレームワーク依存禁止・外部依存禁止
    ↑
repository/（Infrastructure層） ← DB・外部API
```

**禁止事項：**
- Domain層・UseCase層にフレームワーク固有の型を使わない
- Domain層にDB・外部APIへの直接依存を書かない
- handler層にビジネスロジックを書かない

### API Firstで開発すること

1. まず `api/openapi.yaml` を更新する
2. `ogen` でコード生成する
3. 生成されたインターフェースを実装する

```bash
# コード生成コマンド
go generate ./api/...
```

---

## コーディング規約

### Go

- エラーは必ず処理する（`_` で握りつぶさない）
- ロギングは `log/slog`（標準ライブラリ）を使う
- 環境変数・シークレットは**絶対にハードコードしない**
- インターフェースはconsumer側（usecase/）で定義する

```go
// Good
type ListingRepository interface {
    FindByID(ctx context.Context, id int64) (*domain.Listing, error)
}

// Bad - repositoryパッケージでインターフェース定義しない
```

### TypeScript（React）

- `any` 型の使用禁止
- コンポーネントはFunction Componentのみ
- API通信はTanStack Queryを使う（直接fetchしない）
- 状態管理はZustandを使う

### Chrome拡張

- Content Scriptからlocalhost APIへの通信は必ずエラーハンドリングする
- メルカリ・ヤフオクのページ未開時は明確なエラーメッセージを返す
- DOM取得は `try-catch` で必ず囲む（ページ構造変更対策）

---

## セキュリティ要件（必須）

- APIキー・シークレットは環境変数またはOSキーチェーンに保存
- ユーザーパスワードはbcryptでハッシュ化
- Chrome拡張とGoサービス間の通信はlocalhost限定
- SQLクエリはプレースホルダーを使う（SQLインジェクション防止）
- CORSはlocalhost origin限定に設定する

---

## ディレクトリ構成（重要）

新しいサービスを追加する場合は以下の構成に従うこと：

```
internal/services/{service_name}/
├── handler/
│   └── {resource}_handler.go
├── usecase/
│   ├── {resource}_usecase.go
│   └── {resource}_usecase_test.go
├── domain/
│   ├── {resource}.go          # エンティティ・値オブジェクト
│   └── repository.go          # repositoryインターフェース
└── repository/
    └── sqlite_{resource}_repository.go
```

---

## 開発コマンド

```bash
# 開発サーバー起動（全サービス）
./scripts/dev.sh

# フロントエンドのみ
cd frontend && npm run dev

# マイグレーション実行
migrate -path db/migrations -database "sqlite3://frima.db" up

# OpenAPIコード生成
go generate ./api/...

# ビルド（Mac）
./scripts/build.sh

# リリースビルド（Mac + Windows）
./scripts/release.sh
```

---

## 現フェーズのスコープ外機能

以下は設計書のロードマップに記載済み。現フェーズでは実装しないこと。

**Phase 2（近い将来）**
- F-01 販促コメント自動返信Bot
- F-03 予約出品システム
- F-06 日報LINE自動配信
- F-07 反応低調商品レポート

**Phase 3（AI連携）**
- F-02 商品タイトル・説明文生成（Claude API）
- F-05 フォロワークラスタ分析
- aiサービス（:8007）

**Phase 4（クラウド移行）**
- 24時間稼働対応
- Windows向けビルド

---

## 仕様が不明な場合

**憶測で実装しないこと。**
不明点は必ずユーザーに確認してから実装する。

特に以下は実機確認が必要：
- メルカリの各ページのDOM構造
- ヤフオクAPIのレスポンス形式
- 弥生会計のCSVフォーマット詳細
