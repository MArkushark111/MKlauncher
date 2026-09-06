cmake_minimum_required(VERSION 3.16)
project(MKGamesAdmin VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Widgets)

add_executable(MKGamesAdmin WIN32 main.cpp)
target_link_libraries(MKGamesAdmin PRIVATE Qt6::Widgets)

install(DIRECTORY web/ DESTINATION share/mkgames/admin)
install(TARGETS MKGamesAdmin DESTINATION bin)