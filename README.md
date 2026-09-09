[ 🇯🇵 **日本語** | 🇰🇷 [한국어](README.ko.md) ]

# KRNIC GeoIP to Binary Database Converter (`geoip_krnic2dbip`)

KRNIC（韓国インターネット情報センター）が公開している国別IPv4割り当て統計データを、超高速な検索が可能な**固定10バイト・バイナリフォーマット（`.dat`）**およびDB-IP互換フォーマットに変換し、自動配信するGo言語ベースのツールです。

GitHub Actionsを通じて毎週月曜日に最新のKRNICデータを自動ダウンロード・ビルドし、整合性検証を経てGitHub Release Assets（`global.dat`, `jp.dat`, `kr.dat`, `cn.dat`, `dbip-country-lite.csv`, `checksum.sha256`）として自動配信されます。

---

## 1. 主な特徴

- **超高速二分探索（Binary Search）：** 固定10バイトの固定長レコード構造により、メモリ上に巨大なインデックスを展開することなく、`fseek`による $O(\log N)$ 二分探索で1回の検索あたり**0.0005ms（約420ns）**以内で国コードを特定可能です。
- **連続IP帯域の自動マージ（Range Merging）：** 同一国の連続・隣接するIP帯域を自動的に結合し、レコード総数を約45%圧縮（約26万件 $\to$ 約14万件）して軽量化します。
- **ワンパス一括ビルド（`-all`）：** KRNIC CSVを1回走査するだけで、全世界（`global.dat`）、日本特化（`jp.dat`）、韓国特化（`kr.dat`）、中国特化（`cn.dat`）、`dbip-country-lite.csv`、およびSHA-256ハッシュ一覧（`checksum.sha256`）を同時に生成します。
- **動的な複数国コード抽出：** パラメータ（`-country JP,KR,CN`など）で指定された任意の国コード群を1パスで抽出し、個別のバイナリファイルとして出力します。
- **既存シェルスクリプトおよびxtablesとの完全互換：** 引数なしで実行した場合、従来の動作（`ipv4.csv` $\to$ `dbip-country-lite.csv`変換）をそのまま維持します。
- **検証（Verification）モード内蔵：** CLI自体に二分探索検索エンジンを内蔵しており、生成されたバイナリからのIP検索検証を即座に実行できます。
- **100%自動配信パイプライン：** 毎週月曜日 00:00 UTCにGitHub Actionsが起動し、常に最新のIPデータベースをリリース配信します。

---

## 2. バイナリレコード仕様（Fixed 10-Byte Big-Endian）

すべての`.dat`ファイルはヘッダーのない純粋なバイナリストリームであり、1レコードあたり**正確に10バイト**の固定長です。  
ファイルサイズは常に**10の倍数**であり、総レコード数は `ファイルサイズ(bytes) / 10` で即座に算出できます。

### レコード構造

| オフセット (Offset) | フィールド名 | データ型 | バイト順 (Endian) | 説明 |
|---|---|---|---|---|
| `[0 : 4]` | `FirstNum` | `uint32` (4 bytes) | Big-Endian | 帯域開始 IPv4 整数値 |
| `[4 : 8]` | `LastNum` | `uint32` (4 bytes) | Big-Endian | 帯域終了 IPv4 整数値 |
| `[8 : 10]` | `CountryCode` | `char[2]` (2 bytes) | ASCII | ISO 3166-1 alpha-2 国コード2文字（`JP`, `KR`, `US`等） |

### ソート保証
すべてのレコードは **`FirstNum` の昇順**で完全に整列されて書き込まれるため、ファイルへのランダムアクセスによる二分探索を直接適用できます。

---

## 3. インストール & ビルド

### 必要環境
- Go 1.22 以上

### ソースコードからのビルド
```bash
git clone https://github.com/elmitash/geoip_krnic2dbip.git
cd geoip_krnic2dbip
go build -o geoip_krnic2dbip ./cmd
```

### ユニットテスト & ベンチマーク実行
```bash
go test -v -bench=. ./cmd/...
```

---

## 4. CLIの使い方

### 1) 従来互換CSV変換モード（既存シェルおよびxtables互換）
従来の`geoip.sh`スクリプトおよび`xt_geoip_build`と完全互換です。引数なしで実行するとカレントディレクトリの`ipv4.csv`を読み込み、即座に`dbip-country-lite.csv`を生成します。
```bash
# 引数なしで実行した場合：自動的に dbip-country-lite.csv を生成（従来通り）
./geoip_krnic2dbip

# 出力先CSV名を指定する場合
./geoip_krnic2dbip -csv custom-country-lite.csv
```

