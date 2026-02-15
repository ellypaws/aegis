
import { cn } from "../lib/utils";
import { UI } from "../constants";
import type { DiscordUser } from "../types";
import { Link } from "react-router-dom";

export function ProfileSidebar({ user, onLogin }: { user: DiscordUser | null; onLogin: () => void }) {
    if (!user) {
        return (
            <div className={cn("p-6 text-center space-y-4", UI.card)}>
                <div className="mx-auto h-20 w-20 rounded-full bg-zinc-100 flex items-center justify-center text-3xl">
                    👋
                </div>
                <div>
                    <div className="text-lg font-black text-zinc-900">Welcome!</div>
                    <div className="mt-1 text-sm font-bold text-zinc-500">
                        Login to see your profile and access locked content.
                    </div>
                </div>
                <button
                    type="button"
                    onClick={onLogin}
                    className={cn(UI.button, UI.btnBlue, "w-full justify-center")}
                >
                    Login with Discord
                </button>
            </div>
        );
    }

    return (
        <div className={cn("overflow-hidden rounded-[20px] bg-[#111214] text-gray-100 shadow-xl border border-[#1e1f22]")}>
            {/* Banner */}
            <div
                className="h-[120px] w-full bg-[#5865F2] relative bg-cover bg-center"
                style={user.banner ? { backgroundImage: `url(${user.banner})` } : undefined}
            >
                <div className="absolute inset-0 bg-gradient-to-b from-transparent to-black/20" />
            </div>

            <div className="px-4 pb-4 relative">
                {/* Avatar */}
                <div className="absolute -top-[50px] left-4">
                    <div className="relative">
                        <div className="h-[92px] w-[92px] rounded-full bg-[#111214] flex items-center justify-center p-[6px]">
                            {user.avatar ? (
                                <img src={user.avatar} alt="" className="h-full w-full rounded-full object-cover" />
                            ) : (
                                <div className="h-full w-full rounded-full bg-[#5865F2] flex items-center justify-center text-3xl font-bold text-white">
                                    {user.username.slice(0, 1).toUpperCase()}
                                </div>
                            )}
                        </div>
                        <div className="absolute bottom-1 right-1 h-6 w-6 rounded-full bg-[#23a559] border-4 border-[#111214]" title="Online" />
                    </div>
                </div>

                {/* Profile Info */}
                <div className="mt-[50px]">
                    <div className="rounded-lg bg-[#111214] p-3">
                        <div className="mt-[25px] text-xl font-bold text-white max-w-full truncate">{user.username}</div>
                        <div className="text-sm text-gray-300 font-medium">{user.username.toLowerCase()}</div>

                        <div className="my-3 h-[1px] w-full bg-[#2B2D31]" />

                        <div className="text-xs font-bold uppercase tracking-wide text-gray-300 mb-2">Roles</div>
                        <div className="flex flex-wrap gap-1">
                            {user.roles && user.roles.length > 0 ? (
                                // use r.roleId as r
                                user.roles.map((r) => (
                                    <div
                                        key={r.id}
                                        className="flex items-center gap-1.5 rounded bg-[#2B2D31] px-2 py-1 text-xs font-medium text-gray-200 hover:bg-[#3f4147] transition-colors cursor-default"
                                    >
                                        <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: r.color ? `#${r.color.toString(16)}` : '#9ca3af' }} />
                                        {r.name}
                                    </div>
                                ))
                            ) : (
                                <span className="text-xs text-gray-500 italic">No roles</span>
                            )}
                        </div>

                        <div className="mt-4">
                            <div className="text-xs font-bold uppercase tracking-wide text-gray-300 mb-2">Member Since</div>
                            <div className="text-xs text-gray-300">Feb 14, 2026</div>

                            {user.isAdmin && (
                                <Link to="/settings" className="block mt-4 text-xs font-bold uppercase tracking-wide text-yellow-400 hover:text-yellow-300">
                                    Admin Settings
                                </Link>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
