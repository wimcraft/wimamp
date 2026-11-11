#!/bin/sh
# Install wimamp from GitHub releases:
#   curl -fsSL https://raw.githubusercontent.com/wimcraft/wimamp/main/install.sh | sh
# Options (env): WIMAMP_VERSION=v0.1.0 (default: latest), WIMAMP_INSTALL_DIR (default: ~/.local/bin)
set -eu

repo=wimcraft/wimamp
dir=${WIMAMP_INSTALL_DIR:-$HOME/.local/bin}
version=${WIMAMP_VERSION:-latest}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $os in
darwin | linux) ;;
*) echo "wimamp: unsupported OS '$os'. On Windows, run in PowerShell: irm https://raw.githubusercontent.com/$repo/main/install.ps1 | iex" >&2; exit 1 ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) echo "wimamp: unsupported CPU '$(uname -m)'" >&2; exit 1 ;;
esac

asset=wimamp-$os-$arch
if [ "$version" = latest ]; then
	base=https://github.com/$repo/releases/latest/download
else
	base=https://github.com/$repo/releases/download/$version
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "Downloading $asset ($version)..."
curl -fsSL "$base/$asset" -o "$tmp/wimamp"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

expected=$(awk -v f="$asset" '$2 == f { print $1 }' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
	actual=$(sha256sum "$tmp/wimamp" | cut -d' ' -f1)
else
	actual=$(shasum -a 256 "$tmp/wimamp" | cut -d' ' -f1)
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
	echo "wimamp: checksum mismatch for $asset, not installing" >&2
	exit 1
fi

mkdir -p "$dir"
chmod 755 "$tmp/wimamp"
mv -f "$tmp/wimamp" "$dir/wimamp" # mv replaces a symlink instead of writing through it
echo "Installed $("$dir/wimamp" version) to $dir/wimamp"

case ":$PATH:" in
*":$dir:"*) ;;
*) echo "Note: $dir is not on your PATH. Add it, e.g.: export PATH=\"$dir:\$PATH\"" ;;
esac
if ! command -v mpv >/dev/null 2>&1; then
	echo "wimamp plays through mpv, which is not installed yet:"
	if [ "$os" = darwin ]; then echo "  brew install mpv"; else echo "  sudo apt install mpv   (or your distro's package)"; fi
fi
