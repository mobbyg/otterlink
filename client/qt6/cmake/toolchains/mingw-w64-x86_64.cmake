# Cross-compilation toolchain for 64-bit Windows using MinGW-w64.
#
# The Qt target installation is intentionally not hard-coded here. Supply
# CMAKE_PREFIX_PATH (or Qt6_ROOT) when configuring so the same repository can
# be used with different Qt installations.

set(CMAKE_SYSTEM_NAME Windows)
set(CMAKE_SYSTEM_PROCESSOR x86_64)

set(CMAKE_C_COMPILER x86_64-w64-mingw32-gcc)
set(CMAKE_CXX_COMPILER x86_64-w64-mingw32-g++)
set(CMAKE_RC_COMPILER x86_64-w64-mingw32-windres)

# Prefer target libraries and headers for dependency discovery.
set(CMAKE_FIND_ROOT_PATH
    /usr/x86_64-w64-mingw32
    /usr/x86_64-w64-mingw32ucrt
)

set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE BOTH)
