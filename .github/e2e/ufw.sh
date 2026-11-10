#!/bin/bash
# Real ufw on the runner: rules 22, 80, 443, 2053 (IPv4 only), delete "1,3" via delete_ports.
set -u
fn() { awk '/^delete_ports\(\) \{/,/^\}/' "$1"; }
git show "$BASE_SHA:x-ui.sh" > /tmp/x-ui.base.sh
FAIL=0
for side in base head; do
  script=x-ui.sh; [[ $side == base ]] && script=/tmp/x-ui.base.sh
  sudo sed -i 's/^IPV6=.*/IPV6=no/' /etc/default/ufw
  sudo ufw --force reset >/dev/null
  for p in 22/tcp 80/tcp 443/tcp 2053/tcp; do sudo ufw allow $p >/dev/null; done
  sudo ufw --force enable >/dev/null
  echo "== $side: before"; sudo ufw status numbered | grep '^\['
  # Each ufw prompt reads its own answer, as at a terminal: ufw would buffer all of a piped stdin.
  { printf '1\n1,3\n'; sleep 3; printf 'y\n'; sleep 3; printf 'y\n'; } | sudo bash -c "$(fn "$script"); delete_ports" >/tmp/out.txt 2>&1
  grep -E 'Deleting|Rule deleted|Failed' /tmp/out.txt
  echo "== $side: after deleting 1,3 (should delete 22/tcp and 443/tcp)"
  left=$(sudo ufw status numbered | grep '^\[' | awk '{print $3}' | tr '\n' ' ')
  sudo ufw status numbered | grep '^\['
  want="80/tcp 2053/tcp "
  if [[ $side == head && $left != "$want" ]]; then echo "FAIL head left: $left"; FAIL=1; fi
  if [[ $side == base && $left == "$want" ]]; then echo "base unexpectedly correct"; fi
done
sudo ufw --force disable >/dev/null
exit $FAIL
