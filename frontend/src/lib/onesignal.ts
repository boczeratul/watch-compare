/**
 * OneSignal app id for web push alerts. It is public client configuration (like the Amplitude
 * key); override per environment with NEXT_PUBLIC_ONESIGNAL_APP_ID, or set it empty to hide alerts.
 */
export const ONESIGNAL_APP_ID = process.env.NEXT_PUBLIC_ONESIGNAL_APP_ID ?? "6e12337f-212f-4ec3-9c38-01e83635d2fe";
