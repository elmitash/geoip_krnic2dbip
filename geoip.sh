#!/bin/bash

wget --no-check-certificate https://한국인터넷정보센터.한국/jsp/statboard/IPAS/ovrse/natal/IPaddrBandCurrentDownload.jsp -O ipv4.csv
./geoip_krnic2dbip
/usr/local/libexec/xtables-addons/xt_geoip_build -D /usr/share/xt_geoip
