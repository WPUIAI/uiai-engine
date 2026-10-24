# Shared optional HTTP authentication. Source from Bash 3.2+ or POSIX sh.
# Positional parameters avoid nounset failures from initialized empty arrays.
uiai_curl_auth() {
  if [ -n "${UIAI_API_KEY:-}" ]; then
    set -- -H "X-API-Key: ${UIAI_API_KEY}" "$@"
  elif [ -n "${UIAI_BEARER_TOKEN:-}" ]; then
    set -- -H "Authorization: Bearer ${UIAI_BEARER_TOKEN}" "$@"
  fi
  command curl "$@"
}