### 2) ワンパス一括ビルド（バイナリ + CSV 同時生成、推奨）
KRNIC CSV（`ipv4.csv`）を1回パシングし、`global.dat`、指定国バイナリ、`checksum.sha256`、および`dbip-country-lite.csv`を一括生成します。
```bash
# 基本モード: global.dat + jp.dat, kr.dat, cn.dat + dbip-country-lite.csv + checksum.sha256 生成
./geoip_krnic2dbip -all

# カスタム国指定モード: global.dat + us.dat, gb.dat, de.dat + dbip-country-lite.csv 生成
./geoip_krnic2dbip -all -country US,GB,DE
```
- オプション:
  - `-country`（または `-countries`）：抽出対象の国コード2文字（カンマ区切りで複数指定可能、デフォルト：`JP,KR,CN`）
  - `-in <パス>`：入力KRNIC CSVのファイルパス（デフォルト：`ipv4.csv`）
  - `-csv <パス>`：DB-IP互換CSVの出力先パス（デフォルト：`dbip-country-lite.csv`）

### 3) 特定の国コードのみ抽出ビルド（複数国の一括分割抽出）
パラメータで国コードを指定すると、該当する国のレコードのみを個別のバイナリファイルとして抽出します。
```bash
# 複数国を個別の .dat ファイルとして一括抽出（結果: jp.dat, kr.dat, cn.dat）
./geoip_krnic2dbip -country JP,KR,CN

# 欧米諸国のみを個別に抽出（結果: us.dat, gb.dat, de.dat, fr.dat）
./geoip_krnic2dbip -country US,GB,DE,FR

# 単一国指定 & 出力ファイル名のカスタム指定
./geoip_krnic2dbip -country JP -out custom_japan.dat
```

### 4) バイナリ検索・検証モード（Verification）
生成された`.dat`ファイルに対し、任意のIPアドレスを二分探索で検索し、所属国と検索時間を即座に検証します。
```bash
# Yahoo Japan IP の検証 -> JP に一致
./geoip_krnic2dbip -verify jp.dat -ip 182.22.59.229

# 韓国 Naver IP の検証 -> KR に一致
./geoip_krnic2dbip -verify kr.dat -ip 211.249.220.24

# 中国 Baidu IP の検証 -> CN に一致
./geoip_krnic2dbip -verify cn.dat -ip 220.181.38.148

# Google Public DNS の検証 -> US に一致
./geoip_krnic2dbip -verify global.dat -ip 8.8.8.8
```

#### 検証出力例：
```text
Verifying IP 182.22.59.229 in binary file: jp.dat ...
Result: MATCH FOUND!
  - Country   : JP
  - IP Range  : 182.20.0.0 - 182.22.255.255
  - Range (u32): 3054764032 - 3054960639
  - Lookup Time: 31.72µs
```

### 5) 実データ一括検証スイート（`verify_suite.py`）
OECD加盟10カ国（日本、韓国、米国、英国、ドイツ、フランス、豪州、カナダ、イタリア、スペイン各10件）および各大陸の小規模国家10カ国（モナコ、リヒテンシュタイン、アイスランド、ブルネイ、ブータン、モルディブ、セーシェル、モーリシャス、ベリーズ、フィジー）の**計130件**の実在IPアドレスを一括検証します。
```bash
# 全130件のIPを一括検証 (global.dat)
python3 verify_suite.py global.dat

# 特定の国ファイルのみを検証 (例: jp.dat に対して日本のサイト10件のみ検証)
python3 verify_suite.py jp.dat JP
python3 verify_suite.py kr.dat KR
```

---

## 5. 生成アセット & ファイル一覧

