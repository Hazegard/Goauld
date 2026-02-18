#!/bin/bash

CONFIG_FILE="$1"

TLS_SERVER="$( yq .tls-server < "$CONFIG_FILE")"

DNS_DOMAIN="$( yq .dns-domain < "$CONFIG_FILE")"

HTTP_SERVER="$( yq .server < "$CONFIG_FILE")"

DNS_SERVER="127.0.0.1"


ID="00000000000000000000000000000000"

{ echo "${ID}S" ;sleep 1; } | go run ./dns/dnstt-client/ -udp "$DNS_SERVER":53 "$DNS_DOMAIN" 2>/dev/null | grep -q SSH-2.0-OpenSSH_8.7 && echo "SSHoDNS OK" || echo "SSHoDNS KO"

{ echo -ne ""; sleep 1; } | go run ./websocket/ -insecure_conn ws://"$HTTP_SERVER"/wssh/"$ID" 2>/dev/null | grep -q SSH-2.0-OpenSSH_8.7 && echo "SSHoWS OK" || echo "SSHoWS KO"

echo -ne "$ID"  | timeout 1 openssl s_client -connect "$TLS_SERVER":443 -quiet  2>/dev/null | grep -q SSH-2.0-OpenSSH_8.7 && echo "SSHoTLS OK" || echo "SSHoTLS KO"

echo "" | timeout 1 nc localhost 2222 | grep -q SSH-2.0-OpenSSH_8.7 && echo "Direct SSH OK" || echo "Direct SSH KO"
