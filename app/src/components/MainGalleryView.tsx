import { useState, useEffect, useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { GalleryGridCard } from "./GalleryGridCard";

function useColumns() {
    const [cols, setCols] = useState(3);
    useEffect(() => {
        const handleResize = () => {
            if (window.innerWidth < 640) setCols(1);
            else if (window.innerWidth < 1024) setCols(2);
            else setCols(3);
        };
        handleResize();
        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, []);
    return cols;
}

export function MainGalleryView({
    posts,
    selectedId,
    canAccessPost,
    onOpenPost,
    q,
    setQ,
    onLoadMore,
    hasMore,
    loading,
}: {
    posts: Post[];
    selectedId: string | null;
    canAccessPost: (p: Post) => boolean;
    onOpenPost: (id: string, rect?: DOMRect) => void;
    q: string;
    setQ: (s: string) => void;
    onLoadMore: () => void;
    hasMore: boolean;
    loading: boolean;
}) {
    const cols = useColumns();
    const rows = useMemo(() => {
        const r: Post[][] = [];
        for (let i = 0; i < posts.length; i += cols) {
            r.push(posts.slice(i, i + cols));
        }
        return r;
    }, [posts, cols]);

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
                        placeholder="title, description…"
                        className={UI.input}
                    />
                </div>
            </div>

            <div className="mt-4 flex flex-col gap-4">
                {rows.map((rowPosts, i) => (
                    <div key={i} className="flex w-full gap-2">
                        {rowPosts.map((p) => (
                            <div
                                key={p.postKey}
                                className={cn(
                                    "relative transition-[flex-grow] duration-500 ease-in-out overflow-hidden rounded-3xl",
                                    "flex-1 hover:grow-[2]",
                                    "min-w-0" // prevent flex overflow
                                )}
                            >
                                <GalleryGridCard
                                    post={p}
                                    canAccess={canAccessPost(p)}
                                    selected={p.postKey === selectedId}
                                    onOpen={(e) => {
                                        const img = (e.currentTarget as HTMLElement).querySelector("img");
                                        const rect = img?.getBoundingClientRect();
                                        onOpenPost(p.postKey, rect);
                                    }}
                                    variant="flexible"
                                />
                            </div>
                        ))}
                        {/* Fillers for last row to keep alignment */}
                        {Array.from({ length: cols - rowPosts.length }).map((_, idx) => (
                            <div key={`filler-${idx}`} className="flex-1 invisible" />
                        ))}
                    </div>
                ))}
            </div>

            {posts.length === 0 && !loading ? <div className="mt-6 text-center text-sm font-bold text-zinc-500">No results.</div> : null}

            {hasMore ? (
                <div className="mt-8 flex justify-center">
                    <button
                        onClick={onLoadMore}
                        disabled={loading}
                        className={cn(UI.button, UI.btnYellow, "px-8 py-3 text-sm", loading && "opacity-50 cursor-not-allowed")}
                    >
                        {loading ? "Loading..." : "Load More"}
                    </button>
                </div>
            ) : null}
        </div>
    );
}
