[ 🇯🇵 [日本語](README.md) | 🇰🇷 **한국어** ]

# KRNIC GeoIP to Binary Database Converter (`geoip_krnic2dbip`)

KRNIC(한국인터넷정보센터)의 국가별 IPv4 할당 현황 데이터를 초고속 조회가 가능한 **고정 10바이트 바이너리 포맷(`.dat`)** 및 DB-IP 호환 포맷으로 변환하고 자동 배포하는 Go 언어 기반 도구입니다.

GitHub Actions를 통해 매일 최신 KRNIC 데이터를 자동으로 다운로드하여 빌드하고, 무결성 검증을 거친 후 GitHub Release Assets(`global.dat`, `jp.dat`, `kr.dat`, `cn.dat`, `dbip-country-lite.csv`, `checksum.sha256`)로 자동 배포됩니다.

---

## 1. 주요 특징

- **초고속 이진 탐색(Binary Search):** 고정 10바이트 레코드 구조로 설계되어 별도의 메모리 인덱스 구축 없이 `fseek` 기반 $O(\log N)$ 이진 탐색으로 1회 조회당 **0.0005ms(약 430ns)** 이내에 국가 코드를 식별합니다.
- **연속 대역 자동 병합(Range Merging):** 동일 국가의 인접/연속된 IP 대역을 자동 병합하여 레코드 수를 45% 이상 압축(약 26만 개 -> 약 14만 개)합니다.
- **원패스 일괄 빌드(`-all`):** KRNIC CSV를 단 1회만 파싱하여 전 세계(`global.dat`), 일본 특화(`jp.dat`), 한국 특화(`kr.dat`), 중국 특화(`cn.dat`), `dbip-country-lite.csv` 및 SHA-256 체크섬(`checksum.sha256`)을 한 번에 생성합니다.
- **동적 복수 국가 필터링:** 파라미터(`-country JP,KR,CN`)로 지정한 국가 코드들을 단 1회 순회로 동적 분기하여 개별 `.dat` 파일로 각각 추출합니다.
- **기존 쉘 스크립트 및 xtables 완벽 호환:** 인자 없이 실행 시 레거시 모드로 동작하여 `dbip-country-lite.csv`를 즉시 생성합니다.
- **자체 검증(Verification) 모드:** CLI 자체에 이진 탐색 엔진을 내장하여 생성된 바이너리의 IP 조회를 즉시 검증할 수 있습니다.
- **완전 자동화 파이프라인:** 매일 오전 04:23 KST(19:23 UTC)에 GitHub Actions가 최신 데이터를 수집하여 릴리즈를 갱신합니다.

---

## 2. 바이너리 레코드 포맷 규격 (Fixed 10-Byte Big-Endian)

모든 `.dat` 파일은 헤더가 없는 순수 바이너리 스트림으로 구성되며, 1개 레코드는 **정확히 10바이트** 고정 크기를 가집니다.  
파일 전체 크기는 항상 **10의 정수배**이며, 총 레코드 수는 `파일 크기(bytes) / 10`으로 즉시 계산할 수 있습니다.

### 레코드 구조

| 오프셋 (Offset) | 필드명 | 데이터 타입 | 바이트 순서 (Endian) | 설명 |
|---|---|---|---|---|
| `[0 : 4]` | `FirstNum` | `uint32` (4 bytes) | Big-Endian | 대역 시작 IPv4 정수값 |
| `[4 : 8]` | `LastNum` | `uint32` (4 bytes) | Big-Endian | 대역 종료 IPv4 정수값 |
| `[8 : 10]` | `CountryCode` | `char[2]` (2 bytes) | ASCII | ISO 3166-1 alpha-2 2자리 국가코드 (`JP`, `KR`, `US` 등) |

### 정렬 보장
모든 레코드는 **`FirstNum` 오름차순**으로 완벽히 정렬되어 기록되므로, 파일 임의 접근(Random Access)을 통한 이진 탐색이 즉시 가능합니다.

---

## 3. 설치 및 빌드

### 요구 환경
- Go 1.22 이상

### 소스코드 빌드
```bash
git clone https://github.com/elmitash/geoip_krnic2dbip.git
cd geoip_krnic2dbip
go build -o geoip_krnic2dbip ./cmd
```

### 테스트 및 벤치마크 실행
```bash
go test -v -bench=. ./cmd/...
```

---

## 4. CLI 사용법

### 1) 레거시 CSV 변환 모드 (기존 쉘 스크립트 및 xtables 100% 호환)
기존 `geoip.sh` 쉘 스크립트 및 `xt_geoip_build` 파이프라인과 완벽히 호환됩니다. 인자 없이 실행하면 현재 디렉터리의 `ipv4.csv`를 읽어 `dbip-country-lite.csv`로 즉시 변환합니다.
```bash
# 인자 없이 실행 시 자동으로 dbip-country-lite.csv 생성 (기존과 동일)
./geoip_krnic2dbip

# 출력 CSV 파일명을 직접 지정할 때
./geoip_krnic2dbip -csv custom-country-lite.csv
```

