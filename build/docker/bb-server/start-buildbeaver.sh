#!/bin/bash

BB_VAR_PREFIX="BB_VAR_"
BB_CERT_DIR="/tmp/buildbeaver-jwt-certs/"

# Explicitly load vars from /etc/environment
if [ -e "/etc/environment" ]; then
  while IFS="=", read k v # split on =
  do
    ve=$(echo "$v" | sed -e 's/^"//' -e 's/"$//') # remove leading and trailing quotes
    export $k="$ve" # export the var
  done < "/etc/environment"
fi

# Load flags
flags=""

# Flags can be passed via a single combined env var
if [ -n "$BUILDBEAVER_FLAGS" ]; then
  flags="$flags $BUILDBEAVER_FLAGS "
fi

###############################################################################
# JWT certificate handler if we are passing in JWT cert / verify keys
###############################################################################
JWT_CERTS_ENABLED=false
set_jwt_certs () {
  if [ "$JWT_CERTS_ENABLED" = true ]; then
    return
  fi
  mkdir -p $BB_CERT_DIR
  flags="$flags --jwt_certificate_directory $BB_CERT_DIR "
  JWT_CERTS_ENABLED=true
}

# Flags can be passed via env vars prefixed with a special value
while IFS='=' read -r -d '' name value; do
  if [[ $name == $BB_VAR_PREFIX* ]]; then
    name=${name#"BB_VAR_"}
    if [ "${name}" == "github_app_private_key" ]; then
      echo "${value}" > "/tmp/github-private-key.pem"
      flags="$flags --github_app_private_key_file_path /tmp/github-private-key.pem "
    elif [ "${name}" == "jwt_certificate_private_key" ]; then
      set_jwt_certs
      echo "${value}" > "${BB_CERT_DIR}jwt-private-key.pem"
    elif [ "${name}" == "jwt_verifying_public_key" ]; then
      set_jwt_certs
      echo "${value}" > "${BB_CERT_DIR}jwt-cert.pem"
    else
      flags="$flags --$name $value "
    fi
  fi
done < <(env -0)

# Flags can be set via the flag file
if [ -e "/etc/buildbeaver/flags" ]; then
  while read -r line
  do
    flags="$flags $line "
  done < "/etc/buildbeaver/flags"
fi

# NOTE we use exec to ensure signals from the outside world are passed through
# to the BuildBeaver process (not this script)
exec /usr/bin/bb-server $@ $flags
