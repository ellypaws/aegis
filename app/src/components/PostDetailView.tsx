import { useLayoutEffect, useRef, useState, useMemo } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post, DiscordUser } from "../types";
import { LockedOverlay } from "./LockedOverlay";
export function PostDetailView({
    selected,
    onBack,
    transitionRect,
    user,
}: {
    selected: Post | null;
    onBack: () => void;
    transitionRect?: DOMRect | null;
    user: DiscordUser | null;
}) {
    const imgRef = useRef<HTMLImageElement>(null);
    const [animating, setAnimating] = useState(false);
    const [fixedState, setFixedState] = useState<{
        top: number;
        left: number;
        width: number;
        height: number;
    } | null>(null);

    // Use the hydrated post data directly from App.tsx — no extra fetch needed
    const activePost = selected;

    const canAccess = useMemo(() => {
        if (!activePost) return false;
        if (user?.isAdmin) return true;
        const roleIds = activePost.allowedRoles.map(r => r.id);
        if (roleIds.length === 0) return true;
        const userRoleIds = user?.roles?.map(r => r.id) || [];
        return roleIds.some(id => userRoleIds.includes(id));
    }, [activePost, user]);

    const currentAccessLabel = useMemo(() => {
        if (!activePost) return "";
        return activePost.allowedRoles.length ? `Requires: ${activePost.allowedRoles.map(r => r.name).join(", ")}` : "Public";
    }, [activePost]);

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
                        src={displayUrl || ""}
                        className={cn(
                            "absolute inset-0 h-full w-full object-cover rounded-2xl shadow-xl bg-white transition-opacity duration-500",
                            "opacity-0"
                        )}
                        alt=""
                    />

                    <img
                        src={displayUrl || ""}
                        className={cn(
                            "absolute inset-0 h-full w-full object-contain rounded-2xl shadow-xl",
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
                        {activePost?.title ?? "Untitled"}
                    </div>
                    <div className="mt-2 text-xs font-bold text-zinc-400">
                        {activePost ? new Date(activePost.timestamp).toLocaleString() : null}
                    </div>
                </div>
                <button type="button" onClick={onBack} className={cn(UI.button, UI.btnYellow)}>
                    Back
                </button>
            </div>

            <div className="mt-4 relative overflow-hidden rounded-3xl border-4 border-zinc-200 bg-white shadow-[6px_6px_0px_rgba(0,0,0,0.14)]">
                <div className="w-full min-h-[50vh] flex items-center justify-center">
                    {activePost ? (
                        (() => {
                            if (displayUrl) {
                                return (
                                    <img
                                        ref={imgRef}
                                        src={displayUrl}
                                        alt={activePost.title ?? ""}
                                        className={cn(
                                            "max-h-[85vh] w-auto h-auto object-contain bg-white mx-auto",
                                            animating ? "opacity-0" : "opacity-100 transition-opacity duration-200"
                                        )}
                                        draggable={false}
                                    />
                                );
                            }
                            if (!canAccess) {
                                return (
                                    <div className="flex h-full w-full items-center justify-center py-20">
                                        <div className="text-center">
                                            <div className="text-sm font-black uppercase text-zinc-500">🔒 Locked</div>
                                            <div className="mt-1 text-sm font-bold text-zinc-400">{currentAccessLabel}</div>
                                        </div>
                                    </div>
                                );
                            }
                            return (
                                <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 py-20">
                                    No image
                                </div>
                            );
                        })()
                    ) : (
                        <div className="flex h-full w-full items-center justify-center text-sm font-bold text-zinc-400 py-20">
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
                    <div className="text-sm text-zinc-600 font-medium whitespace-pre-wrap">
                        {activePost.description}
                    </div>
                </div>
            ) : null}
        </div>
    );
}
