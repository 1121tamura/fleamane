# DBスキーマ設計書

## 基本方針
- DB：SQLite（WALモード）
- マイグレーション：golang-migrate
- ファイルパス：config.yamlで指定（デフォルト：`./frima.db`）
- テーブルのプレフィックスでサービスを識別

---

## 0. アプリ設定（config.yaml廃止・DB管理）

### app_settings（アプリ設定）
```sql
CREATE TABLE app_settings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE,   -- 設定キー
    value       TEXT NOT NULL,          -- 設定値（JSON or 文字列）
    category    TEXT NOT NULL,
    -- 'service'|'external_api'|'accounting'|'app'
    description TEXT,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 初期データ（バイナリ起動時に自動投入）
INSERT INTO app_settings (key, value, category, description) VALUES
-- アプリ設定
('app.port',          '8080',  'app',          'メインサーバーポート'),
('app.auto_open_browser', 'true', 'app',        'ブラウザ自動起動'),

-- サービス有効化
('service.mercari.enabled',      'true',  'service', 'メルカリサービス'),
('service.mercari.port',         '8001',  'service', 'メルカリサービスポート'),
('service.yahooauction.enabled', 'true',  'service', 'ヤフオクサービス'),
('service.yahooauction.port',    '8002',  'service', 'ヤフオクサービスポート'),
('service.listing.enabled',      'true',  'service', '出品管理サービス'),
('service.listing.port',         '8003',  'service', '出品管理サービスポート'),
('service.notification.enabled', 'false', 'service', '通知サービス'),
('service.notification.port',    '8004',  'service', '通知サービスポート'),
('service.pricing.enabled',      'true',  'service', '価格管理サービス'),
('service.pricing.port',         '8005',  'service', '価格管理サービスポート'),
('service.accounting.enabled',   'false', 'service', '会計サービス'),
('service.accounting.port',      '8006',  'service', '会計サービスポート'),

-- 外部API（値は空・UIから設定・OSキーチェーンに保存）
('external.yahooauction.client_id',     '', 'external_api', 'ヤフオクAPI Client ID'),
('external.line.channel_token',         '', 'external_api', 'LINE Channel Token'),
('external.google_sheets.spreadsheet_id', '', 'external_api', 'GoogleスプレッドシートID'),

-- 会計設定
('accounting.format',     'yayoi',               'accounting', '会計フォーマット'),
('accounting.output_dir', '~/Downloads/frima',   'accounting', 'CSV出力先');
```

---

## 1. ユーザー・アカウント管理

### users（アプリユーザー）
```sql
CREATE TABLE users (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    email       TEXT UNIQUE NOT NULL,
    password    TEXT NOT NULL,  -- bcryptハッシュ
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### platform_accounts（フリマアカウント）
```sql
CREATE TABLE platform_accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    platform    TEXT NOT NULL,  -- 'mercari' | 'yahooauction'
    account_name TEXT NOT NULL, -- 表示用アカウント名
    is_active   INTEGER DEFAULT 1,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, platform, account_name)
);
```

---

## 2. 商品・出品データ

### listings（出品商品）
```sql
CREATE TABLE listings (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_account_id INTEGER NOT NULL REFERENCES platform_accounts(id),
    platform_item_id    TEXT NOT NULL,   -- プラットフォーム上のID
    platform            TEXT NOT NULL,   -- 'mercari' | 'yahooauction'
    title               TEXT NOT NULL,
    description         TEXT,
    category            TEXT,
    price               INTEGER NOT NULL,  -- 現在価格（円）
    cost_price          INTEGER,           -- 仕入れ原価（円）
    status              TEXT NOT NULL,     -- 'listed'|'sold'|'cancelled'|'draft'
    listed_at           DATETIME,          -- 出品日
    sold_at             DATETIME,          -- 売上日
    image_url           TEXT,
    item_url            TEXT,
    shipping_method     TEXT,              -- 発送方法
    shipping_payer      TEXT,              -- 'seller' | 'buyer'
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, platform_item_id)
);
```

### listing_status_history（ステータス履歴）
```sql
CREATE TABLE listing_status_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    listing_id  INTEGER NOT NULL REFERENCES listings(id),
    status      TEXT NOT NULL,
    changed_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 3. 売上・取引データ

