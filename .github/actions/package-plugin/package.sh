#!/bin/bash
# Strict mode
set -e
set -u
set -o pipefail
IFS=$'\n\t' # Set Internal Field Separator to just newline and tab

# This script packages a Grafana plugin for distribution
# It takes a plugin directory as input and generates zip packages
# Further comments explaining the motivation and intended use for the script is in the action.yml file

# --- Function Definitions ---
check_dependency() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: Required dependency '$1' is not installed or not in PATH"
    MISSING_DEPS=1 # This variable needs to be initialized before this function is first called
  else
    echo "✓ Found $1"
  fi
}

# Function to run the plugin signing tool
run_plugin_signer() {
  local dir_to_sign="$1"
  local SIGNER_USED="" # To track if a signer command was successfully chosen
  echo "Signing plugin in directory: $dir_to_sign"

  # Change to the directory to sign
  pushd "$dir_to_sign" > /dev/null

  local yarn_exists=false
  local yarn_v2_plus_available=false

  if command -v yarn &>/dev/null; then
    yarn_exists=true
    local YARN_VERSION # Local to this block
    YARN_VERSION=$(yarn --version 2>/dev/null || echo "0")
    local YARN_MAJOR_VERSION="${YARN_VERSION%%.*}" # Local to this block
    if [[ "$YARN_MAJOR_VERSION" =~ ^[0-9]+$ ]] && [ "$YARN_MAJOR_VERSION" -ge 2 ]; then
        yarn_v2_plus_available=true
    fi
  fi

  if [ "$yarn_v2_plus_available" = true ]; then
    echo "Using yarn dlx to sign plugin"
    yarn dlx @grafana/sign-plugin@latest --distDir=.
    SIGNER_USED="yarn"
  elif command -v npm &>/dev/null; then # Try npm if yarn v2+ not available/used
    if [ "$yarn_exists" = true ]; then # Implies yarn was found but not v2+
      echo "Using npx to sign plugin (yarn v1 or non-numeric/older version detected)"
    else # Yarn was not found
      echo "Using npx to sign plugin"
    fi
    npx --yes @grafana/sign-plugin@latest --distDir=.
    SIGNER_USED="npm"
  fi

  if [ -z "${SIGNER_USED:-}" ]; then
    if [ "$ALLOW_NO_SIGN" = true ]; then # ALLOW_NO_SIGN needs to be in scope
      echo "Info: Plugin signing skipped (--no-sign flag was used or no suitable package manager was found)."
    else
      echo "Error: No suitable package manager (yarn v2+ or npm) found for signing plugin."
      echo "       To build without signing, use the -n flag."
      exit 1
    fi
  fi
  unset SIGNER_USED # Clean up local variable
  # Helper variables like yarn_exists, yarn_v2_plus_available, YARN_VERSION, YARN_MAJOR_VERSION were local and go out of scope.

  # Return to the previous directory
  popd > /dev/null
}

# --- Input Validation & Early Setup ---
ALLOW_NO_SIGN=false
PLUGIN_DIR_INPUT="" # To store the user-provided directory path if any

# Parse options using getopts
USAGE_STRING="Usage: $0 [-n] [-d plugin_directory]"

while getopts ":nd:" opt; do
  case "$opt" in
    n)
      ALLOW_NO_SIGN=true
      echo "Info: -n (no-sign) flag detected. Plugin signing will be skipped if no signer is found or if forced."
      ;;
    d)
      PLUGIN_DIR_INPUT="$OPTARG"
      ;;
    \?)
      echo "Error: Invalid option: -$OPTARG" >&2
      echo "$USAGE_STRING" >&2
      exit 1
      ;;
    :)
      echo "Error: Option -$OPTARG requires an argument." >&2
      echo "$USAGE_STRING" >&2
      exit 1
      ;;
  esac
done

shift $((OPTIND - 1)) # Shift processed options away

# Check for any remaining non-option arguments (should be none)
if [ -n "${1:-}" ]; then
  echo "Error: Unexpected argument: $1" >&2
  echo "$USAGE_STRING" >&2
  exit 1
fi

# Default to current directory if -d was not used
if [ -z "$PLUGIN_DIR_INPUT" ]; then
  echo "Info: No plugin directory specified via -d, defaulting to current directory: $(pwd)"
  PLUGIN_DIR_INPUT="$(pwd)"
fi

# Resolve absolute path and check if it's a directory
ABS_PLUGIN_DIR=""
if cd "$PLUGIN_DIR_INPUT" 2>/dev/null; then
  ABS_PLUGIN_DIR=$(pwd)
  # cd back to original directory if PWD was used as input, to avoid side effects
  # However, the script logic from here uses ABS_PLUGIN_DIR or PLUGIN_DIR, so current PWD state after this matters less.
  # To be absolutely safe, one might `cd - >/dev/null` if the original PWD needs preservation for later script parts not shown/affected.
else
  echo "Error: Plugin directory '$PLUGIN_DIR_INPUT' is not a valid directory or could not be accessed."
  exit 1
fi

