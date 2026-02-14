import React, { useEffect, useState } from "react";
import { cn, resolveImageSrc, safeRevoke } from "../lib/utils";

import type { Post } from "../types";

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
    const [blobUrl, setBlobUrl] = useState<string | null>(null);
    const thumbUrl = resolveImageSrc(post.image?.thumbnail);
    const roleNames = post.allowedRoles.map(r => r.name).join(", ");

    // Effect to fetch image blob if needed
    useEffect(() => {
        if (!post.image?.blobs?.[0]?.ID) {
            setBlobUrl(null);
            return;
        }

        const blobId = post.image.blobs[0].ID;
        // Fallback or public blur URL
        const blurUrl = `http://localhost:3000/blur/${blobId}`;

        if (!canAccess) {
            setBlobUrl(blurUrl);
            return;
        }

        const token = localStorage.getItem("jwt");
        const headers: Record<string, string> = {};
        if (token) headers["Authorization"] = `Bearer ${token}`;

        let active = true;
        fetch(`http://localhost:3000/images/${blobId}`, { headers })
            .then(async (res) => {
                if (!res.ok) {
                    if (active) setBlobUrl(blurUrl);
                    return;
                }
                const blob = await res.blob();
                if (active) {
                    const url = URL.createObjectURL(blob);
                    setBlobUrl(url);
                }
            })
            .catch(() => {
                if (active) setBlobUrl(blurUrl);
            });

        return () => {
            active = false;
        };
    }, [post.image, canAccess]);

    // Cleanup blob url on unmount or change
    useEffect(() => {
        return () => {
            if (blobUrl && blobUrl.startsWith("blob:")) {
                safeRevoke(blobUrl);
            }
        }
    }, [blobUrl]);


    const url = blobUrl || thumbUrl;

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
                <div className="h-48 w-full border-b-4 border-zinc-100 bg-zinc-50 shrink-0 relative overflow-hidden">
                    {url ? (
                        <img src={url} alt={post.title ?? ""} className={cn("h-full w-full object-cover", !canAccess && "blur-md")} draggable={false} />
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">No preview</div>
                    )}
                    {!canAccess ? (
                        thumbUrl ? (
                            <div className="absolute inset-0 bg-white/10" />
                        ) : (
                            <div className="flex h-full w-full items-center justify-center absolute top-0 left-0">
                                <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500 bg-white/80 rounded-full py-1 backdrop-blur-sm">
                                    🔒 {roleNames}
                                </span>
                            </div>
                        )
                    ) : null}
                </div>
                <div className="p-3 min-w-0 flex-1 flex flex-col justify-between whitespace-nowrap overflow-hidden">
                    <div>
                        <div className="truncate text-left text-sm font-black uppercase tracking-wide text-zinc-900">{post.title ?? "Untitled"}</div>
                    </div>
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
            <div className="aspect-square relative">
                {url ? (
                    <img src={url} alt={post.title ?? ""} className={cn("h-full w-full object-cover bg-zinc-50", !canAccess && "blur-md")} draggable={false} />
                ) : (
                    <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">No preview</div>
                )}
                {!canAccess ? (
                    thumbUrl ? (
                        <div className="absolute inset-0 bg-white/10" />
                    ) : (
                        <div className="absolute inset-0 flex items-center justify-center">
                            <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500">🔒 {roleNames}</span>
                        </div>
                    )
                ) : null}
            </div>
            <div className="p-3">
                <div className="truncate text-left text-sm font-black uppercase tracking-wide text-zinc-900">{post.title ?? "Untitled"}</div>
                {/* Tags removed */}
            </div>
        </button>
    );
}