| ファイル名 | 対象地域 | 想定レコード数 | 想定ファイル容量 | 主な用途・特徴 |
|---|---|---|---|---|
| `global.dat` | 全世界 | 約 142,000件 | 約 1.4 MB | 全世界IP判定および地理的アクセス制御 |
| `jp.dat` | 日本（JP） | 約 2,500件 | 約 25 KB | **日本国内専用サービス・ECサイト向け超軽量DB** |
| `kr.dat` | 韓国（KR） | 約 900件 | 約 9 KB | 韓国国内アクセス制御用超軽量DB |
| `cn.dat` | 中国（CN） | 約 4,100件 | 約 41 KB | 中国アクセス識別・制御用軽量DB |
| `dbip-country-lite.csv` | 全世界 | 約 142,000件 | 約 4.3 MB | xtables（`xt_geoip_build`）互換DB-IP CSV |
| `checksum.sha256` | - | - | テキスト | 全生成ファイルのSHA-256ハッシュ一覧 |

### チェックサム検証（Linux / macOS）
```bash
sha256sum -c checksum.sha256
```

---

## 6. なぜ KRNIC データなのか？（JPNICおよび他NICとの比較）

### 1) JPNIC ではなく KRNIC を採用した理由
- **全世界のIP帯域をワンストップ（単一CSV）で提供：**
  - **JPNIC（日本）：** 日本国内（JP）のIPアドレス割り当て・管理を中心とする国別インターネットレジストリ（NIR）であり、全世界の国別IP割り当てデータを1つのファイルに集約・公開していません。全世界データを集めるには上位RIR（APNIC等）の統計ファイルを個別に集約する必要があります。
  - **KRNIC（韓国KISA）：** 韓国国内（KR）だけでなく、IANAおよび世界5大地域レジストリ（APNIC, ARIN, RIPE NCC, LACNIC, AFRINIC）の割り当て情報を定期的に同期・統合し、**全世界200以上の国・地域の全IPv4帯域を単一のCSVファイル（`IPaddrBandCurrentDownload.jsp`）として毎日公開**しています。
- **APIキー・ライセンス制約のない完全自動化への適合性：**
  - MaxMind GeoLite2やDB-IPなどの商用GeoIPデータベースは、アカウント登録、ライセンスキーの発行・更新、再配布制限（EULA）、定期的データ破棄義務などの厳しい制約が伴います。
  - KRNICのデータは公共情報システム（IPAS）を通じてシンプルなHTTPリクエストで最新版を取得可能なため、GitHub Actionsによる週次完全自動ビルドパイプラインに最適です。

### 2) インターネットレジストリ（NIC / RIR）および商用DBとの比較

| 区分 | KRNIC（韓国） | JPNIC（日本） | 大陸別 RIR（APNIC, ARIN 等） | 商用 DB（MaxMind, DB-IP 等） |
|---|---|---|---|---|
| **管轄領域** | 韓国(NIR) ＋ **グローバル統合集計** | 日本国内(NIR) 中心 | 当該大陸・地域(RIR) に限定 | 全世界（独自収集・推定） |
| **提供範囲** | **全世界200以上の国・地域** | 主に日本国内およびリンク案内 | 5つのRIRにファイルが分散 | 全世界 |
| **提供フォーマット** | **クレンジング済みの標準CSV**<br>（`開始IP,終了IP,国コード,割当日`） | 主にWHOIS照会または統計リンク | 原初統計仕様（`delegated-*-latest`） | CSV, MMDB等の独自形式 |
| **前処理の難易度** | **極めて低い**（即座にパース可能） | 収集不可（他機関を参照する必要あり） | **高い**（5ファイル結合 ＋ CIDR/ホスト数換算） | 低い |
| **利用・ライセンス制約** | 公共データ（認証キー不要） | 各機関の会員規約に準拠 | 非営利・統計目的に準拠 | **要登録・APIキー必須・EULA制約** |

### 3) RIR原初統計ファイル（`delegated-*-latest`）に対する優位性
各大陸の公式レジストリ（APNIC, ARIN等）の生データは `開始IP` と `ホスト数` の形式であり、`allocated`（割当済）、`assigned`（配分済）、`reserved`（予約）、`available`（未割当）などのステータスが混在しています。これらを全世界分集めるには5つのFTPから取得し、CIDR計算やフィルタリングを行う必要があります。KRNICは国家機関レベルでこれらをクレンジング・集約した単一テーブルを提供するため、前処理のオーバーヘッドがありません。

---

## 7. データ出典

- **データ出典：** [KRNIC 韓国インターネット情報センター (KISA)](https://xn--3e0bx5euxnjje69i70af08bea817g.xn--3e0b707e/jsp/statboard/IPAS/ovrse/natal/IPaddrBandCurrentDownload.jsp)
