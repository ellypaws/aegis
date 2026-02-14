import React, { useLayoutEffect, useRef, useState } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";

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

    const getUrl = (p: Post, access: boolean) => {
        const thumbUrl = p.image?.thumbnail;
        const fullUrl = p.image?.blobs?.[0]?.data;
        return access ? (fullUrl || thumbUrl) : thumbUrl;
    };

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
                    <img
                        src={getUrl(selected, canAccessSelected)}
                        className={cn(
                            "absolute inset-0 h-full w-full object-cover rounded-2xl shadow-xl bg-white transition-opacity duration-500",
                            !canAccessSelected && selected.image?.thumbnail && "blur-md",
                            "opacity-0"
                        )}
                        alt=""
                    />

                    <img
                        src={getUrl(selected, canAccessSelected)}
                        className={cn(
                            "absolute inset-0 h-full w-full object-contain rounded-2xl shadow-xl",
                            !canAccessSelected && selected.image?.thumbnail && "blur-md"
                        )}
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
                        {selected ? new Date(selected.timestamp).toLocaleString() : null}
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
                            const displayUrl = getUrl(selected, canAccessSelected);
                            if (displayUrl) {
                                return (
                                    <img
                                        ref={imgRef}
                                        src={displayUrl}
                                        alt={selected.title ?? ""}
                                        className={cn(
                                            "h-full w-full object-contain bg-white",
                                            !canAccessSelected && selected.image?.thumbnail && "blur-md",
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

                {!canAccessSelected && selected?.image?.thumbnail ? (
                    <div className="absolute inset-0">
                        <LockedOverlay label={accessLabel} />
                    </div>
                ) : null}
            </div>

            {selected ? (
                <div className={cn("mt-4 p-4", UI.card)}>
                    {/* Tags/Similar posts logic removed as Tags are gone */}
                    <div className="text-sm text-zinc-600 font-medium">
                        {selected.description}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