### 2) 원패스 전체 빌드 (신규 바이너리 + CSV 일괄 생성, 권장)
KRNIC CSV(`ipv4.csv`)를 1회만 파싱하여 `global.dat`, 지정된 국가별 `.dat` 파일들, `checksum.sha256` 및 `dbip-country-lite.csv`를 한 번에 일괄 생성합니다.
```bash
# 기본 모드: global.dat + jp.dat, kr.dat, cn.dat + dbip-country-lite.csv + checksum.sha256 생성
./geoip_krnic2dbip -all

# 사용자 정의 국가 지정 모드: global.dat + us.dat, gb.dat, de.dat + dbip-country-lite.csv 생성
./geoip_krnic2dbip -all -country US,GB,DE
```
- 옵션:
  - `-country` (또는 `-countries`): 추출할 2자리 국가 코드 (콤마 구분으로 복수 지정 가능, 기본값: `JP,KR,CN`)
  - `-in <파일경로>`: 입력 KRNIC CSV 경로 지정 (기본값: `ipv4.csv`)
  - `-csv <파일경로>`: DB-IP 포맷 CSV 출력 경로 지정 (기본값: `dbip-country-lite.csv`)

### 3) 특정 국가 바이너리만 빌드 (단일 / 복수 국가 동적 추출)
파라미터로 국가 코드를 지정하면, 해당 국가 레코드만 별도로 분리하여 추출합니다. 복수 개의 국가도 콤마(`,`)로 구분하여 한 번에 분리할 수 있습니다.
```bash
# 복수 국가를 각각의 .dat 파일로 한 번에 추출 (결과: jp.dat, kr.dat, cn.dat)
./geoip_krnic2dbip -country JP,KR,CN

# 서구권 국가들만 각각 추출 (결과: us.dat, gb.dat, de.dat, fr.dat)
./geoip_krnic2dbip -country US,GB,DE,FR

# 단일 국가 지정 및 출력 파일명 커스텀 지정
./geoip_krnic2dbip -country JP -out custom_japan.dat
```

### 4) 바이너리 데이터 검증 모드 (Verification)
생성된 `.dat` 파일에서 임의의 IP를 이진 탐색으로 조회하여 일치 여부와 소요 시간을 확인합니다.
```bash
# 일본 Yahoo IP 검증 -> JP 매칭 확인
./geoip_krnic2dbip -verify jp.dat -ip 182.22.59.229

# 한국 포털 Naver IP 검증 -> KR 매칭 확인
./geoip_krnic2dbip -verify kr.dat -ip 211.249.220.24

# 중국 검색 Baidu IP 검증 -> CN 매칭 확인
./geoip_krnic2dbip -verify cn.dat -ip 220.181.38.148

# Google Public DNS 검증 -> US 매칭 확인
./geoip_krnic2dbip -verify global.dat -ip 8.8.8.8
```

#### 검증 출력 예시:
```text
Verifying IP 182.22.59.229 in binary file: jp.dat ...
Result: MATCH FOUND!
  - Country   : JP
  - IP Range  : 182.20.0.0 - 182.22.255.255
  - Range (u32): 3054764032 - 3054960639
  - Lookup Time: 31.72µs
```

### 5) 실데이터 대량 검증 스위트 (`verify_suite.py`)
OECD 10개국(한국, 일본, 미국, 영국, 독일, 프랑스, 호주, 캐나다, 이탈리아, 스페인 각 10개) 및 대륙별 소국 10개국(모나코, 리히텐슈타인, 아이슬란드, 브루나이, 부탄, 몰디브, 세이셸, 모리셔스, 벨리즈, 피지) 총 130개 실제 공공/포털/언론사 IP를 원클릭으로 일괄 검증합니다.
```bash
# 전체 130개 IP 일괄 검증 (global.dat)
python3 verify_suite.py global.dat

# 특정 국가만 검증 (예: jp.dat에 대해 일본 사이트 10개만 검증)
python3 verify_suite.py jp.dat JP
python3 verify_suite.py kr.dat KR
```

---

## 5. 생성 에셋 및 파일 설명

