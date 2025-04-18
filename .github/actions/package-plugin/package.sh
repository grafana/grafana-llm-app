#!/bin/bash
set -e

# This script packages a Grafana plugin for distribution
# It takes a plugin directory as input and generates zip packages
# Further comments explaining the motivation and intended use for the script is in the action.yml file

# Check for required dependencies
echo "Checking for required dependencies..."
MISSING_DEPS=0

check_dependency() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Error: Required dependency '$1' is not installed or not in PATH"
    MISSING_DEPS=1
  else
    echo "✓ Found $1"
  fi
}

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

# Get the absolute path of the plugin directory (first argument)
PLUGIN_DIR="$1"
if [ -z "$PLUGIN_DIR" ]; then
  echo "Error: Plugin directory not specified"
  echo "Usage: $0 <plugin_directory>"
  exit 1
fi

PLUGIN_DIR="$(cd "$PLUGIN_DIR" && pwd)"
DIST_DIR="${PLUGIN_DIR}/dist"

if [ ! -d "$DIST_DIR" ]; then
  echo "Error: dist directory not found in $PLUGIN_DIR"
  echo "Make sure the plugin has been built before packaging"
  exit 1
fi

# Read plugin metadata using jq from the dist directory
PLUGIN_ID=$(jq -r '.id' "${DIST_DIR}/plugin.json")
VERSION=$(jq -r '.info.version' "${DIST_DIR}/plugin.json")
OUTPUT_DIR="${PLUGIN_DIR}/__packaging/${PLUGIN_ID}"

echo "Packaging plugin: $PLUGIN_ID version $VERSION"

# Create output directories
mkdir -p "${OUTPUT_DIR}/${VERSION}"
mkdir -p "${OUTPUT_DIR}/latest"

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
PLUGIN_BUILD_DIR="${BUILD_DIR}/${PLUGIN_ID}"
echo "Using temporary directory: $BUILD_DIR"

# Copy all plugin files to the build directory
echo "Copying plugin files..."
cp -r "$DIST_DIR" "$PLUGIN_BUILD_DIR"

# Create the platform-agnostic zip
echo "Creating platform-agnostic package..."
PLATFORM_AGNOSTIC_ZIP="${OUTPUT_DIR}/${VERSION}/${PLUGIN_ID}-${VERSION}.zip"
LATEST_PLATFORM_AGNOSTIC_ZIP="${OUTPUT_DIR}/latest/${PLUGIN_ID}-latest.zip"

(cd "$BUILD_DIR" && zip -r "$PLATFORM_AGNOSTIC_ZIP" "$PLUGIN_ID")
cp "$PLATFORM_AGNOSTIC_ZIP" "$LATEST_PLATFORM_AGNOSTIC_ZIP"

# Look for the executable name in plugin.json
EXECUTABLE=$(jq -r '.executable // empty' "${DIST_DIR}/plugin.json")
if [ -z "$EXECUTABLE" ]; then
  echo "No executable specified in plugin.json, skipping platform-specific packaging"
  # Generate SHA1 checksums for all zip files (with just the filename, not the full path)
  find "${OUTPUT_DIR}" -name "*.zip" -exec sh -c 'FILENAME=$(basename "$1"); sha1sum "$1" | sed "s| .*| $FILENAME|" > "$1.sha1"' _ {} \;
  # Clean up
  rm -rf "$BUILD_DIR"
  echo "Plugin packaging complete! Packages are available in $OUTPUT_DIR"
  exit 0
fi

# Platform patterns to search for
PLATFORM_PATTERNS=(
  "${EXECUTABLE}_darwin_amd64"
  "${EXECUTABLE}_darwin_arm64"
  "${EXECUTABLE}_linux_amd64"
  "${EXECUTABLE}_linux_arm"
  "${EXECUTABLE}_linux_arm64"
  "${EXECUTABLE}_windows_amd64.exe"
)

# Find platform-specific binaries
GO_BINARIES=()
for PATTERN in "${PLATFORM_PATTERNS[@]}"; do
  found=$(find "$PLUGIN_BUILD_DIR" -name "$PATTERN" 2>/dev/null || true)
  if [ -n "$found" ]; then
    GO_BINARIES+=("$found")
  fi
done

if [ ${#GO_BINARIES[@]} -gt 0 ]; then
  echo "Found ${#GO_BINARIES[@]} platform-specific binaries, creating platform-specific packages..."
  
  for BINARY in "${GO_BINARIES[@]}"; do
    # Extract the filename without path
    BINARY_NAME=$(basename "$BINARY")

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
    
    # Copy all plugin files to the platform directory first
    cp -r "$PLUGIN_BUILD_DIR"/* "$PLATFORM_DIR/$PLUGIN_ID/"
    
    # Remove all platform-specific binaries from the platform directory
    find "$PLATFORM_DIR/$PLUGIN_ID" -type f -name "${EXECUTABLE}_*" | xargs rm -f 2>/dev/null || true
    
    # Copy just this specific binary to the correct location
    BINARY_REL_PATH=${BINARY#$PLUGIN_BUILD_DIR/}
    mkdir -p "$(dirname "$PLATFORM_DIR/$PLUGIN_ID/$BINARY_REL_PATH")"
    cp "$BINARY" "$PLATFORM_DIR/$PLUGIN_ID/$BINARY_REL_PATH"
    
    # Create platform-specific output directories
    mkdir -p "${OUTPUT_DIR}/${VERSION}/${GOOS}"
    mkdir -p "${OUTPUT_DIR}/latest/${GOOS}"
    
    # Create the platform-specific zip
    PLATFORM_ZIP="${OUTPUT_DIR}/${VERSION}/${GOOS}/${PLUGIN_ID}-${VERSION}-${GOOS}_${GOARCH}.zip"
    LATEST_PLATFORM_ZIP="${OUTPUT_DIR}/latest/${GOOS}/${PLUGIN_ID}-latest-${GOOS}_${GOARCH}.zip"
    
    (cd "$PLATFORM_DIR" && zip -r "$PLATFORM_ZIP" "$PLUGIN_ID")
    cp "$PLATFORM_ZIP" "$LATEST_PLATFORM_ZIP"
    
    echo "Created $PLATFORM_ZIP"
  done
else
  echo "No platform-specific binaries found, skipping platform-specific packaging"
fi

# Generate SHA1 checksums for all zip files (with just the filename, not the full path)
find "${OUTPUT_DIR}" -name "*.zip" -exec sh -c 'FILENAME=$(basename "$1"); sha1sum "$1" | sed "s| .*| $FILENAME|" > "$1.sha1"' _ {} \;

# Clean up
rm -rf "$BUILD_DIR"
echo "Plugin packaging complete! Packages are available in $OUTPUT_DIR" 
