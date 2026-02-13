import React from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Post } from "../types";
import { GalleryGridCard } from "./GalleryGridCard";

export function MainGalleryView({
    posts,
    selectedId,
    canAccessPost,
    onOpenPost,
    q,
    setQ,
}: {
    posts: Post[];
    selectedId: string | null;
    canAccessPost: (p: Post) => boolean;
    onOpenPost: (id: string, rect?: DOMRect) => void;
    q: string;
    setQ: (s: string) => void;
}) {
    return (
        <div className={cn("p-4", UI.card)}>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <div>
                    <div className={UI.sectionTitle}>Main Gallery</div>
                    <div className="mt-1 text-xs font-bold text-zinc-400">
                        Hover to expand details. Click to view.
                    </div>
                </div>
                <div className="w-full sm:w-80">
                    <div className={UI.label}>Search</div>
                    <input
                        value={q}
                        onChange={(e) => setQ(e.target.value)}
                        placeholder="title, tag, description…"
                        className={UI.input}
                    />
                </div>
            </div>

            <div className="mt-4 flex flex-wrap gap-2 group">
                {posts.map((p) => (
                    <div
                        key={p.id}
                        className={cn(
                            "relative transition-[width] duration-500 ease-in-out overflow-hidden rounded-3xl",
                            "w-64", // Base width
                            "group-hover:w-48", // Siblings shrink
                            "hover:!w-96" // Hovered expands
                        )}
                    >
                        <GalleryGridCard
                            post={p}
                            canAccess={canAccessPost(p)}
                            selected={p.id === selectedId}
                            onOpen={(e) => {
                                // Try to find the image within the button to get its rect
                                const img = (e.currentTarget as HTMLElement).querySelector("img");
                                const rect = img?.getBoundingClientRect();
                                onOpenPost(p.id, rect);
                            }}
                            variant="flexible"
                        />
                    </div>
                ))}
            </div>

            {posts.length === 0 ? <div className="mt-6 text-center text-sm font-bold text-zinc-500">No results.</div> : null}
        </div>
    );
}
