#!/bin/bash

EXE_FILE=krnic2dbip_linux_amd64
#EXE_FILE=krnic2dbip_linux_arm64
BUILD_FILE=/usr/local/libexec/xtables-addons/xt_geoip_build

mkdir -p /usr/share/xt_geoip
wget --no-check-certificate https://한국인터넷정보센터.한국/jsp/statboard/IPAS/ovrse/natal/IPaddrBandCurrentDownload.jsp -O ipv4.csv
./"${EXE_FILE}"

if [ ! -f "$BUILD_FILE" ]; then
  # for Ubuntu 22.04 LTS
  /usr/libexec/xtables-addons/xt_geoip_build -D /usr/share/xt_geoip
else
  # for Ubuntu 18.04 LTS
  /usr/local/libexec/xtables-addons/xt_geoip_build -D /usr/share/xt_geoip
fi
