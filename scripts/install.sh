#!/bin/bash

set -e

echo "Determining platform..."
platform=$(uname | tr '[:upper:]' '[:lower:]')
echo "Finding latest release..."
asset=$(curl --silent https://api.github.com/repos/liamg/scout/releases/latest | jq -r ".assets[] | select(.name | contains(\"${platform}\")) | .url")
echo "Downloading latest release for your platform..."
curl -s -L -H "Accept: application/octet-stream" "${asset}" --output ./scout
echo "Installing scout..."
chmod +x ./scout
installdir="${HOME}/bin/"
if [ "$EUID" -eq 0 ]; then
  installdir="/usr/local/bin/"
fi
mkdir -p $installdir
mv ./scout "${installdir}/scout"
which scout &> /dev/null || (echo "Please add ${installdir} to your PATH to complete installation!" && exit 1)
echo "Installation complete!"
