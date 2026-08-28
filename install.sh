#!/usr/bin/env bash
set -euo pipefail

is_nixos() {
	[[ -f /etc/NIXOS ]] && return 0
	if [[ -f /etc/os-release ]]; then
		# shellcheck disable=SC1091
		. /etc/os-release
		[[ "${ID:-}" == "nixos" || "${NAME:-}" == "NixOS" ]]
	fi
}

if is_nixos; then
	if command -v nix >/dev/null 2>&1; then
		echo "NixOS detected; installing via nix profile."
		nix profile add github:Axenide/axctl
		exit 0
	fi
	echo "NixOS detected but nix command is unavailable."
	exit 1
fi

os="$(uname -s)"
arch="$(uname -m)"

if [[ "$os" != "Linux" ]]; then
	echo "Unsupported OS: $os"
	exit 1
fi

case "$arch" in
x86_64)
	asset="axctl_linux_amd64"
	;;
i386 | i686)
	asset="axctl_linux_386"
	;;
aarch64)
	asset="axctl_linux_arm64"
	;;
armv7l | armv7 | armv6l)
	asset="axctl_linux_armv7"
	;;
*)
	echo "Unsupported architecture: $arch"
	exit 1
	;;
esac

latest_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' https://github.com/Axenide/axctl/releases/latest)"
latest_tag="${latest_url##*/}"
if [[ -z "$latest_tag" || "$latest_tag" == "latest" ]]; then
	echo "Unable to determine latest release tag."
	exit 1
fi

normalize_version() {
	local s="${1#v}"
	if [[ "$s" =~ ([0-9]+(\.[0-9]+)*) ]]; then
		printf '%s\n' "${BASH_REMATCH[1]}"
	fi
}

latest_version="$(normalize_version "$latest_tag")"
if [[ -z "$latest_version" ]]; then
	echo "Unable to parse latest version from tag: $latest_tag"
	exit 1
fi

if command -v axctl >/dev/null 2>&1; then
	current_raw="$(axctl --version 2>/dev/null || true)"
	current_version="$(normalize_version "$current_raw")"
	if [[ -n "$current_version" && "$(printf '%s\n%s\n' "$current_version" "$latest_version" | sort -V | head -n1)" == "$latest_version" ]]; then
		echo "already up to date ($current_version)"
		exit 0
	fi
fi

url="https://github.com/Axenide/axctl/releases/latest/download/${asset}"
tmp="$(mktemp)"

cleanup() {
	rm -f "$tmp"
}
trap cleanup EXIT

echo "Downloading $asset (${latest_tag})..."
curl -fL "$url" -o "$tmp"

if [[ $EUID -ne 0 ]]; then
	if command -v sudo >/dev/null 2>&1; then
		sudo install -m 755 "$tmp" /usr/local/bin/axctl
	else
		echo "sudo is required to install to /usr/local/bin."
		exit 1
	fi
else
	install -m 755 "$tmp" /usr/local/bin/axctl
fi
echo "Installed /usr/local/bin/axctl"
