cmake_minimum_required(VERSION 3.24)
project(AuroraCockpit VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)
set(CMAKE_AUTOUIC ON)

# QtMqtt is built as an add-on against the same Qt 6.11.2 kit.  Keep its
# relocatable install prefix ahead of any system packages; Qt Protobuf is part
# of the Qt kit and must not be confused with Google's Protobuf package.
list(PREPEND CMAKE_PREFIX_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../.tools/qtmqtt-install")
list(PREPEND CMAKE_PROGRAM_PATH "${CMAKE_CURRENT_SOURCE_DIR}/../.tools/protoc/bin")
# qtprotobufgen needs the protoc frontend only. Providing WrapProtoc directly
# avoids a misleading hard dependency on Google Protobuf libraries.
find_program(AURORA_PROTOC_EXECUTABLE NAMES protoc protoc.exe REQUIRED)
add_executable(WrapProtoc::WrapProtoc IMPORTED)
set_property(TARGET WrapProtoc::WrapProtoc PROPERTY IMPORTED_LOCATION "${AURORA_PROTOC_EXECUTABLE}")
find_package(Qt6 6.5 REQUIRED COMPONENTS Core Gui Qml Quick QuickControls2 Network Test Protobuf)
find_package(Qt6Mqtt 6.5 REQUIRED CONFIG)
qt_policy(SET QTP0001 NEW)
qt_policy(SET QTP0004 NEW)

set(AURORA_SOURCES
    src/core/VehicleTypes.h
    src/hal/IVehicleHardware.h
    src/hal/SimulatedVehicleHardware.h
    src/hal/SimulatedVehicleHardware.cpp
    src/controller/VehicleController.h
    src/controller/VehicleController.cpp
    src/concurrency/BoundedTaskQueue.h
    src/concurrency/BoundedTaskQueue.cpp
    src/storage/TelemetryStore.h
    src/storage/TelemetryStore.cpp
    src/network/MqttTransport.h
    src/network/MqttTransport.cpp
    src/network/VehicleProtobuf.h
    src/network/VehicleProtobuf.cpp
    src/network/DeviceNetwork.h
    src/network/DeviceNetwork.cpp
)

add_library(aurora_core STATIC ${AURORA_SOURCES})
target_include_directories(aurora_core PUBLIC src)
target_link_libraries(aurora_core PUBLIC Qt6::Core Qt6::Gui Qt6::Network Qt6::Qml
    Qt6::Mqtt Qt6::Protobuf)
qt_add_protobuf(aurora_core PROTO_FILES proto/vehicle_v1.proto)
if(MSVC)
    target_compile_options(aurora_core PRIVATE /W4 /permissive-)
else()
    target_compile_options(aurora_core PRIVATE -Wall -Wextra -Wpedantic)
endif()

qt_add_executable(aurora src/app/main.cpp)
target_link_libraries(aurora PRIVATE aurora_core Qt6::Quick Qt6::QuickControls2)

set(AURORA_QML_FILES
    qml/Main.qml
    qml/theme/Theme.qml
    qml/components/GlassPanel.qml
    qml/components/MetricTile.qml
    qml/components/ActionButton.qml
    qml/components/NavButton.qml
    qml/components/VehicleGraphic.qml
    qml/components/MapCanvas.qml
    qml/components/CircularGauge.qml
    qml/components/StatusPill.qml
    qml/components/SectionHeader.qml
    qml/pages/HomePage.qml
    qml/pages/ClusterPage.qml
    qml/pages/NavigationPage.qml
    qml/pages/MediaPage.qml
    qml/pages/ClimatePage.qml
    qml/pages/VehiclePage.qml
    qml/pages/NotificationsPage.qml
    qml/pages/SettingsPage.qml
    qml/pages/DebugPanel.qml
)
set_source_files_properties(qml/theme/Theme.qml PROPERTIES QT_QML_SINGLETON_TYPE TRUE)
qt_add_qml_module(aurora URI Aurora VERSION 1.0
    QML_FILES ${AURORA_QML_FILES}
    RESOURCES config/aurora.example.json
)

include(CTest)
if(BUILD_TESTING)
    qt_add_executable(aurora_tests
        tests/test_main.cpp
        tests/TestHal.h
        tests/TestController.h
        tests/TestConcurrency.h
        tests/TestNetwork.h
    )
    target_link_libraries(aurora_tests PRIVATE aurora_core Qt6::Test)
    add_test(NAME aurora_unit_tests COMMAND aurora_tests)
endif()

message(STATUS "Qt version: ${Qt6_VERSION}; QtMqtt: ${Qt6Mqtt_VERSION}; Qt Protobuf enabled")
