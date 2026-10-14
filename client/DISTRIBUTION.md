# OtterLink Client Distribution Guide

This guide explains how to package the native Qt 6 desktop client so people can run it **without installing Qt 6 or a development environment**.

The Qt runtime and platform plugins must be shipped with each release. Build and package separately for each operating system and CPU architecture. Use the deployment tool that belongs to the *same Qt installation and compiler kit* used to build the client.

> **Important:** A successful build is not automatically a distributable application. Test the packaged output on a clean machine or virtual machine that does not have Qt installed.

## General release checklist

1. Start from a clean, successful Release build.
2. Package the executable together with the required Qt libraries, platform plugins, and any non-system runtime libraries.
3. Include the OtterLink license/readme and version information with the release.
4. Test login, chat, audio/media features if enabled, HTTPS certificate validation, and any other enabled functionality from the packaged copy.
5. Never distribute debug builds or development-only files.
6. Publish a separate package for each supported OS/architecture and clearly label it with the version.

The app's server field should continue to let users select their OtterLink server. Do not bundle server credentials, private keys, or local development configuration.

## Windows

Use a Release build made with the intended Windows Qt kit (for example, Qt 6 with MinGW 11.2.0 if that is the kit used to compile the executable). **Do not mix MSVC and MinGW Qt builds or mix different Qt versions.**

### 1. Build Release

From the Windows build directory, configure/build in Release mode using the same Qt kit that will provide `windeployqt`. For a CMake build tree that is already configured for the correct kit:

```powershell
cmake --build . --config Release
```

If the project uses a single-configuration MinGW build directory, make sure it was configured with `-DCMAKE_BUILD_TYPE=Release` before building.

### 2. Create a clean package directory

Create a new empty directory for the release, then copy the built `otterlink.exe` into it. Do not package the entire build directory.

### 3. Deploy Qt dependencies

Open the terminal/environment supplied by the matching Qt kit, or use the full path to that kit's `windeployqt.exe`. Run it against the copied executable. For example:

```powershell
windeployqt --release --compiler-runtime C:\path\to\package\otterlink.exe
```

Run the command from the package directory or pass the full executable path. Replace the example path with the actual package path. If the tool is not on PATH, invoke the `windeployqt.exe` that belongs to the exact Qt kit used for the build.

The deployment tool collects the Qt DLLs and the platform plugins it detects. The package may need plugins for Windows display integration, image formats, multimedia, and TLS depending on the features actually used. Verify the resulting directory instead of assuming every optional plugin was included.

### 4. Verify compiler runtime and TLS

- For a MinGW build, ensure the required MinGW runtime DLLs are beside the executable (the `--compiler-runtime` option asks `windeployqt` to copy them when supported).
- For an MSVC build, follow the matching MSVC runtime deployment requirements.
- Confirm the Qt TLS backend and the required Windows certificate support are present. Test an HTTPS server on a clean Windows system.
- Do not copy DLLs from an unrelated Qt installation or from an arbitrary system directory.

### 5. Zip and test

Zip the **contents** of the package directory so users can extract it and run `otterlink.exe`. Test on a clean Windows machine without Qt installed. Launch from the extracted directory and test login, chat, image handling, media features if enabled, and HTTPS.

## Linux

A Linux build generally needs both the Qt runtime and compatible system libraries. Copying the executable alone is not enough.

- For a simple folder package on a compatible distribution, include the executable, the Qt shared libraries it needs, and the required Qt platform/plugin directories. Check dynamic dependencies with tools such as `ldd`, and verify the plugin paths.
- A folder package that relies on libraries already installed on the target distribution is **not** a fully self-contained package.
- For a broadly distributable download, choose and maintain a packaging format such as AppImage, Flatpak, or distribution-specific packages. Build/test against the compatibility baseline you intend to support; AppImage does not make every system dependency portable automatically.
- Do not bundle core operating-system components indiscriminately. Keep the package aligned with the chosen distribution format and its rules.

Test the final package in a clean VM/test environment that has no Qt development packages installed, and test HTTPS/TLS as well as normal application behavior.

## macOS

Use the Qt deployment utility from the same Qt installation used to build the app.

1. Build the intended Release configuration.
2. Run `macdeployqt` against the generated `OtterLink.app` bundle, for example:

```sh
macdeployqt /path/to/OtterLink.app
```

3. Inspect the app bundle and test it on a Mac without Qt installed.
4. For public distribution, plan for Apple Developer ID code signing and notarization, and test the downloaded/notarized artifact. Follow Apple's current requirements for the release channel you choose.

## Release verification

Before publishing, confirm all of the following:

- [ ] The package launches on a machine without Qt installed.
- [ ] No build-tree paths or developer-only configuration are required.
- [ ] Login and the configured server URL work.
- [ ] HTTPS/TLS validation works against the intended server.
- [ ] Required platform, image, network, multimedia, and TLS plugins are included when those features are used.
- [ ] The package contains the license and release notes.
- [ ] The archive extracts into a clear, self-contained folder (or the selected platform installer/package format works as intended).
- [ ] The release version, OS, architecture, and build type are identified.

## Keep in mind

The exact files required depend on the Qt modules linked by the current client and the features enabled at build time. Treat the deployment tool's output as a starting point, then verify the final package on a clean target system. Re-run the packaging and testing steps for every release build.
