"use client";

/**
 * Web push for alerts, through the OneSignal Web SDK (v16, loaded from OneSignal's CDN on demand).
 * Each browser gets a random subscriber id, kept in localStorage, which it registers with
 * OneSignal as its external_id; alerts are saved under the same id and the crawler job pushes to
 * it. The app id is public; leave NEXT_PUBLIC_ONESIGNAL_APP_ID empty to hide alerts entirely.
 */

export const ONESIGNAL_APP_ID = process.env.NEXT_PUBLIC_ONESIGNAL_APP_ID ?? "";
export const SUBSCRIBER_KEY = "wc_push_subscriber";

interface OneSignalSDK {
  init(options: Record<string, unknown>): Promise<void>;
  login(externalId: string): Promise<void>;
  Notifications: {
    isPushSupported(): boolean;
    permission: boolean;
    requestPermission(): Promise<void>;
  };
  User: { PushSubscription: { optedIn?: boolean; optIn(): Promise<void> } };
}

declare global {
  interface Window {
    OneSignalDeferred?: Array<(sdk: OneSignalSDK) => void | Promise<void>>;
  }
}

/** The stored subscriber id, or null when this browser never saved an alert. */
export function existingSubscriberId(): string | null {
  try {
    return window.localStorage.getItem(SUBSCRIBER_KEY);
  } catch {
    return null;
  }
}

function subscriberId(): string {
  const existing = existingSubscriberId();
  if (existing) return existing;
  const id = crypto.randomUUID();
  try {
    window.localStorage.setItem(SUBSCRIBER_KEY, id);
  } catch {
    // Storage blocked: alerts still work for this page, but this browser cannot list them later.
  }
  return id;
}

function browserSupportsPush(): boolean {
  return typeof window !== "undefined" && "Notification" in window && "serviceWorker" in navigator && "PushManager" in window;
}

let sdk: Promise<OneSignalSDK> | null = null;

/** Loads and initializes the SDK once. Call it early (on mount) so a click can prompt right away. */
export function loadPush(): Promise<OneSignalSDK> | null {
  if (!ONESIGNAL_APP_ID || !browserSupportsPush()) return null;
  if (sdk) return sdk;
  sdk = new Promise<OneSignalSDK>((resolve, reject) => {
    window.OneSignalDeferred = window.OneSignalDeferred || [];
    window.OneSignalDeferred.push(async (os) => {
      try {
        await os.init({ appId: ONESIGNAL_APP_ID, serviceWorkerPath: "/OneSignalSDKWorker.js", allowLocalhostAsSecureOrigin: true });
        resolve(os);
      } catch (err) {
        reject(err);
      }
    });
    const script = document.createElement("script");
    script.src = "https://cdn.onesignal.com/sdks/web/v16/OneSignalSDK.page.js";
    script.defer = true;
    script.onerror = () => reject(new Error("OneSignal SDK failed to load"));
    document.head.appendChild(script);
  });
  sdk.catch(() => {
    sdk = null;
  });
  return sdk;
}

export type PushStatus = "granted" | "denied" | "unsupported";

/** Asks for notification permission (call from a click) and links this browser to its subscriber id. */
export async function enablePush(): Promise<{ status: PushStatus; subscriberId?: string }> {
  if (!browserSupportsPush()) return { status: "unsupported" };
  if (Notification.permission === "denied") return { status: "denied" };
  const loading = loadPush();
  if (!loading) return { status: "unsupported" };
  const os = await loading;
  if (!os.Notifications.isPushSupported()) return { status: "unsupported" };
  if (!os.Notifications.permission) await os.Notifications.requestPermission();
  if (!os.Notifications.permission) return { status: "denied" };
  const id = subscriberId();
  await os.login(id);
  if (!os.User.PushSubscription.optedIn) await os.User.PushSubscription.optIn();
  return { status: "granted", subscriberId: id };
}
