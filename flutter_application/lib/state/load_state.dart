import '../services/repository.dart';

enum LoadStatus { loading, ready, empty, error, offline, unauthorized }

class LoadState<T> {
  const LoadState._(this.status, this.data, this.message);
  const LoadState.loading() : this._(LoadStatus.loading, null, null);
  const LoadState.ready(T data) : this._(LoadStatus.ready, data, null);
  const LoadState.empty([T? data]) : this._(LoadStatus.empty, data, null);
  const LoadState.error(String message) : this._(LoadStatus.error, null, message);
  const LoadState.offline([T? data]) : this._(LoadStatus.offline, data, null);
  const LoadState.unauthorized() : this._(LoadStatus.unauthorized, null, null);
  final LoadStatus status;
  final T? data;
  final String? message;

  factory LoadState.failure(Object error) {
    if (error is RepositoryException) {
      return switch (error.kind) {
        RepositoryErrorKind.offline => const LoadState.offline(),
        RepositoryErrorKind.unauthorized => const LoadState.unauthorized(),
        _ => LoadState.error(error.message),
      };
    }
    return const LoadState.error('暂时无法加载，请稍后重试');
  }
}
