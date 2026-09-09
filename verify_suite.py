#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GeoIP Global Binary Database Test Suite
OECD 10개국 (각 10개 사이트/IP) + 대륙별 작은 나라 10개국 (각 2~3개 사이트/IP) 검증 스크립트
"""

import subprocess
import sys
import time

TEST_DATA = {
    # -------------------------------------------------------------
    # OECD 10개국 (각 10개 사이트)
    # -------------------------------------------------------------
    "OECD": {
        "KR (대한민국)": [
            ("gov.kr", "125.60.35.230", "정부24 포털"),
            ("seoul.go.kr", "115.84.166.115", "서울특별시청"),
            ("naver.com", "223.130.200.219", "네이버 대표 포털"),
            ("daum.net", "121.53.105.193", "다음/카카오 포털"),
            ("snu.ac.kr", "147.46.10.129", "서울대학교"),
            ("korea.kr", "27.101.217.76", "대한민국 정책브리핑"),
            ("police.go.kr", "116.67.83.27", "경찰청"),
            ("mofa.go.kr", "116.67.79.26", "외교부"),
            ("visitkorea.or.kr", "175.122.1.106", "한국관광공사"),
            ("busan.go.kr", "210.103.81.224", "부산광역시청"),
        ],
        "JP (일본)": [
            ("yahoo.co.jp", "124.83.190.252", "야후 재팬 대표 포털"),
            ("rakuten.co.jp", "133.237.182.225", "라쿠텐 대표 쇼핑몰"),
            ("www.u-tokyo.ac.jp", "210.152.243.234", "도쿄대학교"),
            ("kyoto-u.ac.jp", "130.54.130.14", "교토대학교"),
            ("www.osaka-u.ac.jp", "133.1.138.13", "오사카대학교"),
            ("tohoku.ac.jp", "130.34.11.111", "도호쿠대학교"),
            ("www.pref.osaka.lg.jp", "210.149.92.76", "오사카부청"),
            ("city.yokohama.lg.jp", "202.32.8.152", "요코하마시청"),
            ("iij.ad.jp", "202.232.2.191", "IIJ 대표 통신/ISP"),
            ("sakura.ne.jp", "163.43.179.80", "사쿠라 인터넷 IDC"),
        ],
        "US (미국)": [
            ("whitehouse.gov", "192.0.66.51", "백악관 공식 포털"),
            ("nasa.gov", "192.0.66.108", "NASA 항공우주국"),
            ("loc.gov", "104.17.6.58", "미국 의회도서관"),
            ("harvard.edu", "192.0.66.20", "하버드 대학교"),
            ("mit.edu", "23.35.126.95", "매사추세츠공대 MIT"),
            ("stanford.edu", "171.67.215.200", "스탠퍼드 대학교"),
            ("cdc.gov", "23.53.3.141", "질병통제예방센터 CDC"),
            ("irs.gov", "152.216.11.110", "국세청 IRS"),
            ("senate.gov", "23.35.114.182", "미국 연방상원"),
            ("house.gov", "15.197.146.213", "미국 연방하원"),
        ],
        "GB (영국)": [
            ("ucl.ac.uk", "144.82.250.24", "유니버시티 칼리지 런던"),
            ("ed.ac.uk", "129.215.97.20", "에든버러 대학교"),
            ("imperial.ac.uk", "146.179.12.148", "임페리얼 칼리지 런던"),
            ("kcl.ac.uk", "137.73.130.135", "킹스 칼리지 런던"),
            ("warwick.ac.uk", "137.205.28.41", "워릭 대학교"),
            ("bristol.ac.uk", "137.222.180.160", "브리스톨 대학교"),
            ("southampton.ac.uk", "152.78.118.52", "사우샘프턴 대학교"),
            ("birmingham.ac.uk", "147.188.217.187", "버밍엄 대학교"),
            ("www.sheffield.ac.uk", "143.167.2.102", "셰필드 대학교"),
            ("st-andrews.ac.uk", "138.251.7.84", "세인트 앤드루스 대학교"),
        ],
        "DE (독일)": [
            ("bundestag.de", "46.243.122.48", "독일 연방의회"),
            ("bundeskanzler.de", "185.173.230.39", "독일 연방총리실"),
            ("lmu.de", "141.84.44.56", "뮌헨 대학교 LMU"),
            ("hu-berlin.de", "141.20.4.180", "베를린 훔볼트 대학교"),
            ("tum.de", "129.187.254.228", "뮌헨 공과대학교 TUM"),
            ("uni-heidelberg.de", "129.206.13.71", "하이델베르크 대학교"),
            ("uni-koeln.de", "134.95.81.52", "쾰른 대학교"),
            ("uni-frankfurt.de", "141.2.37.198", "프랑크푸르트 대학교"),
            ("kit.edu", "141.3.128.6", "칼스루에 공과대학교 KIT"),
            ("rwth-aachen.de", "137.226.107.60", "아헨 공과대학교 RWTH"),
        ],
        "FR (프랑스)": [
            ("gouvernement.fr", "217.70.184.55", "프랑스 정부 공식 포털"),
            ("elysee.fr", "185.194.81.29", "엘리제궁 대통령실"),
            ("www.assemblee-nationale.fr", "46.105.202.26", "프랑스 국민의회"),
            ("senat.fr", "45.85.52.26", "프랑스 연방상원"),
            ("ens.psl.eu", "129.199.166.211", "파리 고등사범학교 ENS"),
            ("polytechnique.edu", "129.104.30.29", "에콜 폴리테크니크"),
            ("univ-paris1.fr", "193.55.96.23", "판테온 소르본 대학교"),
            ("strasbourg.eu", "185.60.150.88", "스트라스부르 시청"),
            ("bordeaux.fr", "195.214.227.180", "보르도 시청"),
            ("univ-lyon1.fr", "134.214.126.72", "클로드 베르나르 리옹1대학교"),
        ],
        "AU (호주)": [
            ("anu.edu.au", "130.56.67.33", "호주 국립대학교 ANU"),
            ("unimelb.edu.au", "43.245.41.62", "멜버른 대학교"),
            ("uq.edu.au", "130.102.184.3", "퀸즐랜드 대학교 UQ"),
            ("monash.edu", "43.245.41.240", "모나시 대학교"),
            ("vic.gov.au", "103.107.226.226", "빅토리아 주정부 포털"),
            ("tas.gov.au", "147.109.249.170", "태즈메이니아 주정부"),
            ("aarnet.edu.au", "202.158.207.3", "호주 국가연구학술망"),
            ("telstra.com.au", "203.44.22.2", "텔스트라 대표 통신사"),
            ("qut.edu.au", "131.181.196.203", "퀸즐랜드 공과대학교 QUT"),
            ("deakin.edu.au", "128.184.204.21", "디킨 대학교"),
        ],
        "CA (캐나다)": [
            ("mcgill.ca", "132.216.98.121", "맥길 대학교"),
            ("umontreal.ca", "132.204.8.144", "몬트리올 대학교"),
            ("ucalgary.ca", "136.159.96.125", "캘거리 대학교"),
            ("uottawa.ca", "137.122.9.76", "오타와 대학교"),
            ("westernu.ca", "129.100.0.55", "웨스턴 대학교"),
            ("sfu.ca", "142.58.103.107", "사이먼 프레이저 대학교"),
            ("uvic.ca", "142.104.197.120", "빅토리아 대학교"),
            ("dal.ca", "129.173.31.187", "댈하우지 대학교"),
            ("umanitoba.ca", "130.179.16.50", "매니토바 대학교"),
            ("yorku.ca", "130.63.236.137", "요크 대학교"),
        ],
        "IT (이탈리아)": [
            ("camera.it", "80.64.114.73", "이탈리아 하원의회"),
            ("unibo.it", "137.204.24.207", "볼로냐 대학교"),
            ("unimi.it", "159.149.53.140", "밀라노 대학교"),
            ("unipd.it", "147.162.235.155", "파도바 대학교"),
            ("www.unina.it", "143.225.161.30", "나폴리 페데리코 2세 대학교"),
            ("unifi.it", "150.217.3.39", "피렌체 대학교"),
            ("polimi.it", "131.175.187.72", "밀라노 공과대학교"),
            ("www.polito.it", "130.192.182.100", "토리노 공과대학교"),
            ("garr.it", "193.206.158.22", "이탈리아 국가학술망 GARR"),
            ("comune.torino.it", "84.240.178.132", "토리노 시청"),
        ],
        "ES (스페인)": [
            ("lamoncloa.gob.es", "212.128.109.1", "스페인 총리실 라 몽클로아"),
            ("congreso.es", "193.145.227.245", "스페인 하원의회"),
            ("ub.edu", "161.116.109.141", "바르셀로나 대학교"),
            ("ucm.es", "147.96.2.159", "마드리드 콤플루텐세 대학교"),
            ("uab.cat", "158.109.121.133", "바르셀로나 자치대학교"),
            ("uam.es", "150.244.214.237", "마드리드 자치대학교"),
            ("upm.es", "138.100.200.6", "마드리드 공과대학교 UPM"),
            ("www.uv.es", "147.156.200.249", "발렌시아 대학교"),
            ("upv.es", "158.42.4.23", "발렌시아 공과대학교"),
            ("rediris.es", "130.206.13.20", "스페인 학술망 RedIRIS"),
        ],
    },
    # -------------------------------------------------------------
    # 대륙별 작은 나라 10개국 (각 2~3개 사이트)
    # -------------------------------------------------------------
    "SmallCountries": {
        "MC (모나코 - 유럽)": [
            ("gouv.mc", "82.113.11.58", "모나코 정부 공식 포털"),
            ("monaco-telecom.mc", "195.78.23.147", "모나코 텔레콤"),
            ("mairie.mc", "80.94.99.164", "모나코 시청"),
        ],
        "LI (리히텐슈타인 - 유럽)": [
            ("regierung.li", "91.207.130.57", "리히텐슈타인 정부"),
            ("landtag.li", "91.207.130.57", "리히텐슈타인 연방의회"),
            ("uni.li", "193.5.27.37", "리히텐슈타인 대학교"),
        ],
        "IS (아이슬란드 - 유럽)": [
            ("hi.is", "130.208.165.58", "아이슬란드 대학교"),
            ("vedur.is", "94.142.156.174", "아이슬란드 기상청"),
            ("postur.is", "82.221.64.147", "아이슬란드 국립우정청"),
        ],
        "BN (브루나이 - 아시아)": [
            ("gov.bn", "103.4.188.110", "브루나이 정부 공식 포털"),
            ("mof.gov.bn", "103.4.188.86", "브루나이 재무부"),
            ("ubd.edu.bn", "202.160.1.115", "브루나이 다루살람 대학교"),
        ],
        "BT (부탄 - 아시아)": [
            ("gov.bt", "103.78.116.169", "부탄 정부 포털"),
            ("moh.gov.bt", "103.252.84.250", "부탄 보건부"),
            ("tashicell.com", "118.103.136.91", "부탄 타시셀 이동통신"),
        ],
        "MV (몰디브 - 아시아)": [
            ("gov.mv", "123.176.25.10", "몰디브 정부 공식 포털"),
            ("dhiraagu-telecom", "27.114.128.1", "몰디브 디라구 국영텔레콤"),
            ("ooredoo-maldives", "43.226.220.1", "몰디브 오레두 모바일망"),
        ],
        "SC (세이셸 - 아프리카)": [
            ("www.gov.sc", "196.13.208.87", "세이셸 공화국 정부"),
            ("seychelles.travel", "41.86.57.50", "세이셸 국립관광청"),
            ("intelvision.sc", "41.220.110.236", "세이셸 인텔비전 통신망"),
        ],
        "MU (모리셔스 - 아프리카)": [
            ("govmu.org", "196.13.125.126", "모리셔스 정부 포털"),
            ("myt.mu", "196.20.130.50", "모리셔스 마이티 텔레콤"),
            ("uom.ac.mu", "202.60.7.10", "모리셔스 대학교"),
        ],
        "BZ (벨리즈 - 아메리카)": [
            ("belizetourismboard.org", "186.65.88.123", "벨리즈 관광청 포털"),
            ("btl-telemedia.bz", "200.32.192.1", "벨리즈 국영텔레미디어"),
            ("centralbank.org.bz", "200.32.208.1", "벨리즈 중앙은행 전산망"),
        ],
        "FJ (피지 - 오세아니아)": [
            ("www.fiji.gov.fj", "124.108.30.90", "피지 정부 공식 포털"),
            ("usp.ac.fj", "144.120.198.5", "남태평양 대학교 피지 본교"),
            ("vodafone.com.fj", "27.123.183.54", "보다폰 피지 통신망"),
        ],
    }
}

def verify_ip(dat_path, ip):
    cmd = ["./geoip_krnic2dbip", "-verify", dat_path, "-ip", ip]
    res = subprocess.run(cmd, capture_output=True, text=True)
    if res.returncode != 0:
        return False, None, None, None
    country, ip_range, elapsed = "", "", ""
    for line in res.stdout.splitlines():
        line = line.strip()
        if line.startswith("- Country"):
            country = line.split(":", 1)[1].strip()
        elif line.startswith("- IP Range"):
            ip_range = line.split(":", 1)[1].strip()
        elif line.startswith("- Lookup Time"):
            elapsed = line.split(":", 1)[1].strip()
    return True, country, ip_range, elapsed

def main():
    dat_path = "global.dat"
    target_filter = None

    if len(sys.argv) > 1:
        dat_path = sys.argv[1]
    if len(sys.argv) > 2:
        target_filter = sys.argv[2].upper()

    print("=" * 80)
    filter_msg = f" (국가 필터: {target_filter})" if target_filter else ""
    print(f" GeoIP 바이너리 검증 테스트 스위트 (대상 파일: {dat_path}{filter_msg})")
    print("=" * 80)

    total_tests = 0
    passed_tests = 0
    start_all = time.time()

    for category, groups in TEST_DATA.items():
        matched_groups = {k: v for k, v in groups.items() if not target_filter or k.split()[0].upper() == target_filter}
        if not matched_groups:
            continue

        print(f"\n[{category}] 검증 시작")
        print("-" * 80)

        for group_name, items in matched_groups.items():
            expected_cc = group_name.split()[0].upper()
            print(f"\n▶ 국가: {group_name} (기대 국가코드: {expected_cc})")
            
            for domain, ip, desc in items:
                total_tests += 1
                ok, country, ip_range, elapsed = verify_ip(dat_path, ip)
                if ok and country == expected_cc:
                    passed_tests += 1
                    status = "PASS"
                    print(f"  [O] {status} | {domain:26} ({ip:15}) -> [{country}] {ip_range:30} ({elapsed})")
                else:
                    status = "FAIL"
                    actual = country if ok else "NOT FOUND"
                    print(f"  [X] {status} | {domain:26} ({ip:15}) -> Expected: {expected_cc}, Got: {actual}")

    if total_tests == 0:
        print(f"\n[!] 필터 조건 '{target_filter}'에 해당하는 테스트 데이터가 없습니다.")
        sys.exit(1)

    duration = time.time() - start_all
    print("\n" + "=" * 80)
    print(f" 검증 완료: 총 {total_tests}건 중 {passed_tests}건 성공 ({passed_tests/total_tests*100:.1f}%) | 총 소요시간: {duration:.2f}초")
    print("=" * 80)

    if passed_tests == total_tests:
        sys.exit(0)
    else:
        sys.exit(1)

if __name__ == "__main__":
    main()
