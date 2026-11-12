import 'package:dio/browser.dart';
import 'package:dio/dio.dart';

void configureAdapter(Dio dio) {
  // The browser owns ALPN negotiation and HttpOnly cookie handling.
  dio.httpClientAdapter = BrowserHttpClientAdapter()..withCredentials = true;
}