if [ ! -d "$ABS_PLUGIN_DIR" ]; then
  echo "Error: Resolved path '$ABS_PLUGIN_DIR' is not a directory."
  exit 1
fi

PLUGIN_DIR="$ABS_PLUGIN_DIR"
DIST_DIR="${PLUGIN_DIR}/dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: dist directory not found in $PLUGIN_DIR"
  echo "Make sure the plugin has been built before packaging"
  exit 1
fi

# Read plugin metadata using jq from the dist directory
PLUGIN_ID=$(jq -r '.id' "${DIST_DIR}/plugin.json")
VERSION=$(jq -r '.info.version' "${DIST_DIR}/plugin.json")

if [ -z "$PLUGIN_ID" ] || [ "$PLUGIN_ID" == "null" ]; then
  echo "Error: Could not read plugin ID from ${DIST_DIR}/plugin.json, or ID is null."
  exit 1
fi
if [ -z "$VERSION" ] || [ "$VERSION" == "null" ]; then
  echo "Error: Could not read plugin version from ${DIST_DIR}/plugin.json, or version is null."
  exit 1
fi

# --- Environment Dependency Checks ---
echo "Checking for required dependencies..."
MISSING_DEPS=0 # Initialize MISSING_DEPS here, before check_dependency is called

# Check for required tools
check_dependency "jq"      # For parsing plugin.json
check_dependency "zip"     # For creating zip archives
check_dependency "sha1sum" # For generating checksums
check_dependency "find"    # For finding files
check_dependency "mktemp"  # For creating temporary directories

if [ $MISSING_DEPS -ne 0 ]; then
  echo "Please install the missing dependencies and try again."
  exit 1
fi

# --- Main Procedural Logic ---
echo "Packaging plugin: $PLUGIN_ID version $VERSION"

OUTPUT_DIR="${PLUGIN_DIR}/__packaging/${PLUGIN_ID}"

# Create output directories
mkdir -p "${OUTPUT_DIR}/${VERSION}"
mkdir -p "${OUTPUT_DIR}/latest"

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
PLUGIN_BUILD_DIR="${BUILD_DIR}/${PLUGIN_ID}"
mkdir -p "$PLUGIN_BUILD_DIR" # Ensure the target directory for cp exists
echo "Using temporary directory: $BUILD_DIR"

