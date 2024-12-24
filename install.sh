#!/usr/bin/env bash

set -e

# Version can be overridden by setting FILEFUSION_VERSION
VERSION=${FILEFUSION_VERSION:-"latest"}
GITHUB_REPO="drgsn/filefusion"
INSTALL_DIR="${HOME}/.filefusion"

print_help() {
    cat << EOF
FileFusion Installer

Usage: ./install.sh [options]

Options:
    -h, --help      Show this help message
    -v, --version   Install specific version instead of latest
    -d, --dir       Specify installation directory (default: ~/.filefusion)
    -s, --skip-rc   Skip updating shell RC files

Environment variables:
    FILEFUSION_VERSION    Set specific version to install
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            print_help
            exit 0
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -d|--dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -s|--skip-rc)
            SKIP_RC=1
            shift
            ;;
        *)
            echo "Unknown option: $1"
            print_help
            exit 1
            ;;
    esac
done

# Function to detect OS and architecture
detect_platform() {
    local os arch

    # Detect OS
    case "$(uname -s)" in
        Darwin)
            os="darwin"
            ;;
        Linux)
            os="linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            os="windows"
            ;;
        *)
            echo "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac

    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            echo "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Function to detect shell and get rc file path
get_shell_rc() {
    local shell_rc=""

    if [ -n "$ZSH_VERSION" ]; then
        shell_rc="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            shell_rc="$HOME/.bash_profile"
        else
            shell_rc="$HOME/.bashrc"
        fi
    elif [ -n "$FISH_VERSION" ]; then
        shell_rc="$HOME/.config/fish/config.fish"
    fi

    if [ -z "$shell_rc" ]; then
        echo "Unable to detect shell configuration file"
        exit 1
    fi

    echo "$shell_rc"
}

# Function to download the latest release
download_release() {
    local platform="$1"
    local ext="tar.gz"
    if [[ "$platform" == windows* ]]; then
        ext="zip"
    fi

    local version_no
    if [ "$VERSION" = "latest" ]; then
        # Get the latest version number from the latest release
        version_no=$(curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/^v//')
        if [ -z "$version_no" ]; then
            echo "Failed to get latest version number"
            exit 1
        fi
    else
        # Remove 'v' prefix if present
        version_no=$(echo "$VERSION" | sed 's/^v//')
    fi

    local release_url="https://github.com/${GITHUB_REPO}/releases/download/v${version_no}/filefusion_${version_no}_${platform}.${ext}"

    echo "Downloading FileFusion from: $release_url"
    
    # Create temporary directory
    local tmp_dir
    tmp_dir="$(mktemp -d)"
    
    # Download the release
    if ! curl -fsSL "$release_url" -o "${tmp_dir}/filefusion.${ext}"; then
        echo "Failed to download FileFusion"
        rm -rf "$tmp_dir"
        exit 1
    fi

    # Extract the release
    mkdir -p "$INSTALL_DIR"
    if [[ "$platform" == windows* ]]; then
        unzip -o "${tmp_dir}/filefusion.${ext}" -d "$INSTALL_DIR"
    else
        tar xzf "${tmp_dir}/filefusion.${ext}" -C "$INSTALL_DIR"
    fi

    # Cleanup
    rm -rf "$tmp_dir"
}

# Function to update shell configuration
update_shell_rc() {
    local shell_rc="$1"
    local path_export="export PATH=\"\$PATH:$INSTALL_DIR\""
    
    # Check if path is already in shell rc
    if ! grep -q "$INSTALL_DIR" "$shell_rc"; then
        echo "" >> "$shell_rc"
        echo "# FileFusion" >> "$shell_rc"
        echo "$path_export" >> "$shell_rc"
        echo "Updated $shell_rc"
    else
        echo "$shell_rc already contains FileFusion path"
    fi
}

# Function to update PowerShell profile on Windows
update_powershell_profile() {
    local profile_dir="$HOME/Documents/PowerShell"
    local profile_path="$profile_dir/Microsoft.PowerShell_profile.ps1"
    
    mkdir -p "$profile_dir"
    
    if [ ! -f "$profile_path" ] || ! grep -q "$INSTALL_DIR" "$profile_path"; then
        echo "" >> "$profile_path"
        echo "# FileFusion" >> "$profile_path"
        echo "\$env:Path += \";$INSTALL_DIR\"" >> "$profile_path"
        echo "Updated PowerShell profile"
    else
        echo "PowerShell profile already contains FileFusion path"
    fi
}

# Main installation process
main() {
    local platform
    platform="$(detect_platform)"
    
    echo "Installing FileFusion for $platform..."
    
    # Download and extract the release
    download_release "$platform"
    
    # Make binary executable (not needed for Windows)
    if [[ "$platform" != windows* ]]; then
        chmod +x "$INSTALL_DIR/filefusion"
    fi
    
    # Update shell configuration
    if [ -z "$SKIP_RC" ]; then
        if [[ "$platform" == windows* ]]; then
            update_powershell_profile
            # Also update shell rc for Git Bash/MSYS2 if applicable
            if [ -n "$BASH_VERSION" ]; then
                update_shell_rc "$(get_shell_rc)"
            fi
        else
            update_shell_rc "$(get_shell_rc)"
        fi
    fi
    
    echo "FileFusion installed successfully!"
    echo "To start using FileFusion, either:"
    echo "1. Restart your terminal"
    echo "2. Run: source $(get_shell_rc)"
}

# Run the installer
main