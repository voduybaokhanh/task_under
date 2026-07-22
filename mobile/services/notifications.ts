import { Platform } from 'react-native';
import * as Notifications from 'expo-notifications';

import { apiService } from './api';

// Show notifications that arrive while the app is in the foreground; without
// this Expo silently swallows them.
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

/**
 * Asks for notification permission, then hands the Expo push token to the
 * backend so it can reach this device while the app is closed.
 *
 * Safe to call on every launch: it is a no-op on web/simulator and when the
 * user has already declined.
 */
export async function registerForPushNotifications(): Promise<string | null> {
  if (Platform.OS === 'web') {
    return null;
  }

  const existing = await Notifications.getPermissionsAsync();
  let status = existing.status;
  if (status !== 'granted') {
    status = (await Notifications.requestPermissionsAsync()).status;
  }
  if (status !== 'granted') {
    return null;
  }

  if (Platform.OS === 'android') {
    await Notifications.setNotificationChannelAsync('default', {
      name: 'default',
      importance: Notifications.AndroidImportance.DEFAULT,
    });
  }

  try {
    const { data: token } = await Notifications.getExpoPushTokenAsync();
    await apiService.updatePushToken(token);
    return token;
  } catch (error) {
    // A device without a push service (simulator, no Google Play) must not
    // break app startup.
    console.warn('Push registration failed', error);
    return null;
  }
}
