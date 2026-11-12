import 'dart:io';
import 'package:dio/dio.dart';
import 'package:native_dio_adapter/native_dio_adapter.dart';

void configureAdapter(Dio dio) {
  // Android uses Cronet and iOS uses URLSession; both negotiate HTTP/2 over TLS.
  // Keep Dio's default adapter on desktop and in host-side tests.
  if (Platform.isAndroid || Platform.isIOS) {
    dio.httpClientAdapter = NativeAdapter();
  }
}
