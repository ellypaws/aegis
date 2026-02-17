import { useState, useEffect, useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { GalleryGridCard } from "./GalleryGridCard";
import { SkeletonCard } from "./SkeletonCard";

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

import { Patterns } from "./Patterns";

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
    initialLoading,
    sortMode,
    onSortChange,
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
    initialLoading: boolean;
    sortMode: "id" | "date";
    onSortChange: (mode: "id" | "date") => void;
}) {
    const cols = useColumns();

    // Posts are already sorted by the backend; use them directly

    const rows = useMemo(() => {
        const r: Post[][] = [];
        for (let i = 0; i < posts.length; i += cols) {
            r.push(posts.slice(i, i + cols));
        }
        return r;
    }, [posts, cols]);

    return (
        <div className={cn("relative", UI.card)}>
            <div className="absolute inset-0 overflow-hidden rounded-[20px] pointer-events-none">
                <Patterns.Polka color="rgba(253, 224, 71, 0.15)" />
                <div className="absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white shadow-lg bg-yellow-400" />
                <div className="absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white shadow-lg bg-green-400" />
            </div>

            <div className="relative z-10 p-5">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                    <div>
                        <div className={cn(
                            "inline-block rounded-xl border-4 bg-white px-4 py-2 rotate-[-2deg]",
                            "border-yellow-400 shadow-[4px_4px_0px_rgba(250,204,21,1)]"
                        )}>
                            <div className="text-base font-black uppercase tracking-tight text-yellow-600">
                                Main Gallery
                            </div>
                        </div>
                        <div className="mt-3 text-sm font-bold text-zinc-600 dark:text-zinc-400">
                            Hover to expand details. Click to view.
                        </div>
                    </div>
                    <div className="flex items-end gap-3">
                        <div>
                            <div className={UI.label}>Sort</div>
                            <div className="flex overflow-hidden rounded-xl border-4 border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800">
                                <button
                                    type="button"
                                    onClick={() => onSortChange("id")}
                                    className={cn(
                                        "px-3 py-1.5 text-xs font-black uppercase tracking-wide transition-colors duration-200",
                                        sortMode === "id"
                                            ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                                            : "bg-white text-zinc-500 hover:bg-zinc-100 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700"
                                    )}
                                >
                                    Newest
                                </button>
                                <button
                                    type="button"
                                    onClick={() => onSortChange("date")}
                                    className={cn(
                                        "px-3 py-1.5 text-xs font-black uppercase tracking-wide transition-colors duration-200",
                                        sortMode === "date"
                                            ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                                            : "bg-white text-zinc-500 hover:bg-zinc-100 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700"
                                    )}
                                >
                                    By Date
                                </button>
                            </div>
                        </div>
                        <div className="w-full sm:w-64">
                            <div className={UI.label}>Search</div>
                            <input
                                value={q}
                                onChange={(e) => setQ(e.target.value)}
                                placeholder="title, description…"
                                className={UI.input}
                            />
                        </div>
                    </div>
                </div>

                <div className="mt-4 flex flex-col gap-4">
                    {initialLoading && posts.length === 0
                        ? /* Skeleton grid — 6 cards while first page loads */
                        Array.from({ length: Math.ceil(6 / cols) }).map((_, ri) => (
                            <div key={`skel-row-${ri}`} className="flex w-full gap-2">
                                {Array.from({ length: cols }).map((_, ci) => (
                                    <div key={`skel-${ri}-${ci}`} className="flex-1 min-w-0">
                                        <SkeletonCard />
                                    </div>
                                ))}
                            </div>
                        ))
                        : rows.map((rowPosts, i) => (
                            <div key={i} className="flex w-full gap-2">
                                {rowPosts.map((p) => (
                                    <div
                                        key={p.postKey}
                                        className={cn(
                                            "relative transition-[flex-grow] duration-500 ease-in-out overflow-hidden rounded-3xl",
                                            "flex-1 hover:grow-[2]",
                                            "min-w-0"
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

                {posts.length === 0 && !initialLoading && !loading ? (
                    <div className="mt-6 text-center text-sm font-bold text-zinc-500 dark:text-zinc-400">No results.</div>
                ) : null}

                {hasMore && !initialLoading ? (
                    <div className="mt-8 flex justify-center">
                        <button
                            onClick={onLoadMore}
                            disabled={loading}
                            className={cn(
                                UI.button,
                                UI.btnYellow,
                                "px-8 py-3 text-sm inline-flex items-center gap-2 transition-all duration-300",
                                loading && "opacity-60 cursor-not-allowed pointer-events-none"
                            )}
                        >
                            {loading && <span className="spinner" />}
                            {loading ? "Loading" : "Load More"}
                        </button>
                    </div>
                ) : null}
            </div>
        </div>
    );
}
