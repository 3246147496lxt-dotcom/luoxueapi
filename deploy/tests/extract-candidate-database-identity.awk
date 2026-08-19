/^[{][^[:cntrl:]]*[}]$/ {
  identity = $0
  count++
}

END {
  if (count != 1) {
    exit 1
  }
  print identity
}