# Copy all plugin files to the build directory
echo "Copying plugin files..."
cp -r "$DIST_DIR"/* "$PLUGIN_BUILD_DIR/" # Copy contents of dist into PLUGIN_BUILD_DIR

# Look for the executable name in plugin.json
EXECUTABLE=$(jq -r '.executable // empty' "${DIST_DIR}/plugin.json")

# Build GO_BINARIES array if EXECUTABLE is set
GO_BINARIES=()
if [ -n "$EXECUTABLE" ]; then
  PLATFORM_PATTERNS=(
    "${EXECUTABLE}_darwin_amd64"
    "${EXECUTABLE}_darwin_arm64"
    "${EXECUTABLE}_linux_amd64"
    "${EXECUTABLE}_linux_arm"
    "${EXECUTABLE}_linux_arm64"
    "${EXECUTABLE}_windows_amd64.exe"
  )
  for PATTERN in "${PLATFORM_PATTERNS[@]}"; do
    # Search within the copied dist contents (PLUGIN_BUILD_DIR)
    # Correct find path and ensure it handles no matches gracefully
    # Use a subshell for find to not affect the main script with `cd` or errors if find fails to find anything.
    # find returns non-zero if it finds nothing, but we don't want the script to exit due to `set -e` here.
    # So we use `|| true` if we just want to check for presence.
    # Consider if EXECUTABLE could have glob characters; if so, PATTERN needs care.
    # Assuming EXECUTABLE is a clean name.
    temp_found_path=$(find "$PLUGIN_BUILD_DIR" -name "$PATTERN" -print -quit 2>/dev/null || true)
    if [ -n "$temp_found_path" ]; then
      GO_BINARIES+=("$temp_found_path")
    fi    
  done
  # Set permissions on all found Go binaries to 0755
  for BINARY in "${GO_BINARIES[@]}"; do
    chmod 0755 "$BINARY"
  done
fi

# Sign the plugin (contents of PLUGIN_BUILD_DIR) before creating platform-agnostic package
run_plugin_signer "$PLUGIN_BUILD_DIR"

# Create the platform-agnostic zip
echo "Creating platform-agnostic package..."
PLATFORM_AGNOSTIC_ZIP="${OUTPUT_DIR}/${VERSION}/${PLUGIN_ID}-${VERSION}.zip"
LATEST_PLATFORM_AGNOSTIC_ZIP="${OUTPUT_DIR}/latest/${PLUGIN_ID}-latest.zip"

(cd "$BUILD_DIR" && zip -qr "$PLATFORM_AGNOSTIC_ZIP" "$PLUGIN_ID")
cp "$PLATFORM_AGNOSTIC_ZIP" "$LATEST_PLATFORM_AGNOSTIC_ZIP"

if [ -z "$EXECUTABLE" ]; then
  echo "No executable specified in plugin.json, skipping platform-specific packaging"
  # SHA1 sums and cleanup will be done at the end
elif [ ${#GO_BINARIES[@]} -gt 0 ]; then
  echo "Found ${#GO_BINARIES[@]} platform-specific binaries, creating platform-specific packages..."

  for BINARY in "${GO_BINARIES[@]}"; do
    # Extract the filename without path
    BINARY_NAME=$(basename "$BINARY")
    GOOS="" # Initialize to prevent using value from previous iteration if determination fails
    GOARCH=""

    # Determine GOOS and GOARCH from the binary name
    if [[ "$BINARY_NAME" == *"_darwin_amd64"* ]]; then
      GOOS="darwin"
      GOARCH="amd64"
    elif [[ "$BINARY_NAME" == *"_darwin_arm64"* ]]; then
      GOOS="darwin"
      GOARCH="arm64"
    elif [[ "$BINARY_NAME" == *"_linux_amd64"* ]]; then
      GOOS="linux"
      GOARCH="amd64"
    elif [[ "$BINARY_NAME" == *"_linux_arm"* && ! "$BINARY_NAME" == *"_linux_arm64"* ]]; then
      GOOS="linux"
      GOARCH="arm"
    elif [[ "$BINARY_NAME" == *"_linux_arm64"* ]]; then
      GOOS="linux"
      GOARCH="arm64"
    elif [[ "$BINARY_NAME" == *"_windows_amd64.exe"* ]]; then
      GOOS="windows"
      GOARCH="amd64"
    else
      echo "Warning: Could not determine platform for $BINARY_NAME, skipping"
      continue
    fi

    echo "Processing binary for $GOOS/$GOARCH..."

    # Create working directory for this platform
    PLATFORM_DIR="$BUILD_DIR/platform_${GOOS}_${GOARCH}"
    mkdir -p "$PLATFORM_DIR/$PLUGIN_ID"

    # Copy all plugin files (from the signed PLUGIN_BUILD_DIR) to the platform directory first
    cp -r "$PLUGIN_BUILD_DIR"/* "$PLATFORM_DIR/$PLUGIN_ID/"

    # Remove all platform-specific binaries from the platform directory
    # Ensure paths are quoted. The find command itself is robust due to -exec +.
    find "$PLATFORM_DIR/$PLUGIN_ID" -type f -name "${EXECUTABLE}_*" -exec rm -f {} + 2>/dev/null || true

    # Copy just this specific binary (which was originally in PLUGIN_BUILD_DIR) to the correct location in PLATFORM_DIR
    # The path of BINARY is absolute (e.g., /tmp/tmp.X/plugin-id/executable_os_arch)
    # We need its relative path within PLUGIN_BUILD_DIR to reconstruct its target path in PLATFORM_DIR/$PLUGIN_ID
    BINARY_REL_PATH="${BINARY#"$PLUGIN_BUILD_DIR"/}" # e.g., executable_os_arch or sub/executable_os_arch
    mkdir -p "$(dirname "$PLATFORM_DIR/$PLUGIN_ID/$BINARY_REL_PATH")"
    cp "$BINARY" "$PLATFORM_DIR/$PLUGIN_ID/$BINARY_REL_PATH"

    # Sign the platform-specific plugin contents (now with only one binary)
    run_plugin_signer "$PLATFORM_DIR/$PLUGIN_ID"

    # Create platform-specific output directories
    mkdir -p "${OUTPUT_DIR}/${VERSION}/${GOOS}"
    mkdir -p "${OUTPUT_DIR}/latest/${GOOS}"

    # Create the platform-specific zip
    PLATFORM_ZIP="${OUTPUT_DIR}/${VERSION}/${GOOS}/${PLUGIN_ID}-${VERSION}-${GOOS}_${GOARCH}.zip"
    LATEST_PLATFORM_ZIP="${OUTPUT_DIR}/latest/${GOOS}/${PLUGIN_ID}-latest-${GOOS}_${GOARCH}.zip"

    (cd "$PLATFORM_DIR" && zip -qr "$PLATFORM_ZIP" "$PLUGIN_ID")
    cp "$PLATFORM_ZIP" "$LATEST_PLATFORM_ZIP"

    echo "Created $PLATFORM_ZIP"
  done
else # EXECUTABLE was set, but GO_BINARIES array is empty
  echo "No platform-specific binaries found matching defined patterns, skipping platform-specific packaging"
fi

# Generate SHA1 checksums for all zip files (with just the filename, not the full path)
# Ensure pipefail is active in the subshell for sha1sum | sed
find "${OUTPUT_DIR}" -name "*.zip" -exec bash -c 'set -o pipefail; FILENAME=$(basename "$1"); sha1sum "$1" | sed "s| .*| $FILENAME|" > "$1.sha1"' _ {} \;

# Clean up
rm -rf "$BUILD_DIR"
echo "Plugin packaging complete! Packages are available in $OUTPUT_DIR" 
