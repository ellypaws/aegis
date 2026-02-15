import React from "react";
import { cn } from "../lib/utils";

import type { Post } from "../types";

/** Title overlay with dark gradient — sits at the bottom of the image */
function TitleOverlay({ title }: { title: string }) {
    return (
        <div className="absolute inset-x-0 bottom-0 pointer-events-none">
            <div className="bg-gradient-to-t from-black/70 via-black/30 to-transparent px-4 pb-3 pt-8">
                <div className="truncate text-left text-sm font-black uppercase tracking-wide text-white drop-shadow-[0_1px_3px_rgba(0,0,0,0.5)]">
                    {title}
                </div>
            </div>
        </div>
    );
}

export function GalleryGridCard({
    post,
    canAccess,
    selected,
    onOpen,
    variant = "fixed",
}: {
    post: Post;
    canAccess: boolean;
    selected: boolean;
    onOpen: (e: React.MouseEvent) => void;
    variant?: "fixed" | "flexible";
}) {
    const roleNames = post.allowedRoles.map(r => r.name).join(", ");
    const hasTitle = !!post.title?.trim();
    const blobId = post.image?.blobs?.[0]?.ID;
    const token = typeof window !== 'undefined' ? localStorage.getItem("jwt") : null;
    let url = null;

    if (blobId) {
        if (canAccess) {
            url = `/images/${blobId}${token ? `?token=${token}` : ""}`;
        } else {
            url = `/thumb/${blobId}`;
        }
    }

    const hasThumbnail = post.image?.hasThumbnail;
    const focusStyle = { objectPosition: `${post.focusX ?? 50}% ${post.focusY ?? 50}%` };

    if (variant === "flexible") {
        return (
            <button
                type="button"
                onClick={onOpen}
                className={cn(
                    "relative overflow-hidden rounded-3xl border-4 bg-white text-left",
                    "h-60 flex flex-col w-full",
                    selected ? "border-zinc-900" : "border-zinc-200",
                    "shadow-[6px_6px_0px_rgba(0,0,0,0.16)] hover:bg-yellow-50",
                    "transition-[border-color,box-shadow,background-color] duration-500 ease-in-out",
                    "active:translate-x-px active:translate-y-px"
                )}
                title={post.title ?? "Untitled"}
            >
                <div className="h-full w-full bg-zinc-50 relative overflow-hidden">
                    {url ? (
                        <div className="flex h-full w-full items-center justify-center bg-zinc-50 overflow-hidden">
                            <img
                                src={url}
                                alt={post.title ?? ""}
                                className={cn(
                                    "h-full w-full object-cover transition-all duration-500",
                                    !canAccess && !hasThumbnail && "blur-md scale-105"
                                )}
                                style={focusStyle}
                                draggable={false}
                                loading="lazy"
                            />
                        </div>
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">No preview</div>
                    )}
                    {!canAccess ? (
                        url ? (
                            <div className="absolute inset-0 bg-white/10" />
                        ) : (
                            <div className="flex h-full w-full items-center justify-center absolute top-0 left-0">
                                <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500 bg-white/80 rounded-full py-1 backdrop-blur-sm">
                                    🔒 {roleNames}
                                </span>
                            </div>
                        )
                    ) : null}
                    {hasTitle && <TitleOverlay title={post.title!} />}
                </div>
            </button>
        );
    }

    return (
        <button
            type="button"
            onClick={onOpen}
            className={cn(
                "relative overflow-hidden rounded-3xl border-4 bg-white",
                selected ? "border-zinc-900" : "border-zinc-200",
                "shadow-[6px_6px_0px_rgba(0,0,0,0.16)] hover:bg-yellow-50",
                "active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title={post.title ?? "Untitled"}
        >
            <div className="aspect-square relative overflow-hidden">
                {url ? (
                    <div className="flex h-full w-full items-center justify-center bg-zinc-50 overflow-hidden">
                        <img
                            src={url}
                            alt={post.title ?? ""}
                            className={cn(
                                "h-full w-full object-cover transition-all duration-500",
                                !canAccess && !hasThumbnail && "blur-md scale-105"
                            )}
                            style={focusStyle}
                            draggable={false}
                            loading="lazy"
                        />
                    </div>
                ) : (
                    <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">No preview</div>
                )}
                {!canAccess ? (
                    url ? (
                        <div className="absolute inset-0 bg-white/10" />
                    ) : (
                        <div className="absolute inset-0 flex items-center justify-center">
                            <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500">🔒 {roleNames}</span>
                        </div>
                    )
                ) : null}
                {hasTitle && <TitleOverlay title={post.title!} />}
            </div>
        </button>
    );
}
