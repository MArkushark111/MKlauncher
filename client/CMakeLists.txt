cmake_minimum_required(VERSION 3.16)
project(MKLauncher VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)
set(CMAKE_AUTOUIC ON)

find_package(Qt6 REQUIRED COMPONENTS Widgets Network Sql)

set(SOURCES
    src/main.cpp
    src/launcherwindow.cpp
    src/gamegrid.cpp
    src/downloadmanager.cpp
    src/archiveextractor.cpp
    src/updater.cpp
    src/hydralinks.cpp
)

set(HEADERS
    src/launcherwindow.h
    src/gamegrid.h
    src/downloadmanager.h
    src/archiveextractor.h
    src/uninstaller.h
    src/localdb.h
    src/updater.h
    src/settings.h
    src/theme.h
    src/hydralinks.h
)

if(WIN32)
    set(RESOURCES app.rc)
endif()

add_executable(${PROJECT_NAME} WIN32 ${SOURCES} ${HEADERS} ${RESOURCES})

target_include_directories(${PROJECT_NAME} PRIVATE src)

target_link_libraries(${PROJECT_NAME} PRIVATE
    Qt6::Widgets
    Qt6::Network
    Qt6::Sql
)

install(TARGETS ${PROJECT_NAME} DESTINATION /opt/mkgames)
