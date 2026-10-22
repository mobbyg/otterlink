cmake_minimum_required(VERSION 3.21)
project(otterlink-qt6 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTOUIC ON)
set(CMAKE_AUTORCC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Multimedia Network Widgets)

# Qt WebEngine is enabled when this platform's Qt installation provides it.
# Keep it optional so the Windows MinGW cross-build continues to use QTextBrowser.
if(NOT WIN32)
    find_package(Qt6 QUIET COMPONENTS WebEngineWidgets)
endif()

qt_add_executable(otterlink
    main.cpp
    mainwindow.cpp
    mainwindow.h
    mainwindow.ui
    otterhomepage.cpp
    otterhomepage.h
    otterlinkclient.cpp
    otterlinkclient.h
    otterboardsclient.cpp
    otterlinkstyle.cpp
    otterlinkstyle.h
    otterchatwidget.cpp
    otterchatwidget.h
    otterboardswidget.cpp
    otterboardswidget.h
    otterpeoplewidget.cpp
    otterpeoplewidget.h
    otterdmwidget.cpp
    otterdmwidget.h
    ottereventswidget.cpp
    ottereventswidget.h
    otternewswidget.cpp
    otternewswidget.h
    otterwebwidget.cpp
    otterwebwidget.h
    otterservicewindow.cpp
    otterservicewindow.h
    otterkeywordwidget.cpp
    otterkeywordwidget.h
    resources/otterlink.qrc
)

target_link_libraries(otterlink PRIVATE Qt6::Core Qt6::Multimedia Qt6::Network Qt6::Widgets)

if(TARGET Qt6::WebEngineWidgets)
    target_link_libraries(otterlink PRIVATE Qt6::WebEngineWidgets)
    target_compile_definitions(otterlink PRIVATE OTTERLINK_USE_WEBENGINE)
    message(STATUS "OtterWeb: Qt WebEngine enabled")
else()
    message(STATUS "OtterWeb: Qt WebEngine unavailable; using QTextBrowser fallback")
endif()

if (WIN32)
    set_target_properties(otterlink PROPERTIES WIN32_EXECUTABLE TRUE)
endif()

if (APPLE)
    set_target_properties(otterlink PROPERTIES MACOSX_BUNDLE TRUE)
endif()
