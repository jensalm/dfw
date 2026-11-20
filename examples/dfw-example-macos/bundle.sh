#!/bin/sh
# packs the fixture's binary into dfw-example-macos.app beside this script: a
# menu bar application (LSUIElement), signed ad hoc so it runs on the machine
# that built it.
#
# the binary itself is the bundle's executable. a shell script in that place,
# even one that only execs the binary, leaves the tray's status item unshown
# on macOS 26: the process never appears in the menu bar.
set -eu
here=$(cd "$(dirname "$0")" && pwd)
name=dfw-example-macos
app="$here/$name.app"

CGO_ENABLED=1 go build -C "$here" -o "$here/$name" .
rm -rf "$app"
mkdir -p "$app/Contents/MacOS"
cp "$here/$name" "$app/Contents/MacOS/$name"
cat > "$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleIdentifier</key><string>com.quigley.dfw.example.macos</string>
	<key>CFBundleName</key><string>$name</string>
	<key>CFBundleExecutable</key><string>$name</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>0.0.0</string>
	<key>CFBundleVersion</key><string>0</string>
	<key>LSMinimumSystemVersion</key><string>12.0</string>
	<key>LSUIElement</key><true/>
	<key>NSHighResolutionCapable</key><true/>
</dict>
</plist>
EOF
plutil -lint "$app/Contents/Info.plist" >/dev/null
codesign --force --sign - "$app"
codesign --verify --strict "$app"
echo "built '$app'"
