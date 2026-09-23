"use client";

import { useEffect } from "react";
import { startAnalytics, trackSearch, trackViewListing, type SearchEvent, type ViewListingEvent } from "@/lib/analytics";

/** Mounted once in the layout so a session starts on any page. */
export function Analytics() {
  useEffect(() => {
    startAnalytics();
  }, []);
  return null;
}

/** Fires one `search` event per distinct results page (the key changes with the query string). */
export function TrackSearch({ event }: { event: SearchEvent }) {
  const key = JSON.stringify(event);
  useEffect(() => {
    trackSearch(JSON.parse(key) as SearchEvent);
  }, [key]);
  return null;
}

/** Fires one `view_listing` event per listing page. */
export function TrackListingView({ event }: { event: ViewListingEvent }) {
  const key = JSON.stringify(event);
  useEffect(() => {
    trackViewListing(JSON.parse(key) as ViewListingEvent);
  }, [key]);
  return null;
}