| 파일명 | 대상 국가 | 예상 레코드 수 | 예상 파일 크기 | 용도 및 특징 |
|---|---|---|---|---|
| `global.dat` | 전 세계 전체 | 약 142,000건 | 약 1.4 MB | 글로벌 IP 판별 및 지리적 접근 제어 |
| `jp.dat` | 일본 (JP) | 약 2,500건 | 약 25 KB | 일본 내수용 서비스/EC 사이트 특화 경량 DB |
| `kr.dat` | 한국 (KR) | 약 900건 | 약 9 KB | 국내 접속 전용 초경량 DB |
| `cn.dat` | 중국 (CN) | 약 4,100건 | 약 41 KB | 중국 접속 식별 및 제어용 경량 DB |
| `dbip-country-lite.csv` | 전 세계 전체 | 약 142,000건 | 약 4.3 MB | xtables(`xt_geoip_build`) 호환용 DB-IP CSV 파일 |
| `checksum.sha256` | - | - | 텍스트 | 생성된 모든 파일의 SHA-256 해시값 검증 파일 |

### 무결성 검증 (Linux / macOS)
```bash
sha256sum -c checksum.sha256
```

---

## 6. 왜 KRNIC 데이터를 사용하는가? (JPNIC 및 타 NIC과의 비교)

### 1) JPNIC 대신 KRNIC을 선택한 이유
- **전 세계(글로벌) IP 대역의 원스톱(One-Stop) 단일 파일 제공:**
  - **JPNIC (일본):** 일본 국내(JP) 대역 관리 중심의 국가 인터넷 레지스트리(NIR)로서, 전 세계 국가별 IP 할당 대역을 정제하여 단일 파일로 공개 배포하지 않습니다. 전 세계 데이터를 얻으려면 상위 RIR(APNIC 등)의 원시 통계 파일을 별도로 수집해야 합니다.
  - **KRNIC (한국 KISA):** 한국 국내(KR)뿐만 아니라 IANA 및 5대 대륙별 RIR(APNIC, ARIN, RIPE NCC, LACNIC, AFRINIC)의 할당 데이터를 매일 정기적으로 동기화·취합하여 **전 세계 200여 개국의 전체 IPv4 대역을 단일 CSV 파일(`IPaddrBandCurrentDownload.jsp`)로 제공**합니다.
- **인증키 / 라이선스 제약 없는 자동화 최적화:**
  - MaxMind GeoLite2나 DB-IP 같은 상용 데이터베이스는 회원가입, 라이선스 키 갱신, EULA 재배포 제약 및 정기적 파기 의무가 존재합니다.
  - KRNIC 데이터는 공공 전산망(IPAS)을 통해 HTTP 요청 한 번으로 최신 데이터를 즉시 취득할 수 있어, GitHub Actions를 통한 주간 완전 무인 자동화 빌드 파이프라인에 최적입니다.

### 2) 인터넷 레지스트리(NIC / RIR) 및 상용 DB 비교

| 구분 | KRNIC (한국) | JPNIC (일본) | 대륙별 RIR (APNIC, ARIN 등) | 상용 DB (MaxMind, DB-IP 등) |
|---|---|---|---|---|
| **관할 영역** | 한국(NIR) ＋ **글로벌 통합 집계** | 일본 국내(NIR) 중심 | 해당 대륙/지역(RIR) 한정 | 전 세계 (자체 추정/수집) |
| **제공 범위** | **전 세계 200여 개국 통합** | 주로 일본 국내 및 링크 안내 | 5대 RIR로 데이터 파편화 | 전 세계 |
| **제공 포맷** | **정제된 표준 CSV**<br>(`시작IP,끝IP,국가코드,할당일`) | 주로 WHOIS 질의 또는 통계 링크 | 원시 통계 규격 (`delegated-*-latest`) | CSV, MMDB 등 자체 규격 |
| **전처리 난이도** | **매우 낮음** (즉시 파싱 및 변환) | 수집 불가 (타 기관 참조 필요) | **높음** (5개 파일 병합 ＋ CIDR/개수 환산) | 낮음 |
| **라이선스/이용 제약** | 공공 데이터 (인증키 불필요) | 기관 회원/약관 기준 | 비상업적/통계 목적 등 각 규격 준수 | **가입 필수, 라이선스 키, EULA 제약** |

### 3) RIR 원시 통계(`delegated-*-latest`) 대비 장점
대륙별 공식 레지스트리(APNIC, ARIN 등)의 원시 파일은 `시작IP`와 `호스트 개수(Count)` 기반 포맷이며, `allocated`(할당), `assigned`(배정), `reserved`(예약), `available`(미할당) 등의 상태 코드가 혼재되어 있어 5개 대륙의 파일을 모두 수집해 복잡한 필터링과 계산을 거쳐야 합니다. KRNIC은 국가 공공 전산망 차원에서 이를 완전히 정제·통합한 단일 테이블을 제공하므로 전처리 오버헤드가 없습니다.

---

## 7. 데이터 출처

- **데이터 출처:** KRNIC 한국인터넷정보센터 (KISA)

