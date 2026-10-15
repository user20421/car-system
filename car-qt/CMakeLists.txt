cmake_minimum_required(VERSION 3.16)
project(car-qt VERSION 1.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)

find_package(Qt6 REQUIRED COMPONENTS Quick Network)

qt_standard_project_setup(REQUIRES 6.5)

qt_add_executable(car-qt
    src/main.cpp
    src/hal/IVehicleData.h
    src/hal/VehicleSimulator.h
    src/hal/VehicleSimulator.cpp
    src/net/ApiClient.h
    src/net/ApiClient.cpp
    src/core/VehicleController.h
    src/core/VehicleController.cpp
)

target_include_directories(car-qt PRIVATE src)

qt_add_qml_module(car-qt
    URI CarQt
    VERSION 1.0
    QML_FILES
        qml/Main.qml
        qml/Gauge.qml
        qml/BatteryBar.qml
)

target_link_libraries(car-qt PRIVATE Qt6::Quick Qt6::Network)
