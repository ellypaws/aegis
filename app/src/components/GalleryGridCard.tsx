import React from "react";
import { Lock, ShieldCheck } from "lucide-react";
import { cn } from "../lib/utils";
import { buildSrcSet, GALLERY_CARD_SIZES } from "../lib/imageSrcSet";

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
                    "relative overflow-hidden rounded-3xl border-4 bg-white dark:bg-zinc-800 text-left",
                    "h-60 flex flex-col w-full",
                    selected ? "border-zinc-900 dark:border-zinc-100" : "border-zinc-200 dark:border-zinc-700",
                    "shadow-[6px_6px_0px_rgba(0,0,0,0.16)] dark:shadow-[6px_6px_0px_rgba(0,0,0,0.5)] hover:bg-yellow-50 dark:hover:bg-zinc-700",
                    "transition-[border-color,box-shadow,background-color] duration-500 ease-in-out",
                    "active:translate-x-px active:translate-y-px"
                )}
                title={post.title ?? "Untitled"}
            >
                <div className="h-full w-full bg-zinc-50 dark:bg-zinc-900 relative overflow-hidden">
                    {url ? (
                        <div className="flex h-full w-full items-center justify-center bg-zinc-50 dark:bg-zinc-900 overflow-hidden">
                            <img
                                src={url}
                                srcSet={canAccess && blobId ? buildSrcSet(blobId, token) : undefined}
                                sizes={canAccess && blobId ? GALLERY_CARD_SIZES : undefined}
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
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500">No preview</div>
                    )}
                    {!canAccess ? (
                        url ? (
                            <>
                                <div className="absolute inset-0 bg-white/10 dark:bg-black/30" />
                                {!hasThumbnail && (
                                    <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 pointer-events-none">
                                        <Lock className="h-8 w-8 text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.5)]" />
                                        {roleNames && (
                                            <span className="inline-flex items-center gap-1 rounded-full bg-black/50 px-3 py-1 text-[11px] font-bold uppercase tracking-wider text-white backdrop-blur-sm">
                                                <ShieldCheck className="h-3 w-3" />
                                                {roleNames}
                                            </span>
                                        )}
                                    </div>
                                )}
                            </>
                        ) : (
                            <div className="flex h-full w-full items-center justify-center absolute top-0 left-0">
                                <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500 bg-white/80 dark:bg-black/60 dark:text-zinc-200 rounded-full py-1 backdrop-blur-sm">
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
                "relative overflow-hidden rounded-3xl border-4 bg-white dark:bg-zinc-800",
                selected ? "border-zinc-900 dark:border-zinc-100" : "border-zinc-200 dark:border-zinc-700",
                "shadow-[6px_6px_0px_rgba(0,0,0,0.16)] dark:shadow-[6px_6px_0px_rgba(0,0,0,0.5)] hover:bg-yellow-50 dark:hover:bg-zinc-700",
                "active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title={post.title ?? "Untitled"}
        >
            <div className="aspect-square relative overflow-hidden">
                {url ? (
                    <div className="flex h-full w-full items-center justify-center bg-zinc-50 dark:bg-zinc-900 overflow-hidden">
                        <img
                            src={url}
                            srcSet={canAccess && blobId ? buildSrcSet(blobId, token) : undefined}
                            sizes={canAccess && blobId ? GALLERY_CARD_SIZES : undefined}
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
                    <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500">No preview</div>
                )}
                {!canAccess ? (
                    url ? (
                        <>
                            <div className="absolute inset-0 bg-white/10 dark:bg-black/30" />
                            {!hasThumbnail && (
                                <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 pointer-events-none">
                                    <Lock className="h-8 w-8 text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.5)]" />
                                    {roleNames && (
                                        <span className="inline-flex items-center gap-1 rounded-full bg-black/50 px-3 py-1 text-[11px] font-bold uppercase tracking-wider text-white backdrop-blur-sm">
                                            <ShieldCheck className="h-3 w-3" />
                                            {roleNames}
                                        </span>
                                    )}
                                </div>
                            )}
                        </>
                    ) : (
                        <div className="absolute inset-0 flex items-center justify-center">
                            <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500 dark:text-zinc-300">🔒 {roleNames}</span>
                        </div>
                    )
                ) : null}
                {hasTitle && <TitleOverlay title={post.title!} />}
            </div>
        </button>
    );
}
