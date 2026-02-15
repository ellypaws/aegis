import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { Post } from "../types";
import { PostCard } from "./PostCard";

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
                    {posts.slice(0, 18).map((p) => (
                        <PostCard
                            key={p.postKey}
                            post={p}
                            canAccess={canAccessByPost(p)}
                            selected={p.postKey === selectedId}
                            onSelect={() => onSelect(p.postKey)}
                            size="sm"
                        />
                    ))}
                </div>
            </div>

            {posts.length > 18 ? <div className="mt-2 text-xs font-bold text-zinc-400">Showing 18 of {posts.length}</div> : null}
        </div>
    );
}
