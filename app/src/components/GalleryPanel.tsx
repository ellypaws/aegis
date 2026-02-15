import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";

function resolveImageUrl(post: Post, canAccess: boolean): string | null {
    const blobId = post.image?.blobs?.[0]?.ID;
    if (!blobId) return null;

    if (canAccess) {
        const token = typeof window !== "undefined" ? localStorage.getItem("jwt") : null;
        return `/images/${blobId}${token ? `?token=${token}` : ""}`;
    }
    return `/thumb/${blobId}`;
}

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

            <div className="mt-3 overflow-x-auto pb-2">
                <div className="flex gap-3">
                    {posts.slice(0, 18).map((p) => {
                        const canAccess = canAccessByPost(p);
                        const url = resolveImageUrl(p, canAccess);
                        const hasThumbnail = p.image?.hasThumbnail;
                        const selected = p.postKey === selectedId;

                        return (
                            <button
                                key={p.postKey}
                                type="button"
                                onClick={() => onSelect(p.postKey)}
                                className={cn(
                                    "relative aspect-square shrink-0 overflow-hidden rounded-2xl border-4 bg-white w-24",
                                    selected ? "border-zinc-900" : "border-zinc-200",
                                    "shadow-[4px_4px_0px_rgba(0,0,0,0.14)] hover:shadow-[6px_6px_0px_rgba(0,0,0,0.16)]",
                                    "active:translate-x-[1px] active:translate-y-[1px]"
                                )}
                                title={p.title ?? "Untitled"}
                            >
                                {url ? (
                                    <img
                                        src={url}
                                        alt={p.title ?? ""}
                                        className={cn(
                                            "h-full w-full object-cover transition-all duration-500",
                                            !canAccess && !hasThumbnail && "blur-md scale-105"
                                        )}
                                        draggable={false}
                                        loading="lazy"
                                    />
                                ) : (
                                    <div className="flex h-full w-full items-center justify-center text-xs font-bold text-zinc-400">
                                        No image
                                    </div>
                                )}

                                {!canAccess ? (
                                    <div className="absolute inset-0 pointer-events-none">
                                        <LockedOverlay label={`Requires: ${p.allowedRoles.map(r => r.name).join(", ") || "(no roles)"}`} />
                                    </div>
                                ) : null}

                                <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-white/90 to-transparent p-2">
                                    <div className="truncate text-left text-[11px] font-black uppercase tracking-wide text-zinc-700">
                                        {p.title ?? "Untitled"}
                                    </div>
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
