#!/bin/bash
# Sandbox e2e: $1 = 2fa|domain. Needs $BEFORE, $AFTER (panel binaries) and $T (e2etool).
set -u
export XUI_DB_FOLDER=/tmp/e2e/db XUI_LOG_FOLDER=/tmp/e2e/log XUI_BIN_FOLDER=/tmp/e2e/bin
B=http://127.0.0.1:2053/e2e
PID= FAIL=0 OUT=

init_db() { rm -rf /tmp/e2e; mkdir -p $XUI_DB_FOLDER $XUI_LOG_FOLDER $XUI_BIN_FOLDER
  "$1" setting -username admin -password 'S3cret-Pass-e2e' -port 2053 -webBasePath /e2e/ >/dev/null 2>&1
  TOKEN=$("$1" setting -getApiToken 2>/dev/null | sed -n 's/^apiToken: //p')
  [[ -n $TOKEN ]] || { echo "no API token"; exit 1; }; }
start() { "$1" run >/tmp/e2e/run.log 2>&1 & PID=$!
  for _ in $(seq 60); do curl -s -o /dev/null http://127.0.0.1:2053/ && return; sleep 0.5; done
  echo "panel did not start"; tail -30 /tmp/e2e/run.log; exit 1; }
stop() { kill $PID 2>/dev/null; wait $PID 2>/dev/null; }
api() { curl -s -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' "$@"; }
save() { api -X POST $B/panel/api/setting/all -o /tmp/e2e/all.json; "$T" set /tmp/e2e/all.json "$@" > /tmp/e2e/upd.json
  OUT=$(api -X POST $B/panel/api/setting/update --data @/tmp/e2e/upd.json -w ' HTTP %{http_code}'); }
login() { rm -f /tmp/e2e/jar; local csrf
  csrf=$(curl -s -c /tmp/e2e/jar -b /tmp/e2e/jar $B/csrf-token | sed 's/.*"obj":"\([^"]*\)".*/\1/')
  OUT=$(curl -s -c /tmp/e2e/jar -b /tmp/e2e/jar -H "X-CSRF-Token: $csrf" -w ' HTTP %{http_code}' \
    --data-urlencode username=admin --data-urlencode password=S3cret-Pass-e2e --data-urlencode "twoFactorCode=$1" $B/login); }
status() { OUT=$(curl -s -o /dev/null -w 'HTTP %{http_code}' "$@"); }
stored() { api -X POST $B/panel/api/setting/all -o /tmp/e2e/all.json; OUT="stored $1 = '$("$T" get /tmp/e2e/all.json "$1")'"; }
# expect "<what>" "<extended regex the output must match>"
expect() { if [[ $OUT =~ $2 ]]; then echo "ok   $1: $OUT"; else echo "FAIL $1: $OUT (want /$2/)"; FAIL=1; fi; }
code() { "$T" code JBSWY3DPEHPK3PXP; }

if [[ $1 == 2fa ]]; then
  echo "== base (upstream main): turn 2FA on through the settings API with a lower-case secret"
  init_db "$BEFORE"; start "$BEFORE"
  save twoFactorEnable=true twoFactorToken=jbswy3dpehpk3pxp; expect "save lower-case secret" '"success":true'
  stop; start "$BEFORE"
  login "$(code)"; expect "login with the right code" 'HTTP 500$'
  save twoFactorEnable=false twoFactorToken= "twoFactorCode=\"$(code)\""; expect "turn 2FA off with the right code" 'HTTP 500$'
  stop
  echo "== head (fix) on the same database, secret still stored in lower case"
  start "$AFTER"
  login "$(code)"; expect "login with the right code" '"success":true.* HTTP 200$'
  login 000000; expect "login with a wrong code" '"success":false.* HTTP 200$'
  save twoFactorEnable=false twoFactorToken= "twoFactorCode=\"$(code)\""; expect "turn 2FA off with the right code" '"success":true.* HTTP 200$'
  stop
  echo "== head (fix), fresh database"
  init_db "$AFTER"; start "$AFTER"
  save twoFactorEnable=true 'twoFactorToken=jbsw y3dp ehpk 3pxp'; expect "save lower-case grouped secret" '"success":true'
  login "$(code)"; expect "login with the right code" '"success":true.* HTTP 200$'
  stop; init_db "$AFTER"; start "$AFTER"
  save twoFactorEnable=true twoFactorToken=totp-secret; expect "save non-base32 secret" '"success":false.*base32'
  save twoFactorEnable=true twoFactorToken=; expect "turn 2FA on with no secret" '"success":false.*base32'
  login ""; expect "login without a code (2FA stayed off)" '"success":true.* HTTP 200$'
  stop
else
  H='Host: panel.example.com:2053'
  echo "== base (upstream main): web domain saved as 'Panel.Example.com'"
  init_db "$BEFORE"; start "$BEFORE"
  save webDomain=Panel.Example.com; expect "save" '"success":true'
  stop; start "$BEFORE"
  status -H "$H" $B/csrf-token; expect "GET /e2e/csrf-token as panel.example.com" "^HTTP 403$"
  status -H "$H" -H "Authorization: Bearer $TOKEN" -X POST $B/panel/api/setting/all; expect "settings API as panel.example.com" '^HTTP 403$'
  stop
  echo "== head (fix) on the same database, domain still stored with capitals"
  start "$AFTER"
  status -H "$H" $B/csrf-token; expect "GET /e2e/csrf-token as panel.example.com" "^HTTP 200$"
  status -H "Host: PANEL.example.com.:2053" $B/csrf-token; expect "GET /e2e/csrf-token as PANEL.example.com." "^HTTP 200$"
  status -H "Host: other.example.com:2053" $B/csrf-token; expect "GET /e2e/csrf-token as other.example.com" '^HTTP 403$'
  status -H "$H" -H "Authorization: Bearer $TOKEN" -X POST $B/panel/api/setting/all; expect "settings API as panel.example.com" '^HTTP 200$'
  stop
  echo "== head (fix), fresh database"
  init_db "$AFTER"; start "$AFTER"
  save 'webDomain= Panel.Example.COM. ' subDomain=Sub.Example.com; expect "save ' Panel.Example.COM. '" '"success":true'
  stored webDomain; expect "web domain" "= 'panel.example.com'$"
  stored subDomain; expect "sub domain" "= 'sub.example.com'$"
  stop; init_db "$AFTER"; start "$AFTER"
  save webDomain=https://panel.example.com/; expect "save https://panel.example.com/" '"success":false.*host name'
  stored webDomain; expect "web domain" "= ''$"
  stop
fi
exit $FAIL
