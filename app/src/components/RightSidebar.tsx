import React from "react";
import { cn } from "../lib/utils";
import { UI } from "../constants";
import { Post, DiscordUser } from "../types";
import { GalleryPanel } from "./GalleryPanel";
import { ChannelPill, RolePill } from "./Pills";
import { Tag } from "./Tag";
import { ViewerActions } from "./ViewerActions";

export function RightSidebar({
    tagFilter,
    setTagFilter,
    q,
    posts,
    selectedId,
    canAccessPost,
    onSelect,
    selected,
    canAccessSelected,
    user,
}: {
    tagFilter: string | null;
    setTagFilter: (t: string | null) => void;
    q: string;
    posts: Post[];
    selectedId: string | null;
    canAccessPost: (p: Post) => boolean;
    onSelect: (id: string) => void;
    selected: Post | null;
    canAccessSelected: boolean;
    user: DiscordUser | null;
}) {
    return (
        <div className="sticky top-4 space-y-4">
            <GalleryPanel
                title="Gallery"
                subtitle={
                    tagFilter
                        ? `Filtered by #${tagFilter}`
                        : q.trim()
                            ? `Search: “${q.trim()}”`
                            : "Recent + browse"
                }
                posts={posts}
                selectedId={selectedId}
                canAccessByPost={canAccessPost}
                onSelect={onSelect}
            />

            <div className={cn("p-4", UI.card)}>
                <div className="flex items-start justify-between gap-3">
                    <div>
                        <div className={UI.sectionTitle}>Post details</div>
                        <div className="mt-1 text-xs font-bold text-zinc-400">
                            Title, description, tags, access + downloads
                        </div>
                    </div>
                    {selected ? (
                        <div
                            className={cn(
                                "rounded-full border-4 px-3 py-1 text-[11px] font-black uppercase tracking-wide shadow-[3px_3px_0px_rgba(0,0,0,0.12)]",
                                canAccessSelected
                                    ? "border-green-300 bg-green-200 text-green-900"
                                    : "border-zinc-200 bg-white text-zinc-500"
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
                                <div className={UI.label}>Title</div>
                                <div className="mt-1 text-sm font-black text-zinc-900">{selected.title ?? "Untitled"}</div>
                            </div>
                            <div>
                                <div className={UI.label}>Description</div>
                                <div className="mt-1 whitespace-pre-wrap text-sm font-bold text-zinc-700">
                                    {selected.description?.trim() ? selected.description : "—"}
                                </div>
                            </div>
                            <div>
                                <div className={UI.label}>Channels</div>
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {selected.channelIds.length ? (
                                        selected.channelIds.map((c) => <ChannelPill key={c} channelId={c} />)
                                    ) : (
                                        <span className="text-sm font-bold text-zinc-400">—</span>
                                    )}
                                </div>
                            </div>
                            <div>
                                <div className={UI.label}>Tags</div>
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {selected.tags.length ? (
                                        selected.tags.map((t) => (
                                            <Tag
                                                key={t}
                                                label={t}
                                                active={tagFilter === t}
                                                onClick={() => setTagFilter(tagFilter === t ? null : t)}
                                            />
                                        ))
                                    ) : (
                                        <span className="text-sm font-bold text-zinc-400">—</span>
                                    )}
                                </div>
                            </div>
                            <div>
                                <div className={UI.label}>Allowed roles</div>
                                <div className="mt-2 flex flex-wrap gap-2">
                                    {selected.allowedRoleIds.length ? (
                                        selected.allowedRoleIds.map((r) => <RolePill key={r} roleId={r} />)
                                    ) : (
                                        <span className="text-sm font-bold text-zinc-400">—</span>
                                    )}
                                </div>
                            </div>
                        </div>

                        <div className="mt-4 border-t-4 border-zinc-200 pt-4">
                            <div className={UI.sectionTitle}>Downloads</div>
                            <div className="mt-2">
                                <ViewerActions post={selected} canAccess={canAccessSelected} />
                            </div>
                        </div>
                    </>
                ) : (
                    <div className="mt-3 text-sm font-bold text-zinc-500">No post selected.</div>
                )}
            </div>

            <div className={cn("p-4", UI.card)}>
                <div className={UI.sectionTitle}>Role-based access</div>
                <div className="mt-2 text-sm font-bold text-zinc-700">
                    Your roles:
                    <div className="mt-2 flex flex-wrap gap-2">
                        {(user?.roles ?? []).length ? (
                            (user?.roles ?? []).map((r) => <RolePill key={r.id} roleId={r.id} />)
                        ) : (
                            <span className="text-sm font-bold text-zinc-400">Not logged in</span>
                        )}
                    </div>
                </div>
                <div className="mt-3 text-xs font-bold text-zinc-400">
                    Access is granted if your roles intersect a post’s allowed roles. Authors always have access to their own
                    posts.
                </div>
            </div>
        </div>
    );
}
