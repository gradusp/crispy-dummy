if [ "$1" -ge 1 ]; then
  systemctl stop crispy-dummy.service
fi
if [ "$1" = 0 ]; then
  systemctl disable --now crispy-dummy.service
fi
