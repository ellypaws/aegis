import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { DiscordUser, Guild, ViewMode } from "../types";

export function TopBar({
    guild,
    view,
    setView,
    tagFilter,
    setTagFilter,
    user,
    setLoginOpen,
    setUser,
    selectedId,
}: {
    guild: Guild;
    view: ViewMode;
    setView: (v: ViewMode) => void;
    tagFilter: string | null;
    setTagFilter: (t: string | null) => void;
    user: DiscordUser | null;
    setLoginOpen: (b: boolean) => void;
    setUser: (u: DiscordUser | null) => void;
    selectedId: string | null;
}) {
    return (
        <div className="mt-6 flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="flex flex-wrap items-center gap-2">
                <div className={cn("px-3 py-2", UI.soft)}>
                    <div className={UI.label}>Guild</div>
                    <div className="text-sm font-black text-zinc-900">{guild.name}</div>
                </div>

                <div className={cn("px-3 py-2", UI.soft)}>
                    <div className={UI.label}>View</div>
                    <div className="flex gap-2">
                        <button
                            type="button"
                            onClick={() => setView("gallery")}
                            className={cn(UI.button, "px-3 py-1.5 text-xs", view === "gallery" ? UI.btnGreen : UI.btnYellow)}
                        >
                            Gallery
                        </button>
                        <button
                            type="button"
                            onClick={() => setView("post")}
                            className={cn(UI.button, "px-3 py-1.5 text-xs", view === "post" ? UI.btnGreen : UI.btnYellow)}
                            disabled={!selectedId}
                        >
                            Post
                        </button>
                    </div>
                </div>

                {tagFilter ? (
                    <button type="button" onClick={() => setTagFilter(null)} className={cn(UI.button, UI.btnYellow)}>
                        Clear tag #{tagFilter}
                    </button>
                ) : null}
            </div>

            <div className="flex flex-wrap items-center justify-end gap-2">
                {!user ? (
                    <button type="button" onClick={() => setLoginOpen(true)} className={cn(UI.button, UI.btnBlue)}>
                        Login with Discord
                    </button>
                ) : (
                    <>
                        <div className={cn("px-3 py-2", UI.soft)}>
                            <div className={UI.label}>Logged in</div>
                            <div className="text-sm font-black text-zinc-900">{user.username}</div>
                        </div>
                        <button type="button" onClick={() => { localStorage.removeItem("jwt"); setUser(null); }} className={cn(UI.button, UI.btnYellow)}>
                            Logout
                        </button>
                    </>
                )}
            </div>
        </div>
    );
}
