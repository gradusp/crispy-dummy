if [ "$1" -ge 1 ]; then
  echo "$1"
fi
if [ "$1" = 0 ]; then
  systemctl daemon-reload
  rm -rf "/opt/crispy/dummy"
fi
exit 0