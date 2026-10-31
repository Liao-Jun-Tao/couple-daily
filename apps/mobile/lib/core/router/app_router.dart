import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:couple_daily/core/models/models.dart';
import 'package:couple_daily/core/session/app_session.dart';
import 'package:couple_daily/features/anniversary/anniversary_form_page.dart';
import 'package:couple_daily/features/anniversary/anniversary_list_page.dart';
import 'package:couple_daily/features/auth_pair/auth_page.dart';
import 'package:couple_daily/features/auth_pair/invite_page.dart';
import 'package:couple_daily/features/auth_pair/pairing_done_page.dart';
import 'package:couple_daily/features/auth_pair/splash_page.dart';
import 'package:couple_daily/features/compose/compose_page.dart';
import 'package:couple_daily/features/compose/location_picker_page.dart';
import 'package:couple_daily/features/me/me_page.dart';
import 'package:couple_daily/features/restaurant/favorites_stub_page.dart';
import 'package:couple_daily/features/restaurant/restaurant_page.dart';
import 'package:couple_daily/features/timeline/timeline_page.dart';
import 'package:couple_daily/shared/widgets/home_shell.dart';

GoRouter buildRouter(AppSession session) {
  final rootKey = GlobalKey<NavigatorState>(debugLabel: 'root');
  return GoRouter(
    navigatorKey: rootKey,
    initialLocation: '/splash',
    refreshListenable: session,
    redirect: (context, state) => _redirect(session, state.uri.path),
    routes: [
      GoRoute(path: '/splash', builder: (context, state) => const SplashPage()),
      GoRoute(path: '/auth', builder: (context, state) => const AuthPage()),
      GoRoute(path: '/invite', builder: (context, state) => const InvitePage()),
      GoRoute(
        path: '/pairing-done',
        builder: (context, state) => const PairingDonePage(),
      ),
      GoRoute(
        path: '/compose',
        parentNavigatorKey: rootKey,
        builder: (context, state) => const ComposePage(),
      ),
      GoRoute(
        path: '/location-picker',
        parentNavigatorKey: rootKey,
        builder: (context, state) {
          final extra = state.extra;
          final initial = extra is LocationInput ? extra : null;
          return LocationPickerPage(initial: initial);
        },
      ),
      GoRoute(
        path: '/favorites',
        parentNavigatorKey: rootKey,
        builder: (context, state) => const FavoritesStubPage(),
      ),
      GoRoute(
        path: '/anniversary/new',
        parentNavigatorKey: rootKey,
        builder: (context, state) => const AnniversaryFormPage(),
      ),
      GoRoute(
        path: '/anniversary/edit/:id',
        parentNavigatorKey: rootKey,
        builder: (context, state) {
          return AnniversaryFormPage(anniversaryId: state.pathParameters['id']);
        },
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, navigationShell) {
          return HomeShell(navigationShell: navigationShell);
        },
        branches: [
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/timeline',
                builder: (context, state) => const TimelinePage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/restaurant',
                builder: (context, state) => const RestaurantPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/anniversary',
                builder: (context, state) => const AnniversaryListPage(),
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/me',
                builder: (context, state) => const MePage(),
              ),
            ],
          ),
        ],
      ),
    ],
  );
}

String? _redirect(AppSession session, String path) {
  if (!session.ready) {
    return path == '/splash' ? null : '/splash';
  }
  if (!session.isLoggedIn) {
    return path == '/auth' ? null : '/auth';
  }
  if (!session.isPaired) {
    if (path == '/invite' || path == '/pairing-done') return null;
    return '/invite';
  }
  if (session.pendingPairingDone) {
    return path == '/pairing-done' ? null : '/pairing-done';
  }
  if (path == '/splash' || path == '/auth' || path == '/invite') {
    return '/timeline';
  }
  return null;
}
