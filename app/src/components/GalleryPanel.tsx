import { cn } from "../lib/utils";
import { Lock, ShieldCheck } from "lucide-react";
import { UI } from "../constants";
import type { Post } from "../types";
import { buildSrcSet, PANEL_THUMB_SIZES } from "../lib/imageSrcSet";

function resolveImageUrl(post: Post, canAccess: boolean): string | null {
    const blobId = post.image?.blobs?.[0]?.ID;
    if (!blobId) return null;

    const contentType = post.image?.blobs?.[0]?.contentType || "";
    if (contentType.startsWith("video/")) {
        // Always use thumbnail/preview for video in panel
        return `/thumb/${blobId}`;
    }

    if (canAccess) {
        const token = typeof window !== "undefined" ? localStorage.getItem("jwt") : null;
        return `/images/${blobId}${token ? `?token=${token}` : ""}`;
    }
    return `/thumb/${blobId}`;
}

import { ImageWithSpinner } from "./ImageWithSpinner";

export function GalleryPanel({
    title,
    posts,
    selectedId,
    canAccessByPost,
    onSelect,
}: {
    title: string;
    posts: Post[];
    selectedId: string | null;
    canAccessByPost: (p: Post) => boolean;
    onSelect: (id: string) => void;
}) {
    return (
        <div className={cn("p-4", UI.card)}>
            <div className="flex items-start justify-between gap-3">
                <div>
                    <div className={UI.sectionTitle}>{title}</div>
                </div>
            </div>

            <div className="mt-3 overflow-x-auto pb-2 [&::-webkit-scrollbar]:h-2 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-zinc-300 dark:[&::-webkit-scrollbar-thumb]:bg-zinc-600">
                <div className="flex gap-3">
                    {posts.slice(0, 18).map((p) => {
                        const canAccess = canAccessByPost(p);
                        const url = resolveImageUrl(p, canAccess);
                        const hasThumbnail = p.image?.hasThumbnail;
                        const selected = p.postKey === selectedId;
                        const blobId = p.image?.blobs?.[0]?.ID;
                        const token = typeof window !== "undefined" ? localStorage.getItem("jwt") : null;
                        const roleNames = p.allowedRoles.map(r => r.name).join(", ");
                        const focusStyle = { objectPosition: `${p.focusX ?? 50}% ${p.focusY ?? 50}%` };

                        return (
                            <button
                                key={p.postKey}
                                type="button"
                                onClick={() => onSelect(p.postKey)}
                                className={cn(
                                    "relative aspect-square shrink-0 overflow-hidden rounded-2xl border-4 bg-white dark:bg-zinc-800 w-24",
                                    selected ? "border-zinc-900 dark:border-zinc-100" : "border-zinc-200 dark:border-zinc-700",
                                    "shadow-[4px_4px_0px_rgba(0,0,0,0.14)] dark:shadow-[4px_4px_0px_rgba(0,0,0,0.5)] hover:shadow-[6px_6px_0px_rgba(0,0,0,0.16)] dark:hover:bg-zinc-700",
                                    "active:translate-x-[1px] active:translate-y-[1px]",
                                    "transition-all duration-200"
                                )}
                                title={p.title ?? "Untitled"}
                            >
                                <div className="h-full w-full bg-zinc-50 dark:bg-zinc-900 relative overflow-hidden">
                                    {url ? (
                                        <div className="flex h-full w-full items-center justify-center bg-zinc-50 dark:bg-zinc-900 overflow-hidden relative">
                                            <ImageWithSpinner
                                                src={url}
                                                srcSet={canAccess && blobId ? buildSrcSet(blobId, token) : undefined}
                                                sizes={canAccess && blobId ? PANEL_THUMB_SIZES : undefined}
                                                alt={p.title ?? ""}
                                                className={cn(
                                                    "h-full w-full object-cover transition-all duration-500",
                                                    !canAccess && !hasThumbnail && "blur-md scale-105"
                                                )}
                                                style={focusStyle}
                                            />
                                        </div>
                                    ) : (
                                        <div className="flex h-full w-full items-center justify-center text-[10px] font-bold text-zinc-400 dark:text-zinc-500">No img</div>
                                    )}

                                    {!canAccess ? (
                                        url ? (
                                            <>
                                                <div className="absolute inset-0 bg-white/10 dark:bg-black/30" />
                                                {!hasThumbnail && (
                                                    <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 pointer-events-none">
                                                        <Lock className="h-5 w-5 text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.5)]" />
                                                        {roleNames && (
                                                            <span className="inline-flex items-center gap-0.5 rounded-full bg-black/50 px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider text-white backdrop-blur-sm max-w-[90%] truncate">
                                                                <ShieldCheck className="h-2.5 w-2.5 shrink-0" />
                                                                <span className="truncate">{roleNames}</span>
                                                            </span>
                                                        )}
                                                    </div>
                                                )}
                                            </>
                                        ) : (
                                            <div className="absolute inset-0 flex items-center justify-center">
                                                <span className="px-1 text-center text-[9px] font-black uppercase text-zinc-500 dark:text-zinc-300">🔒 {roleNames}</span>
                                            </div>
                                        )
                                    ) : null}

                                    {p.title?.trim() && (
                                        <div className="absolute inset-x-0 bottom-0 pointer-events-none">
                                            <div className="bg-gradient-to-t from-black/80 via-black/40 to-transparent px-2 pb-1.5 pt-6">
                                                <div className="truncate text-left text-[10px] font-black uppercase tracking-wide text-white drop-shadow-[0_1px_2px_rgba(0,0,0,0.5)]">
                                                    {p.title}
                                                </div>
                                            </div>
                                        </div>
                                    )}
                                </div>
                            </button>
                        );
                    })}
                </div>
            </div>

            {posts.length > 18 ? <div className="mt-2 text-xs font-bold text-zinc-400">Showing 18 of {posts.length}</div> : null}
        </div>
    );
}

