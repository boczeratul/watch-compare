import type { Listing } from "@/lib/api";
import { ListingCard } from "./listing-card";

export function ListingGrid({ items }: { items: Listing[] }) {
  return (
    <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:gap-4">
      {items.map((l) => (
        <li key={l.id}>
          <ListingCard listing={l} />
        </li>
      ))}
    </ul>
  );
}
