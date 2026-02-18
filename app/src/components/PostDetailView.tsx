import { useRef, useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { LockedOverlay } from "./LockedOverlay";
import { ImageWithSpinner } from "./ImageWithSpinner";
export function PostDetailView({
    selected,
    onBack, 
    canAccessPost,
}: {
    selected: Post | null;
    onBack: () => void;
    canAccessPost: (p: Post) => boolean;
}) {
    const imgRef = useRef<HTMLImageElement>(null);

    // Use the hydrated post data directly from App.tsx — no extra fetch needed
    const activePost = selected;

    const canAccess = useMemo(() => {
        if (!activePost) return false;
        return canAccessPost(activePost);
    }, [activePost, canAccessPost]);

    const currentAccessLabel = useMemo(() => {
        if (!activePost) return "";
        if (canAccess) return "Public"; // Simplified label if accessible
        return activePost.allowedRoles.length ? `Requires: ${activePost.allowedRoles.map(r => r.name).join(", ")}` : "Public";
    }, [activePost, canAccess]);

    const getDisplayUrl = (p: Post | null, access: boolean) => {
        if (!p) return undefined;
        const blobId = p.image?.blobs?.[0]?.ID;
        if (!blobId) return undefined;

        const token = localStorage.getItem("jwt");

        if (access) {
            return `/images/${blobId}${token ? `?token=${token}` : ""}`;
        }
        return `/thumb/${blobId}`;
    };

    const displayUrl = getDisplayUrl(activePost, canAccess);

    return (
        <div className={cn("p-4", UI.card, "backdrop-blur-sm")}>
            <div className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                    <div className="truncate text-lg font-black uppercase tracking-wide text-zinc-900 dark:text-zinc-100">
                        {activePost?.title ?? "Untitled"}
                    </div>
                </div>
                <button type="button" onClick={onBack} className={cn(UI.button, UI.btnYellow)}>
                    Back
                </button>
            </div>

            <div className="mt-4 relative overflow-hidden rounded-3xl border-4 border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-[6px_6px_0px_rgba(0,0,0,0.14)] dark:shadow-[6px_6px_0px_rgba(0,0,0,0.5)]">
                <div className="w-full min-h-[50vh] flex items-center justify-center">
                    {activePost ? (
                        (() => {
                            if (displayUrl) {
                                const contentType = activePost.image?.blobs?.[0]?.contentType || "";
                                const filename = activePost.image?.blobs?.[0]?.filename || "";
                                const isVideo = contentType.startsWith("video/") || /\.(mp4|webm|mov|mkv|avi)$/i.test(filename);

                                if (isVideo && canAccess) {
                                    return (
                                        <video
                                            src={displayUrl}
                                            controls
                                            autoPlay
                                            loop
                                            className="max-h-[85vh] w-auto h-auto object-contain bg-black mx-auto rounded-lg shadow-lg"
                                        />
                                    );
                                }

                                return (
                                    <ImageWithSpinner
                                        ref={imgRef}
                                        src={displayUrl}
                                        alt={activePost.title ?? ""}
                                        className="max-h-[85vh] w-auto h-auto object-contain bg-white dark:bg-zinc-900 mx-auto"
                                        draggable={false}
                                    />
                                );
                            }
                            if (!canAccess) {
                                return (
                                    <div className="flex h-full w-full items-center justify-center py-20">
                                        <div className="text-center">
                                            <div className="text-sm font-black uppercase text-zinc-500 dark:text-zinc-400">🔒 Locked</div>
                                            <div className="mt-1 text-sm font-bold text-zinc-400 dark:text-zinc-500">{currentAccessLabel}</div>
                                        </div>
                                    </div>
                                );
                            }
                            return (
                                <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500 py-20">
                                    No image
                                </div>
                            );
                        })()
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 dark:text-zinc-500 py-20">
                            No posts
                        </div>
                    )}
                </div>

                {!canAccess && activePost?.image?.blobs?.[0]?.ID ? (
                    <div className="absolute inset-0 pointer-events-none">
                        <LockedOverlay label={currentAccessLabel} />
                    </div>
                ) : null}
            </div>

            {activePost ? (
                <div className={cn("mt-4 p-4", UI.card)}>
                    <div className="text-sm text-zinc-600 dark:text-zinc-300 font-medium whitespace-pre-wrap">
                        {activePost.description}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