### transactions（取引）
```sql
CREATE TABLE transactions (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    listing_id          INTEGER NOT NULL REFERENCES listings(id),
    platform_account_id INTEGER NOT NULL REFERENCES platform_accounts(id),
    platform_tx_id      TEXT NOT NULL,      -- プラットフォーム上の取引ID
    platform            TEXT NOT NULL,
    status              TEXT NOT NULL,
    -- 'purchased'|'waiting_shipment'|'shipped'|'completed'|'cancelled'
    selling_price       INTEGER NOT NULL,   -- 販売価格（円）
    platform_fee        INTEGER,            -- プラットフォーム手数料（円）
    shipping_cost       INTEGER,            -- 送料（円）
    net_income          INTEGER,            -- 実収入（円）
    purchased_at        DATETIME,           -- 購入日
    shipped_at          DATETIME,           -- 発送日
    completed_at        DATETIME,           -- 取引完了日
    cancelled_at        DATETIME,           -- キャンセル日
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, platform_tx_id)
);
```

---

## 4. ジョブ管理

### fetch_jobs（データ取得ジョブ）
```sql
CREATE TABLE fetch_jobs (
    id              TEXT PRIMARY KEY,       -- UUID
    user_id         INTEGER NOT NULL REFERENCES users(id),
    platform        TEXT NOT NULL,          -- 'mercari' | 'yahooauction'
    platform_account_id INTEGER REFERENCES platform_accounts(id),
    job_type        TEXT NOT NULL,          -- 'full' | 'diff'
    date_from       DATE,
    date_to         DATE,
    status          TEXT NOT NULL DEFAULT 'pending',
    -- 'pending'|'running'|'completed'|'error'|'cancelled'
    total_count     INTEGER,                -- 取得予定件数
    fetched_count   INTEGER DEFAULT 0,      -- 取得済み件数
    error_message   TEXT,
    checkpoint      TEXT,                   -- 中断時の再開ポイント（JSON）
    started_at      DATETIME,
    completed_at    DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 5. 価格・原価管理

### price_settings（価格設定）
```sql
CREATE TABLE price_settings (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    platform_account_id INTEGER NOT NULL REFERENCES platform_accounts(id),
    discount_rate       REAL NOT NULL DEFAULT 0.8,  -- 割引率（例：0.8 = 20%オフ）
    min_price           INTEGER DEFAULT 0,           -- 最低価格（円）
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### price_history（価格変更履歴）
```sql
CREATE TABLE price_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    listing_id  INTEGER NOT NULL REFERENCES listings(id),
    old_price   INTEGER NOT NULL,
    new_price   INTEGER NOT NULL,
    reason      TEXT,   -- 'manual'|'bulk_discount'|'promotional'
    changed_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 6. 通知・ログ

### notification_logs（通知ログ）
```sql
CREATE TABLE notification_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    type            TEXT NOT NULL,  -- 'daily_report'|'morning_report'|'promo_comment'
    platform        TEXT,
    status          TEXT NOT NULL,  -- 'success'|'error'
    message         TEXT,           -- 送信内容サマリ
    error_message   TEXT,
    sent_at         DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### system_logs（システムログ）
```sql
CREATE TABLE system_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    service     TEXT NOT NULL,      -- サービス名
    level       TEXT NOT NULL,      -- 'info'|'warn'|'error'
    message     TEXT NOT NULL,
    detail      TEXT,               -- JSON形式の詳細
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 7. 会計データ

### accounting_exports（会計エクスポート履歴）
```sql
CREATE TABLE accounting_exports (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL REFERENCES users(id),
    format          TEXT NOT NULL,      -- 'yayoi' | 'freee'
    date_from       DATE NOT NULL,
    date_to         DATE NOT NULL,
    file_path       TEXT NOT NULL,      -- 出力ファイルパス
    record_count    INTEGER NOT NULL,
    exported_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## インデックス

```sql
-- よく使う検索条件にインデックスを設定
CREATE INDEX idx_listings_platform_status ON listings(platform, status);
CREATE INDEX idx_listings_listed_at ON listings(listed_at);
CREATE INDEX idx_listings_sold_at ON listings(sold_at);
CREATE INDEX idx_transactions_completed_at ON transactions(completed_at);
CREATE INDEX idx_transactions_platform_account ON transactions(platform_account_id);
CREATE INDEX idx_fetch_jobs_status ON fetch_jobs(status);
CREATE INDEX idx_system_logs_created_at ON system_logs(created_at);
```

---

## 未確定事項

| 項目 | 状態 |
|------|------|
| メルカリのステータス値の実際のテキスト | ❌ DOM調査要 |
| ヤフオクのステータス値 | ❌ API仕様確認要 |
| 弥生CSV連携に必要な追加項目 | ❌ 弥生フォーマット要調査 |
| 仕入れ原価のGoogleスプレッドシート項目 | ❌ 確認要 |
