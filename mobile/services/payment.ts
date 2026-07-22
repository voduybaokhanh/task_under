import { initPaymentSheet, presentPaymentSheet } from '@stripe/stripe-react-native';

import { apiService } from './api';

/** Card payments are only available when the app is built with a Stripe key. */
export const stripePublishableKey = process.env.EXPO_PUBLIC_STRIPE_KEY ?? '';
export const cardPaymentsEnabled = stripePublishableKey !== '';

/**
 * Collects a card for a task's escrow hold. The backend already created the
 * PaymentIntent (manual capture) when the task was created; here the owner
 * attaches a card to it, which is what makes the hold real.
 *
 * Returns false when the user cancels, or when card payments are switched off.
 */
export async function payForTask(taskId: string): Promise<boolean> {
  if (!cardPaymentsEnabled) {
    return false;
  }

  const { client_secret } = await apiService.getPaymentIntent(taskId);
  if (!client_secret) {
    return false;
  }

  const init = await initPaymentSheet({
    merchantDisplayName: 'Task Underground',
    paymentIntentClientSecret: client_secret,
  });
  if (init.error) {
    throw new Error(init.error.message);
  }

  const result = await presentPaymentSheet();
  if (result.error) {
    // Canceling is a normal outcome, not a failure to report.
    if (result.error.code === 'Canceled') {
      return false;
    }
    throw new Error(result.error.message);
  }

  return true;
}
