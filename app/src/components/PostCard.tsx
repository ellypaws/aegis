import React from "react";
import { cn } from "../lib/utils";
import { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";
import { getRoleName } from "../data/mock";

export function PostCard({
    post,
    canAccess,
    selected,
    onSelect,
    size = "md",
}: {
    post: Post;
    canAccess: boolean;
    selected: boolean;
    onSelect: () => void;
    size?: "sm" | "md";
}) {
    const url = canAccess ? post.full.url : post.thumb?.url;
    const w = size === "sm" ? "w-24" : "w-28";
    return (
        <button
            type="button"
            onClick={onSelect}
            className={cn(
                "relative aspect-square shrink-0 overflow-hidden rounded-2xl border-4 bg-white",
                w,
                selected ? "border-zinc-900" : "border-zinc-200",
                "shadow-[4px_4px_0px_rgba(0,0,0,0.14)] hover:shadow-[6px_6px_0px_rgba(0,0,0,0.16)]",
                "active:translate-x-[1px] active:translate-y-[1px]"
            )}
            title={post.title ?? "Untitled"}
        >
            {url ? (
                <img src={url} alt={post.title ?? ""} className={cn("h-full w-full object-cover", !canAccess && "blur-md")} draggable={false} />
            ) : (
                <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">No image</div>
            )}

            {!canAccess ? (
                post.thumb ? (
                    <div className="absolute inset-0 bg-white/10" />
                ) : (
                    <LockedOverlay label={`Requires: ${post.allowedRoleIds.map(getRoleName).join(", ") || "(no roles)"}`} />
                )
            ) : null}

            <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-white/90 to-transparent p-2">
                <div className="truncate text-left text-[11px] font-black uppercase tracking-wide text-zinc-700">{post.title ?? "Untitled"}</div>
            </div>
        </button>
    );
}
