import React from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Post } from "../types";
import { getRoleName } from "../data/mock";

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
    onOpen: () => void;
    variant?: "fixed" | "flexible";
}) {
    const url = canAccess ? post.full.url : post.thumb?.url;

    if (variant === "flexible") {
        return (
            <button
                type="button"
                onClick={onOpen}
                className={cn(
                    "relative overflow-hidden rounded-3xl border-4 bg-white text-left",
                    "h-80 flex flex-col w-full", // Added w-full to fill parent container
                    selected ? "border-zinc-900" : "border-zinc-200",
                    "shadow-[6px_6px_0px_rgba(0,0,0,0.16)] hover:bg-yellow-50",
                    "transition-[border-color,box-shadow,background-color] duration-500 ease-in-out", // Animate style properties manually, width handled by parent
                    "active:translate-x-[1px] active:translate-y-[1px]"
                )}
                title={post.title ?? "Untitled"}
            >
                <div className="h-48 w-full border-b-4 border-zinc-100 bg-zinc-50 shrink-0 relative overflow-hidden">
                    {/* Use object-contain to maintain aspect ratio fitting within the box */}
                    {url ? (
                        <img src={url} alt={post.title ?? ""} className={cn("h-full w-full object-cover", !canAccess && "blur-md")} draggable={false} />
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400">No preview</div>
                    )}
                    {!canAccess ? (
                        post.thumb ? (
                            <div className="absolute inset-0 bg-white/10" />
                        ) : (
                            <div className="flex h-full w-full items-center justify-center absolute top-0 left-0">
                                <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500 bg-white/80 rounded-full py-1 backdrop-blur-sm">
                                    🔒 {post.allowedRoleIds.map(getRoleName).slice(0, 1).join(", ")}
                                    {post.allowedRoleIds.length > 1 ? "..." : ""}
                                </span>
                            </div>
                        )
                    ) : null}
                </div>
                <div className="p-3 min-w-0 flex-1 flex flex-col justify-between whitespace-nowrap overflow-hidden">
                    <div>
                        <div className="truncate text-left text-sm font-black uppercase tracking-wide text-zinc-900">{post.title ?? "Untitled"}</div>
                        <div className="mt-1 flex flex-wrap gap-1">
                            {post.tags.slice(0, 2).map((t) => (
                                <span key={t} className={cn(UI.pill, "border-zinc-200", "text-[9px] px-1.5 py-0")}>
                                    #{t}
                                </span>
                            ))}
                        </div>
                    </div>
                </div>
            </button>
        );
    }

    // Original "fixed" aspect-square variant
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
                    post.thumb ? (
                        <div className="absolute inset-0 bg-white/10" />
                    ) : (
                        <div className="absolute inset-0 flex items-center justify-center">
                            <span className="px-3 text-center text-[11px] font-black uppercase text-zinc-500">🔒 {post.allowedRoleIds.map(getRoleName).join(", ")}</span>
                        </div>
                    )
                ) : null}
            </div>
            <div className="p-3">
                <div className="truncate text-left text-sm font-black uppercase tracking-wide text-zinc-900">{post.title ?? "Untitled"}</div>
                <div className="mt-1 flex flex-wrap gap-2">
                    {post.tags.slice(0, 3).map((t) => (
                        <span key={t} className={cn(UI.pill, "border-zinc-200", "text-[10px]")}>
                            #{t}
                        </span>
                    ))}
                </div>
            </div>
        </button>
    );
}
