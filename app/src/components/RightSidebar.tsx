
import { useState } from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post, DiscordUser } from "../types";
import { GalleryPanel } from "./GalleryPanel";
import { ChannelPill, RolePill } from "./Pills";
import { ViewerActions } from "./ViewerActions";

export function RightSidebar({
    posts,
    selectedId,
    canAccessPost,
    onSelect,
    selected,
    user,
    onEditPost,
    onDeletePost,
}: {
    posts: Post[];
    selectedId: string | null;
    canAccessPost: (p: Post) => boolean;
    onSelect: (id: string) => void;
    selected: Post | null;
    user: DiscordUser | null;
    onEditPost?: (post: Post) => void;
    onDeletePost?: (postKey: string) => void;
}) {
    const canAccessSelected = selected ? canAccessPost(selected) : false;
    const isAdmin = !!user?.isAdmin;
    const [confirmDelete, setConfirmDelete] = useState(false);

    return (
        <div className="sticky top-4 space-y-4">
            <GalleryPanel
                title="Gallery"
                posts={posts}
                selectedId={selectedId}
                canAccessByPost={canAccessPost}
                onSelect={onSelect}
            />

            <div className={cn("p-4", UI.card)}>
                <div className="flex items-start justify-between gap-3">
                    <div>
                        <div className={UI.sectionTitle}>{selected?.title}</div>
                    </div>
                    {selected ? (
                        <div
                            className={cn(
                                "rounded-full border-4 px-3 py-1 text-[11px] font-black uppercase tracking-wide shadow-[3px_3px_0px_rgba(0,0,0,0.12)]",
                                canAccessSelected
                                    ? "border-green-300 bg-green-200 text-green-900 dark:border-green-700 dark:bg-green-900 dark:text-green-100"
                                    : "border-zinc-200 bg-white text-zinc-500 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400"
                            )}
                        >
                            {canAccessSelected ? "Access OK" : "Locked"}
                        </div>
                    ) : null}
                </div>

                {selected ? (
                    <>
                        <div className="mt-3 space-y-3">
                            <div>
                                <div className="mt-1 whitespace-pre-wrap text-sm font-bold text-zinc-700 dark:text-zinc-300">
                                    {selected.description?.trim() ? selected.description : "—"}
                                </div>
                            </div>
                            <div>
                                <div className={UI.label}>Posted</div>
                                <div className="mt-1 text-sm font-bold text-zinc-700 dark:text-zinc-300">
                                    {new Date(selected.timestamp).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" })}
                                </div>
                            </div>
                            <div>
                                <div className={UI.label}>Channels</div>
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {selected.channelId ? (
                                        <ChannelPill name={selected.channelId} />
                                    ) : (
                                        <span className="text-sm font-bold text-zinc-400 dark:text-zinc-500">—</span>
                                    )}
                                </div>
                            </div>

                            <div>
                                <div className={UI.label}>Allowed roles</div>
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {selected.allowedRoles.length ? (
                                        selected.allowedRoles.map((r) => <RolePill key={r.id} name={r.name} color={r.color} />)
                                    ) : (
                                        <span className="text-sm font-bold text-zinc-400 dark:text-zinc-500">—</span>
                                    )}
                                </div>
                            </div>
                        </div>

                        <div className="mt-4 border-t-4 border-zinc-200 dark:border-zinc-700 pt-4">
                            <div className={UI.sectionTitle}>Downloads</div>
                            <div className="mt-2">
                                <ViewerActions post={selected} canAccess={canAccessSelected} />
                            </div>
                        </div>

                        {isAdmin && (
                            <div className="mt-4 border-t-4 border-zinc-200 dark:border-zinc-700">
                                <div className="mt-2 flex gap-2">
                                    <button
                                        type="button"
                                        onClick={() => onEditPost?.(selected)}
                                        className={cn(UI.button, UI.btnBlue, "flex-1 text-xs py-2")}
                                    >
                                        ✏️ Edit
                                    </button>
                                    {!confirmDelete ? (
                                        <button
                                            type="button"
                                            onClick={() => setConfirmDelete(true)}
                                            className={cn(UI.button, UI.btnRed, "flex-1 text-xs py-2")}
                                        >
                                            🗑️ Delete
                                        </button>
                                    ) : (
                                        <div className="flex-1 flex gap-1">
                                            <button
                                                type="button"
                                                onClick={() => {
                                                    onDeletePost?.(selected.postKey);
                                                    setConfirmDelete(false);
                                                }}
                                                className={cn(UI.button, "border-red-600 bg-red-500 text-white hover:bg-red-600 flex-1 text-xs py-2 dark:border-red-800 dark:bg-red-900")}
                                            >
                                                Confirm
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => setConfirmDelete(false)}
                                                className={cn(UI.button, UI.btnYellow, "flex-1 text-xs py-2")}
                                            >
                                                Cancel
                                            </button>
                                        </div>
                                    )}
                                </div>
                            </div>
                        )}
                    </>
                ) : (
                    <div className="mt-3 text-sm font-bold text-zinc-500 dark:text-zinc-400">No post selected.</div>
                )}
            </div>

            <div className={cn("p-4", UI.card)}>
                <div className={UI.sectionTitle}>Roles</div>
                <div className="mt-2 flex flex-wrap gap-2">
                    {(user?.roles ?? []).length ? (
                        (user?.roles ?? []).map((r) => <RolePill key={r.id} name={r.name} color={r.color} />)
                    ) : (
                        <span className="text-sm font-bold text-zinc-400 dark:text-zinc-500">Not logged in</span>
                    )}
                </div>
            </div>
        </div>
    );
}
