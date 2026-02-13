import React, { useLayoutEffect, useRef, useState } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";
import { Tag } from "./Tag";
import { getRoleName } from "../data/mock";

export function PostDetailView({
    selected,
    canAccessSelected,
    accessLabel,
    onBack,
    tagFilter,
    setTagFilter,
    similarPosts,
    canAccessPost,
    onSelectSimilar,
    transitionRect,
}: {
    selected: Post | null;
    canAccessSelected: boolean;
    accessLabel: string;
    onBack: () => void;
    tagFilter: string | null;
    setTagFilter: (t: string | null) => void;
    similarPosts: Post[];
    canAccessPost: (p: Post) => boolean;
    onSelectSimilar: (id: string) => void;
    transitionRect?: DOMRect | null;
}) {
    const imgRef = useRef<HTMLImageElement>(null);
    const [animating, setAnimating] = useState(false);
    const [fixedState, setFixedState] = useState<{
        top: number;
        left: number;
        width: number;
        height: number;
    } | null>(null);

    useLayoutEffect(() => {
        if (!transitionRect || !selected) return;

        setAnimating(true);
        setFixedState({
            top: transitionRect.top,
            left: transitionRect.left,
            width: transitionRect.width,
            height: transitionRect.height,
        });

        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                if (!imgRef.current) {
                    setAnimating(false);
                    return;
                }

                const targetRect = imgRef.current.getBoundingClientRect();

                setFixedState({
                    top: targetRect.top,
                    left: targetRect.left,
                    width: targetRect.width,
                    height: targetRect.height,
                });

                setTimeout(() => {
                    setAnimating(false);
                    setFixedState(null);
                }, 500);
            });
        });
    }, [transitionRect, selected]);

    return (
        <div className={cn("p-4", UI.card)}>
            {animating && fixedState && selected ? (
                <div
                    className="fixed z-50 pointer-events-none transition-all duration-500 ease-[cubic-bezier(0.25,1,0.5,1)]"
                    style={{
                        top: fixedState.top,
                        left: fixedState.left,
                        width: fixedState.width,
                        height: fixedState.height,
                    }}
                >
                    {/* Crossfade strategy:
                 1. Background image (Cover): Matches the thumbnail crop. Fades out.
                 2. Foreground image (Contain): Matches the target full view. Fades in.
             */}
                    <img
                        src={canAccessSelected ? selected.full.url : selected.thumb?.url}
                        className={cn(
                            "absolute inset-0 h-full w-full object-cover rounded-2xl shadow-xl bg-white transition-opacity duration-500",
                            !canAccessSelected && selected.thumb && "blur-md",
                            "opacity-0" // Fades out effectively by being covered or explicit opacity?
                            // Actually, if we transition `opacity-0` here, we need state.
                            // But we used CSS transition on the container. We can use animation keyframes or just rely on the fact that `animating` is true.
                            // Wait, we don't have a "start" vs "end" state variable for opacity inside the component unless we add one.
                            // Simplification: Just keep it opacity-100? No, then it stays cropped.
                            // Let's use a CSS animation or simply rely on the fact that we can't easily add a second state without `useEffect` delay.
                            // But we DO have a timeout in `useLayoutEffect`.
                        )}
                        // We'll handle opacity via style to ensure it starts at 1 and goes to 0 if possible?
                        // Actually, doing a JS state "animationProgress" is expensive.
                        // Let's us CSS keyframes if possible, or just two images where one is on top.
                        // If we put the "Contain" image on top and fade it IN (`opacity-0` -> `opacity-100`).
                        alt=""
                    />

                    {/*
                We need a state `showTarget` that goes true immediately after mount?
             */}
                    <img
                        src={canAccessSelected ? selected.full.url : selected.thumb?.url}
                        className={cn(
                            "absolute inset-0 h-full w-full object-contain rounded-2xl shadow-xl",
                            !canAccessSelected && selected.thumb && "blur-md"
                        )}
                        // To fade in, we can use a CSS animation on mount?
                        style={{ animation: "fadeIn 0.5s ease-in-out forwards" }}
                        alt=""
                    />
                    <style>{`
                @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
             `}</style>
                </div>
            ) : null}

            <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                    <div className="truncate text-lg font-black uppercase tracking-wide text-zinc-900">
                        {selected?.title ?? "Untitled"}
                    </div>
                    <div className="mt-2 text-xs font-bold text-zinc-400">
                        {selected ? new Date(selected.createdAt).toLocaleString() : null}
                    </div>
                </div>
                <button type="button" onClick={onBack} className={cn(UI.button, UI.btnYellow)}>
                    Back
                </button>
            </div>

            <div className="mt-4 relative overflow-hidden rounded-3xl border-4 border-zinc-200 bg-white shadow-[6px_6px_0px_rgba(0,0,0,0.14)]">
                <div className="aspect-[4/3] w-full">
                    {selected ? (
                        (() => {
                            const displayUrl = canAccessSelected ? selected.full.url : selected.thumb?.url;
                            if (displayUrl) {
                                return (
                                    <img
                                        ref={imgRef}
                                        src={displayUrl}
                                        alt={selected.title ?? ""}
                                        className={cn(
                                            "h-full w-full object-contain bg-white",
                                            !canAccessSelected && selected.thumb && "blur-md",
                                            // Hide the real image while the fixed overlay is animating
                                            animating ? "opacity-0" : "opacity-100 transition-opacity duration-200"
                                        )}
                                        draggable={false}
                                    />
                                );
                            }
                            if (!canAccessSelected) {
                                return (
                                    <div className="flex h-full w-full items-center justify-center">
                                        <div className="text-center">
                                            <div className="text-sm font-black uppercase text-zinc-500">🔒 Locked</div>
                                            <div className="mt-1 text-sm font-bold text-zinc-400">{accessLabel}</div>
                                        </div>
                                    </div>
                                );
                            }
                            return (
                                <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">
                                    No image
                                </div>
                            );
                        })()
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">
                            No posts
                        </div>
                    )}
                </div>

                {!canAccessSelected && selected?.thumb ? (
                    <div className="absolute inset-0">
                        <LockedOverlay label={accessLabel} />
                    </div>
                ) : null}
            </div>

            {selected ? (
                <div className={cn("mt-4 p-4", UI.card)}>
                    <div className="flex items-center justify-between">
                        <div className={UI.sectionTitle}>Tags</div>
                        <div className="text-xs font-bold text-zinc-400">Click a tag to filter</div>
                    </div>
                    <div className="mt-2 flex flex-wrap gap-2">
                        {selected.tags.length ? (
                            selected.tags.map((t) => (
                                <Tag
                                    key={t}
                                    label={t}
                                    active={tagFilter === t}
                                    onClick={() => setTagFilter(tagFilter === t ? null : t)}
                                />
                            ))
                        ) : (
                            <span className="text-sm font-bold text-zinc-400">No tags</span>
                        )}
                    </div>

                    {tagFilter ? (
                        <div className="mt-4">
                            <div className={UI.sectionTitle}>Similar tagged</div>
                            <div className="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
                                {similarPosts.slice(0, 8).map((p) => (
                                    <button
                                        key={p.id}
                                        type="button"
                                        onClick={() => onSelectSimilar(p.id)}
                                        className={cn(
                                            "relative aspect-square overflow-hidden rounded-2xl border-4 bg-white",
                                            p.id === selected.id ? "border-zinc-900" : "border-zinc-200",
                                            "shadow-[4px_4px_0px_rgba(0,0,0,0.14)] hover:bg-yellow-50",
                                            "active:translate-x-[1px] active:translate-y-[1px]"
                                        )}
                                        title={p.title ?? "Untitled"}
                                    >
                                        {(() => {
                                            const canAccess = canAccessPost(p);
                                            const thumbUrl = canAccess ? p.full.url : p.thumb?.url;
                                            if (thumbUrl)
                                                return (
                                                    <img
                                                        src={thumbUrl}
                                                        className={cn("h-full w-full object-cover", !canAccess && "blur-md")}
                                                        alt=""
                                                        draggable={false}
                                                    />
                                                );
                                            return (
                                                <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">
                                                    No preview
                                                </div>
                                            );
                                        })()}
                                        {!canAccessPost(p) ? (
                                            p.thumb ? (
                                                <div className="absolute inset-0 bg-white/10" />
                                            ) : (
                                                <div className="absolute inset-0 flex items-center justify-center">
                                                    <span className="px-2 text-center text-[11px] font-black uppercase text-zinc-500">
                                                        🔒 {p.allowedRoleIds.map(getRoleName).join(", ")}
                                                    </span>
                                                </div>
                                            )
                                        ) : null}
                                    </button>
                                ))}
                            </div>
                        </div>
                    ) : null}
                </div>
            ) : null}
        </div>
    );
}
