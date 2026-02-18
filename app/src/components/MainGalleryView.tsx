import { useState, useEffect, useMemo, useRef } from "react";
import { cn } from "../lib/utils";
import { Eye, EyeOff } from "lucide-react";
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
    onOpenPost: (id: string) => void;
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
    const [showLocked, setShowLocked] = useState(true);
    const [isSticky, setIsSticky] = useState(false);
    const sentinelRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const observer = new IntersectionObserver(
            ([entry]) => {
                setIsSticky(!entry.isIntersecting);
            },
            { threshold: [0, 1], rootMargin: "0px" }
        );

        if (sentinelRef.current) {
            observer.observe(sentinelRef.current);
        }

        return () => observer.disconnect();
    }, []);

    const hasLockedPosts = useMemo(() => {
        return posts.some((p) => !canAccessPost(p));
    }, [posts, canAccessPost]);

    useEffect(() => {
        if (!hasLockedPosts && !showLocked) {
            setShowLocked(true);
        }
    }, [hasLockedPosts, showLocked]);

    const filteredPosts = useMemo(() => {
        if (showLocked) return posts;
        return posts.filter((p) => canAccessPost(p));
    }, [posts, showLocked, canAccessPost]);

    // Posts are already sorted by the backend; use them directly

    const rows = useMemo(() => {
        const r: Post[][] = [];
        for (let i = 0; i < filteredPosts.length; i += cols) {
            r.push(filteredPosts.slice(i, i + cols));
        }
        return r;
    }, [filteredPosts, cols]);

    return (
        <div className={cn("relative", UI.card, "backdrop-blur-sm")}>
            <div className="absolute inset-0 overflow-hidden rounded-[20px] pointer-events-none">
                <Patterns.Polka color="rgba(253, 224, 71, 0.15)" />
                <div className="absolute bottom-[-10px] right-[-10px] h-20 w-20 rotate-12 border-4 border-white shadow-lg bg-green-400" />
            </div>

            <div className="relative z-10">
                <div ref={sentinelRef} className="absolute top-0 h-px w-full pointer-events-none opacity-0" />
                <div className={cn(
                    "sticky top-0 z-50 px-5 pt-5 pb-4 -mx-0.5 mb-4 transition-all duration-300 ease-in-out overflow-hidden rounded-t-[20px]",
                    isSticky
                        ? "bg-white/60 dark:bg-zinc-900/60 backdrop-blur-md shadow-[0_8px_30px_rgb(0,0,0,0.04)] border-b border-white/20 rounded-b-[20px] mt-[-1px]"
                        : "bg-transparent translate-y-0"
                )}>
                    <div className={cn(
                        "absolute inset-0 pointer-events-none transition-opacity duration-300",
                        isSticky ? "opacity-100" : "opacity-0"
                    )}>
                        <Patterns.Polka color="rgba(253, 224, 71, 0.15)" />
                    </div>
                    <div className="absolute top-[-16px] left-[-16px] h-24 w-24 rounded-full border-4 border-white shadow-lg bg-yellow-400 pointer-events-none" />
                    <div className={cn(
                        "relative z-10 flex gap-3 sm:flex-row sm:items-end sm:justify-between",
                        isSticky ? "flex-row items-center justify-between" : "flex-col"
                    )}>
                        <div>
                            <div className={cn(
                                "inline-block rounded-xl border-4 bg-white px-4 py-2 rotate-[-2deg]",
                                "border-yellow-400 shadow-[4px_4px_0px_rgba(250,204,21,1)]"
                            )}>
                                <div className="text-base font-black uppercase tracking-tight text-yellow-600">
                                    Main Gallery
                                </div>
                            </div>
                            <div className="mt-6 text-sm font-bold text-zinc-600 dark:text-zinc-400" />
                        </div>
                        <div className="flex items-end gap-3">
                            <div className={cn("transition-all duration-300", isSticky && "hidden sm:block")}>
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
                            {hasLockedPosts && (
                                <div>
                                    <div className={UI.label}>View</div>
                                    <div className="flex overflow-hidden rounded-xl border-4 border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-800">
                                        <button
                                            type="button"
                                            onClick={() => setShowLocked(!showLocked)}
                                            className={cn(
                                                "flex h-[28px] w-[36px] items-center justify-center transition-colors duration-200",
                                                showLocked
                                                    ? "bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900"
                                                    : "bg-white text-zinc-500 hover:bg-zinc-100 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700"
                                            )}
                                            title={showLocked ? "Hide locked posts" : "Show locked posts"}
                                        >
                                            {showLocked ? <Eye className="h-4 w-4" /> : <EyeOff className="h-4 w-4" />}
                                        </button>
                                    </div>
                                </div>
                            )}
                            <div className={cn("w-full sm:w-64 transition-all duration-300", isSticky && "hidden sm:block")}>
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
                </div>

                <div className="px-5 pb-5 flex flex-col gap-4">
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
                                            onOpen={() => {
                                                onOpenPost(p.postKey);
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

                {filteredPosts.length === 0 && !initialLoading && !loading ? (
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
