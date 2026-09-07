"use client";

import { useState } from "react";
import { cn } from "@/lib/utils";

export function ImageGallery({ images, alt }: { images: string[]; alt: string }) {
  const [active, setActive] = useState(0);
  const main = images[active];
  return (
    <div>
      <div className="aspect-square w-full overflow-hidden rounded-lg border border-slate-200 bg-slate-50">
        {main ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={main} alt={alt} referrerPolicy="no-referrer" className="h-full w-full object-contain p-4" />
        ) : (
          <div className="flex h-full items-center justify-center text-slate-300">—</div>
        )}
      </div>
      {images.length > 1 && (
        <ul className="mt-3 flex gap-2 overflow-x-auto">
          {images.map((src, i) => (
            <li key={src + i}>
              <button type="button" onClick={() => setActive(i)} className={cn("h-16 w-16 overflow-hidden rounded-md border bg-white", i === active ? "border-slate-900" : "border-slate-200 hover:border-slate-400")} aria-label={`${alt} ${i + 1}`}>
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={src} alt="" referrerPolicy="no-referrer" loading="lazy" className="h-full w-full object-contain" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
