cmake_minimum_required(VERSION 3.21)
project(otterlink-qt6 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTOUIC ON)
set(CMAKE_AUTORCC ON)

find_package(Qt6 REQUIRED COMPONENTS Core Network Widgets)

qt_add_executable(otterlink
    main.cpp
    mainwindow.cpp
    mainwindow.h
    mainwindow.ui
    otterlinkclient.cpp
    otterlinkclient.h
    otterlinkstyle.cpp
    otterlinkstyle.h
)

target_link_libraries(otterlink PRIVATE Qt6::Core Qt6::Network Qt6::Widgets)

if (WIN32)
    set_target_properties(otterlink PROPERTIES WIN32_EXECUTABLE TRUE)
endif()

if (APPLE)
    set_target_properties(otterlink PROPERTIES MACOSX_BUNDLE TRUE)
endif()
