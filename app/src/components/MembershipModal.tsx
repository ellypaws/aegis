import { useEffect, useState } from "react";
import { cn, getAvatarUrl, getBannerUrl } from "../lib/utils";
import { UI } from "../constants";
import { useSettings } from "../contexts/SettingsContext";
import type { DiscordUser } from "../types";
import { Patterns } from "./Patterns";
import { X, Shield, ShieldCheck, Globe } from "lucide-react";

export function MembershipModal({
    onClose,
}: {
    onClose: () => void;
}) {
    const [users, setUsers] = useState<DiscordUser[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const { settings, updateSettings } = useSettings();

    useEffect(() => {
        document.body.style.overflow = "hidden";
        return () => {
            document.body.style.overflow = "unset";
        };
    }, []);

    useEffect(() => {
        const fetchUsers = async () => {
            const token = localStorage.getItem("jwt");
            if (!token) {
                setError("Unauthorized");
                setLoading(false);
                return;
            }
            try {
                const res = await fetch("/users", {
                    headers: { Authorization: `Bearer ${token}` }
                });
                if (!res.ok) throw new Error("Failed to fetch users");
                const data = await res.json();
                setUsers(data);
            } catch (err) {
                setError("Failed to load users");
            } finally {
                setLoading(false);
            }
        };
        fetchUsers();
    }, []);

    const toggleAdmin = async (userId: string, currentStatus: boolean) => {
        const token = localStorage.getItem("jwt");
        if (!token) return;

        // Optimistic update
        setUsers(prev => prev.map(u => u.userId === userId ? { ...u, isAdmin: !currentStatus } : u));

        try {
            const res = await fetch(`/users/${userId}/admin`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${token}`
                },
                body: JSON.stringify({ isAdmin: !currentStatus })
            });
            if (!res.ok) throw new Error("Failed");
        } catch (err) {
            // Revert
            setUsers(prev => prev.map(u => u.userId === userId ? { ...u, isAdmin: currentStatus } : u));
            alert("Failed to update admin status");
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-md animate-in fade-in duration-200">
            <div className={cn("relative w-full max-w-3xl overflow-hidden shadow-2xl", UI.card)}>
                <div className="absolute inset-0 z-0 overflow-hidden pointer-events-none">
                    <Patterns.Polka color="rgba(99, 102, 241, 0.08)" />
                    <div className="absolute top-[-20px] left-[-20px] h-32 w-32 rounded-full border-[6px] border-white/20 bg-blue-500/20 shadow-lg blur-xl" />
                    <div className="absolute bottom-[-10px] right-[-10px] h-24 w-24 rotate-12 border-[6px] border-white/20 bg-purple-500/20 shadow-lg blur-xl" />
                </div>

                <div className="relative z-10 flex flex-col max-h-[85vh]">
                    <div className="flex items-center justify-between border-b 2800 dark:border-zinc-700 p-5 bg-white/50 dark:bg-zinc-900/50 backdrop-blur-sm">
                        <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-100 text-indigo-600 dark:bg-indigo-900/50 dark:text-indigo-400">
                                <ShieldCheck className="h-6 w-6" />
                            </div>
                            <div>
                                <div className={UI.sectionTitle}>Membership</div>
                                <div className="text-xs text-zinc-500 font-medium">Manage user access & permissions</div>
                            </div>
                        </div>
                        <button onClick={onClose} className="rounded-full p-2 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors">
                            <X className="h-5 w-5 text-zinc-500" />
                        </button>
                    </div>

                    <div className="flex-1 overflow-y-auto p-5 space-y-3 custom-scrollbar">
                        {/* Public Access Setting */}
                        <div className={cn("flex items-center justify-between p-4 rounded-xl mb-4", UI.soft)}>
                            <div className="flex items-center gap-3">
                                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-100 text-blue-600 dark:bg-blue-900/50 dark:text-blue-400">
                                    <Globe className="h-6 w-6" />
                                </div>
                                <div>
                                    <div className="font-bold text-zinc-900 dark:text-zinc-100">Public Access</div>
                                    <div className="text-xs text-zinc-500">Allow everyone to view all posts</div>
                                </div>
                            </div>
                            <label className="relative inline-flex cursor-pointer items-center">
                                <input
                                    type="checkbox"
                                    className="peer sr-only"
                                    checked={!!settings.public_access}
                                    onChange={() => updateSettings({ ...settings, public_access: !settings.public_access })}
                                />
                                <div className="peer h-7 w-12 rounded-full bg-zinc-200 dark:bg-zinc-700 after:absolute after:left-[4px] after:top-[4px] after:h-5 after:w-5 after:rounded-full after:border after:border-zinc-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-blue-500 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800"></div>
                            </label>
                        </div>

                        <div className="text-xs font-bold text-zinc-400 uppercase tracking-wider mb-2">Users</div>

                        {loading ? (
                            <div className="flex flex-col items-center justify-center py-12 gap-3 text-zinc-400">
                                <div className="h-8 w-8 animate-spin rounded-full border-4 border-indigo-500 border-t-transparent" />
                                <div className="text-sm font-bold uppercase tracking-wider">Loading...</div>
                            </div>
                        ) : error ? (
                            <div className="rounded-xl bg-red-50 p-6 text-center text-red-600 dark:bg-red-900/20 dark:text-red-400 border-2 border-red-100 dark:border-red-900/50">
                                <div className="font-bold">{error}</div>
                            </div>
                        ) : (
                            users.map(u => (
                                <div key={u.userId} className={cn(
                                    "group flex items-center gap-4 p-3 transition-all duration-200 hover:scale-[1.01]",
                                    UI.soft,
                                    u.isAdmin && "border-indigo-300 dark:border-indigo-700 bg-indigo-50/50 dark:bg-indigo-900/10"
                                )}>
                                    {/* User Visuals */}
                                    <div className="relative shrink-0">
                                        <div className="h-14 w-14 overflow-hidden rounded-full border-2 border-white dark:border-zinc-800 shadow-sm relative z-10">
                                            <img src={getAvatarUrl(u.userId, u.avatar)} alt="" className="h-full w-full object-cover bg-zinc-200" />
                                        </div>
                                        {/* Banner hint */}
                                        {u.banner && (
                                            <div className="absolute -top-1 -left-1 -right-1 h-8 rounded-t-lg overflow-hidden opacity-0 group-hover:opacity-100 transition-opacity z-0 pointer-events-none">
                                                <img src={getBannerUrl(u.userId, u.banner) || ""} className="h-full w-full object-cover blur-[2px]" />
                                            </div>
                                        )}
                                        {u.isAdmin && (
                                            <div className="absolute -bottom-1 -right-1 z-20 flex h-6 w-6 items-center justify-center rounded-full bg-yellow-400 text-yellow-900 shadow-md border-2 border-white dark:border-zinc-800">
                                                <Shield className="h-3 w-3 fill-current" />
                                            </div>
                                        )}
                                    </div>

                                    <div className="flex-1 min-w-0 flex flex-col justify-center">
                                        <div className="flex items-baseline gap-2">
                                            <div className="text-base font-bold text-zinc-900 dark:text-zinc-100 truncate">
                                                {u.globalName || u.username}
                                            </div>
                                            {u.bot && <span className={cn(UI.pill, "bg-blue-100 text-blue-700 border-blue-200 scale-75 origin-left")}>BOT</span>}
                                        </div>
                                        <div className="text-xs font-mono text-zinc-500 truncate">@{u.username} • {u.userId}</div>
                                    </div>

                                    <div className="flex items-center gap-4">
                                        <div className="text-right hidden sm:block">
                                            <div className={UI.label}>Role</div>
                                            <div className={cn(
                                                "text-xs font-bold",
                                                u.isAdmin ? "text-indigo-600 dark:text-indigo-400" : "text-zinc-500"
                                            )}>
                                                {u.isAdmin ? "Admin" : "Member"}
                                            </div>
                                        </div>

                                        <label className="relative inline-flex cursor-pointer items-center">
                                            <input
                                                type="checkbox"
                                                className="peer sr-only"
                                                checked={!!u.isAdmin}
                                                onChange={() => toggleAdmin(u.userId, !!u.isAdmin)}
                                            />
                                            <div className="peer h-7 w-12 rounded-full bg-zinc-200 dark:bg-zinc-700 after:absolute after:left-[4px] after:top-[4px] after:h-5 after:w-5 after:rounded-full after:border after:border-zinc-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-indigo-500 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-indigo-300 dark:peer-focus:ring-indigo-800"></div>
                                        </label>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
